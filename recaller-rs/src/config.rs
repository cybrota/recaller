use std::fs;
use std::path::PathBuf;

use anyhow::{Context, Result};
use directories::BaseDirs;
use serde::{Deserialize, Serialize};

use crate::constants::{GREEN, RESET};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HistoryConfig {
    #[serde(default = "default_enable_fuzzing")]
    pub enable_fuzzing: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FilesystemConfig {
    #[serde(default = "default_fs_enabled")]
    pub enabled: bool,
    #[serde(default = "default_fs_dirs")]
    pub index_directories: Vec<String>,
    #[serde(default = "default_fs_ignore_patterns")]
    pub ignore_patterns: Vec<String>,
    #[serde(default = "default_max_indexed_files")]
    pub max_indexed_files: usize,
    #[serde(default = "default_bloom_filter_size")]
    pub bloom_filter_size: u32,
    #[serde(default = "default_bloom_filter_hashes")]
    pub bloom_filter_hashes: u32,
    #[serde(default = "default_sketch_width")]
    pub sketch_width: usize,
    #[serde(default = "default_sketch_depth")]
    pub sketch_depth: usize,
    #[serde(default = "default_auto_index")]
    pub auto_index_on_startup: bool,
    #[serde(default = "default_index_cache_duration")]
    pub index_cache_duration_hours: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(default)]
    pub history: HistoryConfig,
    #[serde(default)]
    pub filesystem: FilesystemConfig,
    #[serde(default)]
    pub quiet: bool,
}

impl Default for HistoryConfig {
    fn default() -> Self {
        Self {
            enable_fuzzing: default_enable_fuzzing(),
        }
    }
}

impl Default for FilesystemConfig {
    fn default() -> Self {
        Self {
            enabled: default_fs_enabled(),
            index_directories: default_fs_dirs(),
            ignore_patterns: default_fs_ignore_patterns(),
            max_indexed_files: default_max_indexed_files(),
            bloom_filter_size: default_bloom_filter_size(),
            bloom_filter_hashes: default_bloom_filter_hashes(),
            sketch_width: default_sketch_width(),
            sketch_depth: default_sketch_depth(),
            auto_index_on_startup: default_auto_index(),
            index_cache_duration_hours: default_index_cache_duration(),
        }
    }
}

impl Default for Config {
    fn default() -> Self {
        Self {
            history: HistoryConfig::default(),
            filesystem: FilesystemConfig::default(),
            quiet: false,
        }
    }
}

const fn default_enable_fuzzing() -> bool {
    true
}

const fn default_fs_enabled() -> bool {
    false
}

fn default_fs_dirs() -> Vec<String> {
    vec![
        ".".to_string(),
        "~/Documents".to_string(),
        "~/Projects".to_string(),
    ]
}

fn default_fs_ignore_patterns() -> Vec<String> {
    vec![
        "node_modules".to_string(),
        ".git".to_string(),
        "*.tmp".to_string(),
        "*.log".to_string(),
        ".DS_Store".to_string(),
        "target".to_string(),
        "build".to_string(),
        "dist".to_string(),
    ]
}

const fn default_max_indexed_files() -> usize {
    50_000
}

const fn default_bloom_filter_size() -> u32 {
    1_000_000
}

const fn default_bloom_filter_hashes() -> u32 {
    7
}

const fn default_sketch_width() -> usize {
    2_048
}

const fn default_sketch_depth() -> usize {
    4
}

const fn default_auto_index() -> bool {
    false
}

const fn default_index_cache_duration() -> u32 {
    24
}

pub fn load_config() -> Result<Config> {
    let (cfg, _) = load_config_with_status()?;
    Ok(cfg)
}

pub fn load_config_with_status() -> Result<(Config, bool)> {
    let path = config_path()?;
    if !path.exists() {
        return Ok((Config::default(), false));
    }

    let data = fs::read_to_string(&path)
        .with_context(|| format!("failed to read config file at {}", path.display()))?;
    let cfg: Config = serde_yaml::from_str(&data)
        .with_context(|| "failed to parse configuration from YAML".to_string())?;
    Ok((cfg, true))
}

pub fn config_path() -> Result<PathBuf> {
    let base = BaseDirs::new().context("failed to determine home directory")?;
    Ok(base.home_dir().join(".recaller.yaml"))
}

pub fn create_default_config_file() -> Result<PathBuf> {
    let path = config_path()?;
    let cfg = Config::default();
    let yaml = serde_yaml::to_string(&cfg)?;
    fs::write(&path, yaml)
        .with_context(|| format!("failed to write default config to {}", path.display()))?;
    Ok(path)
}

pub fn display_settings() -> Result<()> {
    let path = config_path()?;
    let (mut config, existed) = load_config_with_status()?;

    if config.filesystem.index_directories.is_empty() {
        config.filesystem = FilesystemConfig::default();
    }

    let config_existed = existed && path.exists();

    if !config_existed {
        println!("📝 Configuration file not found. Creating default configuration...\n");
        let created_path = create_default_config_file()?;
        println!(
            "✅ Created default configuration at: {}\n",
            created_path.display()
        );
    }

    println!("🔧 Recaller Configuration Settings");
    println!("═══════════════════════════════════\n");

    if config_existed {
        println!("📍 Config file: {}", path.display());
    } else {
        println!("📍 Config file: {} (newly created)", path.display());
    }

    println!("Current settings:\n");

    println!("🔘 {green}Verbosity:{reset}", green = GREEN, reset = RESET);
    println!(
        "  • {green}quiet{reset}: {}\n",
        config.quiet,
        green = GREEN,
        reset = RESET
    );

    println!(
        "🔍 {green}History Search:{reset}",
        green = GREEN,
        reset = RESET
    );
    let fuzzy_desc = if config.history.enable_fuzzing {
        "Fuzzy search (substring matching anywhere)"
    } else {
        "Prefix-based search (commands starting with query)"
    };
    println!(
        "  • {green}enable_fuzzing{reset}: {}",
        config.history.enable_fuzzing,
        green = GREEN,
        reset = RESET
    );
    println!("    {}\n", fuzzy_desc);

    println!(
        "📁 {green}Filesystem Search:{reset}",
        green = GREEN,
        reset = RESET
    );
    let fs_desc = if config.filesystem.enabled {
        format!(
            "Enabled - indexing up to {} files",
            config.filesystem.max_indexed_files
        )
    } else {
        "Disabled - filesystem indexing is off".to_string()
    };
    println!(
        "  • {green}enabled{reset}: {}",
        config.filesystem.enabled,
        green = GREEN,
        reset = RESET
    );
    println!("    {}", fs_desc);
    println!(
        "  • {green}index_directories{reset}: {:?}",
        config.filesystem.index_directories,
        green = GREEN,
        reset = RESET
    );
    println!(
        "  • {green}max_indexed_files{reset}: {}",
        config.filesystem.max_indexed_files,
        green = GREEN,
        reset = RESET
    );
    println!(
        "  • {green}auto_index_on_startup{reset}: {}\n",
        config.filesystem.auto_index_on_startup,
        green = GREEN,
        reset = RESET
    );

    if !config.history.enable_fuzzing {
        println!(
            "💡 Fuzzy search is disabled. To enable it, edit {}:",
            path.display()
        );
        println!("   history:\n     enable_fuzzing: true\n");
    } else {
        println!("💡 To use prefix-only search, edit {}:", path.display());
        println!("   history:\n     enable_fuzzing: false\n");
    }

    if !config.filesystem.enabled {
        println!("💡 To enable filesystem search, edit {}:", path.display());
        println!("   filesystem:\n     enabled: true\n");
    }

    println!("📚 For more information, see: https://github.com/cybrota/recaller#search-modes");
    Ok(())
}

pub fn print_config_error(err: &anyhow::Error) {
    eprintln!("❌ Failed to load configuration: {err}");
}
