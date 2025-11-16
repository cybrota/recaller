use std::collections::{BTreeMap, HashMap};
use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::PathBuf;

use chrono::{DateTime, Duration, TimeZone, Utc};
use directories::BaseDirs;
use thiserror::Error;

#[derive(Debug, Clone)]
pub struct HistoryEntry {
    pub command: String,
    pub timestamp: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct CommandMetadata {
    pub command: String,
    pub timestamp: Option<DateTime<Utc>>,
    pub frequency: i32,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct RankedCommand {
    pub command: String,
    pub score: f64,
    pub metadata: CommandMetadata,
}

#[derive(Debug, Default, Clone)]
pub struct HistoryIndex {
    commands: BTreeMap<String, CommandMetadata>,
}

impl HistoryIndex {
    pub fn from_entries(entries: Vec<HistoryEntry>) -> Self {
        let capacity = entries.len().max(1);
        let mut freq_map: HashMap<String, i32> = HashMap::with_capacity(capacity / 4 + 1);
        let mut last_timestamp: HashMap<String, DateTime<Utc>> =
            HashMap::with_capacity(capacity / 4 + 1);

        let fallback_base = Utc::now();
        let mut fallback_counter = 0_i64;

        for entry in entries.into_iter().rev() {
            let command = entry.command.trim();
            if command.is_empty() {
                continue;
            }
            let command = command.to_string();

            *freq_map.entry(command.clone()).or_insert(0) += 1;

            if let Some(ts) = entry.timestamp {
                let slot = last_timestamp.entry(command).or_insert(ts);
                if ts > *slot {
                    *slot = ts;
                }
            } else {
                last_timestamp.entry(command).or_insert_with(|| {
                    fallback_counter += 1;
                    fallback_base - Duration::seconds(fallback_counter)
                });
            }
        }

        let mut commands = BTreeMap::new();
        for (command, frequency) in freq_map {
            let timestamp = last_timestamp.get(&command).cloned();
            let metadata = CommandMetadata {
                command: command.clone(),
                timestamp,
                frequency,
            };
            commands.insert(command, metadata);
        }

        HistoryIndex { commands }
    }

    pub fn search(&self, query: &str, enable_fuzzing: bool) -> Vec<RankedCommand> {
        let nodes: Vec<&CommandMetadata> = if enable_fuzzing {
            self.search_fuzzy(query)
        } else {
            self.search_prefix(query)
        };

        let mut ranked: Vec<RankedCommand> = nodes
            .into_iter()
            .map(|meta| RankedCommand {
                command: meta.command.clone(),
                score: calculate_score(meta),
                metadata: meta.clone(),
            })
            .collect();

        ranked.sort_by(|a, b| {
            b.score
                .partial_cmp(&a.score)
                .unwrap_or(std::cmp::Ordering::Equal)
        });
        ranked
    }

    fn search_prefix(&self, prefix: &str) -> Vec<&CommandMetadata> {
        if prefix.is_empty() {
            return self.commands.values().collect();
        }

        use std::ops::Bound::{Excluded, Included};

        let start = Included(prefix.to_string());
        let mut upper = prefix.to_string();
        upper.push(char::MAX);
        let end = Excluded(upper);

        self.commands
            .range((start, end))
            .map(|(_, meta)| meta)
            .collect()
    }

    fn search_fuzzy(&self, query: &str) -> Vec<&CommandMetadata> {
        if query.is_empty() {
            return self.commands.values().collect();
        }
        let query_lower = query.to_lowercase();
        self.commands
            .values()
            .filter(|meta| meta.command.to_lowercase().contains(&query_lower))
            .collect()
    }
}

fn calculate_score(metadata: &CommandMetadata) -> f64 {
    let frequency_score = metadata.frequency as f64;
    let recency_score = match metadata.timestamp {
        Some(ts) => {
            let delta = Utc::now().signed_duration_since(ts).num_seconds();
            let hours = if delta.is_negative() {
                0.0
            } else {
                delta as f64 / 3600.0
            };
            1.0 / (hours + 1.0)
        }
        None => 0.0,
    };

    (0.6 * frequency_score) + (0.4 * recency_score)
}

#[derive(Debug, Clone, Copy)]
enum ShellKind {
    Zsh,
    Bash,
}

#[derive(Debug, Error)]
pub enum HistoryError {
    #[error("failed to determine current shell: {0}")]
    DetectShell(String),
    #[error("{shell} history file not found. {help}")]
    MissingHistoryFile { shell: String, help: String },
    #[error("failed to read history file {path}: {source}")]
    Io {
        path: PathBuf,
        #[source]
        source: std::io::Error,
    },
    #[error("unsupported shell '{0}' detected")]
    UnknownShell(String),
}

pub fn load_history_index() -> Result<HistoryIndex, HistoryError> {
    let shell = detect_shell()?;
    let entries = match shell {
        ShellKind::Zsh => read_zsh_history()?,
        ShellKind::Bash => read_bash_history()?,
    };

    Ok(HistoryIndex::from_entries(entries))
}

pub fn get_suggestions(index: &HistoryIndex, query: &str, enable_fuzzing: bool) -> Vec<String> {
    index
        .search(query, enable_fuzzing)
        .into_iter()
        .map(|ranked| ranked.command)
        .collect()
}

fn detect_shell() -> Result<ShellKind, HistoryError> {
    let shell_path = std::env::var("SHELL").unwrap_or_else(|_| "/bin/bash".to_string());
    let shell_name = PathBuf::from(shell_path)
        .file_name()
        .and_then(|s| s.to_str())
        .unwrap_or("bash")
        .to_string();

    match shell_name.as_str() {
        "zsh" => Ok(ShellKind::Zsh),
        "bash" => Ok(ShellKind::Bash),
        other => Err(HistoryError::UnknownShell(other.to_string())),
    }
}

fn read_zsh_history() -> Result<Vec<HistoryEntry>, HistoryError> {
    let path = history_path(".zsh_history")?;
    let file = File::open(&path).map_err(|err| match err.kind() {
        std::io::ErrorKind::NotFound => HistoryError::MissingHistoryFile {
            shell: "zsh".to_string(),
            help: format!(
                "Run some commands in zsh to create {} and then try again",
                path.display()
            ),
        },
        _ => HistoryError::Io {
            path: path.clone(),
            source: err,
        },
    })?;

    let mut history = Vec::new();
    let reader = BufReader::new(file);
    for line in reader.lines() {
        let line = line.map_err(|err| HistoryError::Io {
            path: path.clone(),
            source: err,
        })?;
        if !line.starts_with(": ") {
            history.push(HistoryEntry {
                command: line,
                timestamp: None,
            });
            continue;
        }

        let parts: Vec<&str> = line.splitn(3, ':').collect();
        if parts.len() < 3 {
            continue;
        }

        let epoch_str = parts[1].trim();
        let epoch = epoch_str.parse::<i64>().ok();
        let timestamp = epoch.and_then(|ts| Utc.timestamp_opt(ts, 0).single());

        let cmd_parts: Vec<&str> = parts[2].splitn(2, ';').collect();
        if cmd_parts.len() < 2 {
            continue;
        }

        history.push(HistoryEntry {
            command: cmd_parts[1].to_string(),
            timestamp,
        });
    }

    Ok(history)
}

fn read_bash_history() -> Result<Vec<HistoryEntry>, HistoryError> {
    let path = history_path(".bash_history")?;
    let file = File::open(&path).map_err(|err| match err.kind() {
        std::io::ErrorKind::NotFound => HistoryError::MissingHistoryFile {
            shell: "bash".to_string(),
            help: format!(
                "Run 'history -w' to create {} and then try again",
                path.display()
            ),
        },
        _ => HistoryError::Io {
            path: path.clone(),
            source: err,
        },
    })?;

    let reader = BufReader::new(file);
    let mut history = Vec::new();
    let mut last_timestamp: Option<DateTime<Utc>> = None;

    for line in reader.lines() {
        let line = line.map_err(|err| HistoryError::Io {
            path: path.clone(),
            source: err,
        })?;

        if let Some(stripped) = line.strip_prefix('#') {
            if let Ok(epoch) = stripped.trim().parse::<i64>() {
                if let Some(ts) = Utc.timestamp_opt(epoch, 0).single() {
                    last_timestamp = Some(ts);
                }
            }
            continue;
        }

        history.push(HistoryEntry {
            command: line,
            timestamp: last_timestamp,
        });
        last_timestamp = None;
    }

    Ok(history)
}

fn history_path(filename: &str) -> Result<PathBuf, HistoryError> {
    let base = BaseDirs::new()
        .ok_or_else(|| HistoryError::DetectShell("Failed to resolve home directory".to_string()))?;
    Ok(base.home_dir().join(filename))
}
