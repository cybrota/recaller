use std::io::stdout;
use std::time::{Duration, Instant};

use anyhow::Result;
use chrono::{Local, Utc};
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyModifiers};
use crossterm::execute;
use crossterm::terminal::{
    EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode,
};
use ratatui::Terminal;
use ratatui::backend::CrosstermBackend;
use ratatui::layout::{Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::widgets::{Block, Borders, Clear, List, ListItem, ListState, Paragraph, Wrap};

use crate::config::Config;
use crate::fs::{FilesystemIndexer, RankedFile};
use crate::platform::{copy_to_clipboard, open_path};
use crate::state::AppState;

const FILTER_MODES: [&str; 3] = ["All", "Dirs", "Files"];
const FILTER_ICONS: [&str; 3] = ["📁📄", "📁", "📄"];
const STATUS_TIMEOUT: Duration = Duration::from_secs(4);
const HELP_TEXT: &[&str] = &[
    "Filesystem Shortcuts",
    "",
    "General:",
    "  Esc / Ctrl+C  - Exit UI",
    "  Ctrl+H        - Toggle this help window",
    "",
    "Search Pane:",
    "  Typing        - Filter indexed files",
    "  Backspace     - Delete character",
    "  Tab           - Toggle metadata focus",
    "  Up/Down       - Navigate results (or metadata when focused)",
    "",
    "Actions:",
    "  Enter         - Open file/directory",
    "  Ctrl+Y        - Copy selected path",
    "  Ctrl+T        - Cycle filter (All/Dirs/Files)",
];

#[derive(Copy, Clone, Eq, PartialEq)]
enum FilterMode {
    All = 0,
    Dirs = 1,
    Files = 2,
}

pub fn run(state: &mut AppState, indexer: &mut FilesystemIndexer) -> Result<()> {
    enable_raw_mode()?;
    let mut stdout = stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;
    terminal.clear()?;

    let mut ui_state = FilesystemUiState::new(state.config.history.enable_fuzzing);
    ui_state.refresh_results(indexer)?;

    let result = run_loop(&mut terminal, &mut ui_state, &state.config, indexer);

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    result
}

fn run_loop(
    terminal: &mut Terminal<CrosstermBackend<std::io::Stdout>>,
    ui_state: &mut FilesystemUiState,
    state: &Config,
    indexer: &mut FilesystemIndexer,
) -> Result<()> {
    let mut status = String::new();
    let mut status_time = Instant::now();

    loop {
        terminal.draw(|f| {
            let layout = Layout::default()
                .direction(Direction::Vertical)
                .constraints([
                    Constraint::Length(3),
                    Constraint::Min(5),
                    Constraint::Length(3),
                ])
                .split(f.size());

            let input_block = Block::default()
                .borders(Borders::ALL)
                .title("Filesystem Search");
            let input = Paragraph::new(ui_state.input.clone()).block(input_block);
            f.render_widget(input, layout[0]);

            let body = Layout::default()
                .direction(Direction::Horizontal)
                .constraints([Constraint::Percentage(45), Constraint::Percentage(55)])
                .split(layout[1]);

            let title = format!(
                " {} {} ",
                FILTER_ICONS[ui_state.filter_mode as usize],
                FILTER_MODES[ui_state.filter_mode as usize]
            );
            let list_block = Block::default()
                .borders(Borders::ALL)
                .border_style(if !ui_state.focus_metadata {
                    Style::default().fg(Color::Cyan)
                } else {
                    Style::default()
                })
                .title(title);

            let items: Vec<ListItem> = if ui_state.filtered_results.is_empty() {
                vec![ListItem::new("Type to search files and directories...")]
            } else {
                ui_state
                    .filtered_results
                    .iter()
                    .map(|file| ListItem::new(format_file_entry(file)))
                    .collect()
            };

            let list = List::new(items)
                .block(list_block)
                .highlight_style(Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD))
                .highlight_symbol("▶ ");
            f.render_stateful_widget(list, body[0], &mut ui_state.list_state);

            let meta_block = Block::default()
                .borders(Borders::ALL)
                .border_style(if ui_state.focus_metadata {
                    Style::default().fg(Color::Cyan)
                } else {
                    Style::default()
                })
                .title("Metadata");
            let meta_text = Paragraph::new(ui_state.metadata_text())
                .block(meta_block)
                .wrap(Wrap { trim: true });
            f.render_widget(meta_text, body[1]);

            if status_time.elapsed() > STATUS_TIMEOUT {
                status.clear();
            }
            let footer = Paragraph::new(if status.is_empty() {
                "Enter: open  Ctrl+Y: copy path  Ctrl+T: toggle filter  Ctrl+H: help  Tab: focus metadata  Esc: quit".into()
            } else {
                status.clone()
            })
            .wrap(Wrap { trim: true })
            .block(Block::default().borders(Borders::ALL));
            f.render_widget(footer, layout[2]);

            if ui_state.show_help {
                let area = centered_rect(70, 70, f.size());
                f.render_widget(Clear, area);
                let help_text = HELP_TEXT.join("\n");
                let help = Paragraph::new(help_text)
                    .block(
                        Block::default()
                            .borders(Borders::ALL)
                            .title("Filesystem Help")
                            .border_style(Style::default().fg(Color::Magenta)),
                    )
                    .wrap(Wrap { trim: true })
                    .scroll((ui_state.help_scroll, 0));
                f.render_widget(help, area);
            }
        })?;

        if event::poll(Duration::from_millis(50))? {
            match event::read()? {
                Event::Key(key) => {
                    if handle_key(key, ui_state, indexer, &mut status, &mut status_time, state)? {
                        return Ok(());
                    }
                }
                Event::Resize(_, _) => {}
                _ => {}
            }
        }
    }
}

fn handle_key(
    key: KeyEvent,
    state: &mut FilesystemUiState,
    indexer: &mut FilesystemIndexer,
    status: &mut String,
    status_time: &mut Instant,
    config: &Config,
) -> Result<bool> {
    if state.show_help {
        match key.code {
            KeyCode::Esc => state.hide_help_modal(),
            KeyCode::Char('h') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                state.hide_help_modal()
            }
            KeyCode::Up => state.scroll_help_up(),
            KeyCode::Down => state.scroll_help_down(),
            _ => {}
        }
        return Ok(false);
    }

    match key.code {
        KeyCode::Esc => return Ok(true),
        KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => return Ok(true),
        KeyCode::Char('h') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            state.show_help_modal();
        }
        KeyCode::Char(ch) if key.modifiers.is_empty() => {
            if !state.focus_metadata {
                state.input.push(ch);
                state.refresh_results(indexer)?;
            }
        }
        KeyCode::Backspace => {
            if !state.focus_metadata {
                state.input.pop();
                state.refresh_results(indexer)?;
            }
        }
        KeyCode::Tab => state.focus_metadata = !state.focus_metadata,
        KeyCode::Up => {
            if state.focus_metadata {
                state.scroll_metadata_up();
            } else {
                state.move_selection_up();
            }
        }
        KeyCode::Down => {
            if state.focus_metadata {
                state.scroll_metadata_down();
            } else {
                state.move_selection_down();
            }
        }
        KeyCode::Enter => {
            if let Some(file) = state.current_file() {
                open_path(&file.path)?;
                indexer.add_path(&file.path, Some(Utc::now()), true);
                indexer.persist_index(!config.quiet)?;
                println!("\n🚀 Opened: {}", file.path);
                return Ok(true);
            }
        }
        KeyCode::Char('y') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            if let Some(file) = state.current_file() {
                copy_to_clipboard(&file.path)?;
                *status = format!("📋 Copied path: {}", file.path);
                *status_time = Instant::now();
            }
        }
        KeyCode::Char('t') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            state.cycle_filter();
            state.refresh_results(indexer)?;
        }
        _ => {}
    }
    Ok(false)
}

struct FilesystemUiState {
    input: String,
    results: Vec<RankedFile>,
    filtered_results: Vec<RankedFile>,
    enable_fuzzy: bool,
    selected: usize,
    focus_metadata: bool,
    filter_mode: FilterMode,
    list_state: ListState,
    metadata_scroll: usize,
    show_help: bool,
    help_scroll: u16,
}

impl FilesystemUiState {
    fn new(enable_fuzzy: bool) -> Self {
        let mut list_state = ListState::default();
        list_state.select(Some(0));
        Self {
            input: String::new(),
            results: Vec::new(),
            filtered_results: Vec::new(),
            enable_fuzzy,
            selected: 0,
            focus_metadata: false,
            filter_mode: FilterMode::All,
            list_state,
            metadata_scroll: 0,
            show_help: false,
            help_scroll: 0,
        }
    }

    fn refresh_results(&mut self, indexer: &FilesystemIndexer) -> Result<()> {
        self.results = indexer.search_files(&self.input, self.enable_fuzzy);
        self.filtered_results = self
            .results
            .iter()
            .filter(|file| match self.filter_mode {
                FilterMode::All => true,
                FilterMode::Dirs => file.metadata.is_directory,
                FilterMode::Files => !file.metadata.is_directory,
            })
            .cloned()
            .collect();

        if self.filtered_results.is_empty() {
            self.selected = 0;
            self.list_state.select(None);
        } else {
            if self.selected >= self.filtered_results.len() {
                self.selected = self.filtered_results.len() - 1;
            }
            self.list_state.select(Some(self.selected));
        }
        self.metadata_scroll = 0;
        Ok(())
    }

    fn current_file(&self) -> Option<&RankedFile> {
        self.filtered_results.get(self.selected)
    }

    fn move_selection_up(&mut self) {
        if self.selected > 0 {
            self.selected -= 1;
            self.list_state.select(Some(self.selected));
            self.metadata_scroll = 0;
        }
    }

    fn move_selection_down(&mut self) {
        if self.selected + 1 < self.filtered_results.len() {
            self.selected += 1;
            self.list_state.select(Some(self.selected));
            self.metadata_scroll = 0;
        }
    }

    fn scroll_metadata_up(&mut self) {
        if self.metadata_scroll > 0 {
            self.metadata_scroll -= 1;
        }
    }

    fn scroll_metadata_down(&mut self) {
        self.metadata_scroll = self.metadata_scroll.saturating_add(1);
    }

    fn metadata_text(&self) -> String {
        if let Some(file) = self.current_file() {
            let mut lines = Vec::new();
            lines.push(format!("📍 Path: {}", file.path));
            lines.push(format!(
                "📁 Type: {}",
                if file.metadata.is_directory {
                    "Directory"
                } else {
                    "File"
                }
            ));
            lines.push(format!(
                "🕒 Last Accessed: {}",
                file.metadata
                    .timestamp
                    .map(|ts| ts
                        .with_timezone(&Local)
                        .format("%Y-%m-%d %H:%M:%S")
                        .to_string())
                    .unwrap_or_else(|| "Unknown".into())
            ));
            lines.push(format!("📊 Access Count: {}", file.metadata.access_count));
            lines.push(format!("⭐ Score: {:.2}", file.score));
            if let Some(size) = file.metadata.size {
                lines.push(format!("💾 Size: {}", human_size(size)));
            }
            if let Some(modified) = file.metadata.last_modified {
                lines.push(format!(
                    "✏️  Modified: {}",
                    modified.format("%Y-%m-%d %H:%M:%S")
                ));
            }
            if file.metadata.is_hidden {
                lines.push("🔒 Hidden file".into());
            }
            if file.metadata.is_symlink {
                lines.push("🔗 Symbolic link".into());
            }

            let max_scroll = lines.len().saturating_sub(1);
            let start = self.metadata_scroll.min(max_scroll);
            lines[start..].join("\n")
        } else {
            "Select a file to view details".into()
        }
    }

    fn cycle_filter(&mut self) {
        self.filter_mode = match self.filter_mode {
            FilterMode::All => FilterMode::Dirs,
            FilterMode::Dirs => FilterMode::Files,
            FilterMode::Files => FilterMode::All,
        };
        self.selected = 0;
        self.metadata_scroll = 0;
    }

    fn scroll_help_up(&mut self) {
        self.help_scroll = self.help_scroll.saturating_sub(1);
    }

    fn scroll_help_down(&mut self) {
        let max_scroll = HELP_TEXT.len().saturating_sub(1) as u16;
        if self.help_scroll < max_scroll {
            self.help_scroll += 1;
        }
    }

    fn show_help_modal(&mut self) {
        self.show_help = true;
        self.help_scroll = 0;
    }

    fn hide_help_modal(&mut self) {
        self.show_help = false;
        self.help_scroll = 0;
    }
}

fn format_file_entry(file: &RankedFile) -> String {
    let icon = if file.metadata.is_directory {
        "📁"
    } else {
        "📄"
    };
    let path = &file.path;
    let truncated = if path.len() > 70 {
        format!("{}…", &path[..67])
    } else {
        path.to_string()
    };
    format!("{} {}", icon, truncated)
}

fn human_size(size: u64) -> String {
    const UNITS: [&str; 5] = ["B", "KB", "MB", "GB", "TB"];
    let mut value = size as f64;
    let mut idx = 0;
    while value >= 1024.0 && idx < UNITS.len() - 1 {
        value /= 1024.0;
        idx += 1;
    }
    format!("{:.1} {}", value, UNITS[idx])
}

fn centered_rect(percent_x: u16, percent_y: u16, r: Rect) -> Rect {
    let vertical = Layout::default()
        .direction(Direction::Vertical)
        .constraints(&[
            Constraint::Percentage((100 - percent_y) / 2),
            Constraint::Percentage(percent_y),
            Constraint::Percentage((100 - percent_y) / 2),
        ])
        .split(r);

    let horizontal = Layout::default()
        .direction(Direction::Horizontal)
        .constraints(&[
            Constraint::Percentage((100 - percent_x) / 2),
            Constraint::Percentage(percent_x),
            Constraint::Percentage((100 - percent_x) / 2),
        ])
        .split(vertical[1]);

    horizontal[1]
}
