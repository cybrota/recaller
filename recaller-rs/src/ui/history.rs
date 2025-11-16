use std::io::stdout;
use std::sync::mpsc;
use std::time::{Duration, Instant};

use anyhow::Result;
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyModifiers};
use crossterm::execute;
use crossterm::terminal::{
    EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode,
};
use ratatui::Terminal;
use ratatui::backend::CrosstermBackend;
use ratatui::layout::{Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span, Text};
use ratatui::widgets::{Block, Borders, Clear, List, ListItem, ListState, Paragraph, Wrap};

use crate::help_system::parser::split_command;
use crate::history::RankedCommand;
use crate::platform::{copy_to_clipboard, send_to_terminal};
use crate::state::AppState;

const SEARCH_DEBOUNCE: Duration = Duration::from_millis(120);
const HELP_TEXT: &[&str] = &[
    "Shortcut Reference",
    "",
    "General:",
    "  Esc / Ctrl+C  - Exit the UI",
    "  Ctrl+H        - Toggle this help window",
    "",
    "Search Pane:",
    "  Typing        - Filter history",
    "  Backspace     - Delete character",
    "  Up/Down       - Navigate suggestions",
    "  Home/End      - Jump to start/end",
    "  Ctrl+K/Ctrl+J - Jump to first/last result",
    "",
    "Actions:",
    "  Enter         - Print command and quit",
    "  Ctrl+E        - Send command to terminal",
    "  Ctrl+Y        - Copy command to clipboard",
];

pub fn run(state: &mut AppState) -> Result<()> {
    enable_raw_mode()?;
    let mut stdout = stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;
    terminal.clear()?;

    let result = run_loop(state, &mut terminal);

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    result
}

fn run_loop(
    state: &mut AppState,
    terminal: &mut Terminal<CrosstermBackend<std::io::Stdout>>,
) -> Result<()> {
    let mut ui_state = HistoryUiState::new(state.config.history.enable_fuzzing);
    ui_state.refresh_results(state)?;

    let help_manager = state.help_manager();
    let (help_tx, help_rx) = mpsc::channel::<String>();
    let (help_resp_tx, help_resp_rx) = mpsc::channel::<(String, String)>();

    std::thread::spawn(move || {
        while let Ok(cmd) = help_rx.recv() {
            let response = match split_command(&cmd)
                .map_err(|err| err.into())
                .and_then(|parts| help_manager.get_help(&parts))
            {
                Ok(help) => help,
                Err(err) => format!("Relax and take a deep breath.\n{err}"),
            };
            let _ = help_resp_tx.send((cmd, response));
        }
    });

    let mut status = String::new();
    let mut status_time = Instant::now();
    if let Some(cmd) = ui_state.current_command() {
        let _ = help_tx.send(cmd.to_string());
        ui_state.pending_help = Some(cmd.to_string());
    }

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

            let input = Paragraph::new(ui_state.input.clone())
                .block(Block::default().borders(Borders::ALL).title("Search"))
                .style(Style::default().fg(Color::Cyan));
            f.render_widget(input, layout[0]);

            let body = Layout::default()
                .direction(Direction::Horizontal)
                .constraints([Constraint::Percentage(45), Constraint::Percentage(55)])
                .split(layout[1]);

            let items: Vec<ListItem> = if ui_state.results.is_empty() {
                vec![ListItem::new("Type to search your history...")]
            } else {
                ui_state
                    .results
                    .iter()
                    .map(|cmd| ListItem::new(cmd.command.clone()))
                    .collect()
            };

            let suggestions = List::new(items)
                .block(Block::default().borders(Borders::ALL).title("History"))
                .highlight_style(
                    Style::default()
                        .fg(Color::Yellow)
                        .add_modifier(Modifier::BOLD),
                )
                .highlight_symbol("▶ ");
            f.render_stateful_widget(suggestions, body[0], &mut ui_state.list_state);

            let help_block = Block::default().borders(Borders::ALL).title("Help");
            let help = Paragraph::new(ui_state.help_text_widget())
                .block(help_block)
                .wrap(Wrap { trim: true });
            f.render_widget(help, body[1]);

            if status_time.elapsed() > Duration::from_secs(4) {
                status.clear();
            }
            let footer = Paragraph::new(if status.is_empty() {
                "Enter: print  Ctrl+E: send to terminal  Ctrl+Y: copy  /: help search  Ctrl+H: help  Esc: quit"
                    .to_string()
            } else {
                status.clone()
            })
            .wrap(Wrap { trim: true })
            .block(Block::default().borders(Borders::ALL).title("Shortcuts"));
            f.render_widget(footer, layout[2]);

            if ui_state.show_help_modal {
                let area = centered_rect(70, 70, f.size());
                f.render_widget(Clear, area);
                let help_text = HELP_TEXT.join("\n");
                let help = Paragraph::new(help_text)
                    .block(
                        Block::default()
                            .borders(Borders::ALL)
                            .title("Help")
                            .border_style(Style::default().fg(Color::Magenta)),
                    )
                    .wrap(Wrap { trim: true })
                    .scroll((ui_state.help_modal_scroll, 0));
                f.render_widget(help, area);
            }

            if ui_state.show_help_search {
                let area = centered_rect(80, 70, f.size());
                f.render_widget(Clear, area);
                let chunks = Layout::default()
                    .direction(Direction::Vertical)
                    .constraints([Constraint::Length(3), Constraint::Min(3)])
                    .split(area);

                let input = Paragraph::new(format!("Search: {}", ui_state.help_search_query))
                    .block(
                        Block::default()
                            .borders(Borders::ALL)
                            .title("Help Search"),
                    );
                f.render_widget(input, chunks[0]);

                let search_text = Paragraph::new(ui_state.help_search_text_widget())
                    .block(
                        Block::default()
                            .borders(Borders::ALL)
                            .title("Help Matches"),
                    )
                    .wrap(Wrap { trim: true })
                    .scroll((ui_state.help_search_scroll, 0));
                f.render_widget(search_text, chunks[1]);
            }
        })?;

        while let Ok((cmd, text)) = help_resp_rx.try_recv() {
            if ui_state
                .current_command()
                .map(|c| c == cmd)
                .unwrap_or(false)
            {
                ui_state.set_help_text(text);
                ui_state.pending_help = None;
            }
        }

        if ui_state.should_refresh() {
            ui_state.refresh_results(state)?;
            if let Some(cmd) = ui_state.current_command() {
                if ui_state.pending_help.as_deref() != Some(cmd) {
                    let _ = help_tx.send(cmd.to_string());
                    ui_state.pending_help = Some(cmd.to_string());
                }
            }
        }

        if event::poll(Duration::from_millis(50))? {
            match event::read()? {
                Event::Key(key) => {
                    if handle_key_event(key, &mut ui_state, &mut status, &mut status_time)? {
                        return Ok(());
                    }
                }
                Event::Resize(_, _) => {}
                _ => {}
            }
        }
    }
}

fn handle_key_event(
    key: KeyEvent,
    state: &mut HistoryUiState,
    status: &mut String,
    status_time: &mut Instant,
) -> Result<bool> {
    if state.show_help_modal {
        match key.code {
            KeyCode::Esc => state.close_help_modal(),
            KeyCode::Char('h') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                state.close_help_modal()
            }
            KeyCode::Up => state.scroll_help_modal_up(),
            KeyCode::Down => state.scroll_help_modal_down(),
            _ => {}
        }
        return Ok(false);
    }

    if state.show_help_search {
        match key.code {
            KeyCode::Esc => {
                state.close_help_search();
                *status = "Help search cancelled".into();
                *status_time = Instant::now();
            }
            KeyCode::Enter => {
                state.update_help_search_matches();
                *status = format!("Searching: {}", state.help_search_query);
                *status_time = Instant::now();
            }
            KeyCode::Backspace => {
                state.pop_help_search_char();
            }
            KeyCode::Char(ch) => {
                state.push_help_search_char(ch);
            }
            KeyCode::Up => {
                state.help_search_scroll = state.help_search_scroll.saturating_sub(1);
                state.help_search_manual_scroll = true;
            }
            KeyCode::Down => {
                state.help_search_scroll = state.help_search_scroll.saturating_add(1);
                state.help_search_manual_scroll = true;
            }
            _ => {}
        }
        return Ok(false);
    }

    match key.code {
        KeyCode::Esc => return Ok(true),
        KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => return Ok(true),
        KeyCode::Char('/') => {
            state.open_help_search();
            *status = "Help search: ".into();
            *status_time = Instant::now();
            return Ok(false);
        }
        KeyCode::Char('h') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            state.open_help_modal();
        }
        KeyCode::Char(ch) => {
            if key.modifiers.is_empty() {
                state.input.push(ch);
                state.mark_dirty();
            } else if key.modifiers.contains(KeyModifiers::CONTROL) {
                match ch {
                    'y' => {
                        if let Some(cmd) = state.current_command() {
                            copy_to_clipboard(cmd)?;
                            *status = format!("📋 Copied: {}", cmd);
                            *status_time = Instant::now();
                        }
                    }
                    'e' => {
                        if let Some(cmd) = state.current_command() {
                            send_to_terminal(cmd)?;
                            *status = format!("🚀 Sent to terminal: {}", cmd);
                            *status_time = Instant::now();
                        }
                    }
                    _ => {}
                }
            }
        }
        KeyCode::Backspace => {
            state.input.pop();
            state.mark_dirty();
        }
        KeyCode::Enter => {
            if let Some(cmd) = state.current_command() {
                println!(
                    "\n{}
",
                    cmd
                );
                return Ok(true);
            }
        }
        KeyCode::Up => state.move_selection_up(),
        KeyCode::Down => state.move_selection_down(),
        KeyCode::Home => state.select_first(),
        KeyCode::End => state.select_last(),
        _ => {}
    }
    Ok(false)
}

struct HistoryUiState {
    input: String,
    results: Vec<RankedCommand>,
    enable_fuzzing: bool,
    selected: usize,
    pending_help: Option<String>,
    last_search: Instant,
    list_state: ListState,
    help_lines: Vec<String>,
    show_help_modal: bool,
    help_modal_scroll: u16,
    show_help_search: bool,
    help_search_query: String,
    help_search_scroll: u16,
    help_search_highlights: Vec<(usize, usize, usize)>,
    help_search_manual_scroll: bool,
}

impl HistoryUiState {
    fn new(enable_fuzzing: bool) -> Self {
        let mut list_state = ListState::default();
        list_state.select(Some(0));
        Self {
            input: String::new(),
            results: Vec::new(),
            enable_fuzzing,
            selected: 0,
            pending_help: None,
            last_search: Instant::now(),
            list_state,
            help_lines: vec!["Select a command to load help".into()],
            show_help_modal: false,
            help_modal_scroll: 0,
            show_help_search: false,
            help_search_query: String::new(),
            help_search_scroll: 0,
            help_search_highlights: Vec::new(),
            help_search_manual_scroll: false,
        }
    }

    fn refresh_results(&mut self, state: &mut AppState) -> Result<()> {
        let index = state.history_index()?;
        self.results = index.search(&self.input, self.enable_fuzzing);
        if self.results.is_empty() {
            self.selected = 0;
            self.list_state.select(None);
        } else {
            if self.selected >= self.results.len() {
                self.selected = self.results.len() - 1;
            }
            self.list_state.select(Some(self.selected));
        }
        self.last_search = Instant::now();
        Ok(())
    }

    fn set_help_text(&mut self, text: String) {
        self.help_lines = if text.is_empty() {
            vec![String::new()]
        } else {
            text.lines().map(|s| s.to_string()).collect()
        };
        if !self.help_search_query.is_empty() {
            self.update_help_search_matches();
        }
    }

    fn mark_dirty(&mut self) {
        self.last_search = Instant::now() - SEARCH_DEBOUNCE - Duration::from_millis(1);
    }

    fn should_refresh(&self) -> bool {
        self.last_search.elapsed() >= SEARCH_DEBOUNCE
    }

    fn current_command(&self) -> Option<&str> {
        self.results
            .get(self.selected)
            .map(|cmd| cmd.command.as_str())
    }

    fn move_selection_up(&mut self) {
        if self.selected > 0 {
            self.selected -= 1;
            self.list_state.select(Some(self.selected));
        }
    }

    fn move_selection_down(&mut self) {
        if self.selected + 1 < self.results.len() {
            self.selected += 1;
            self.list_state.select(Some(self.selected));
        }
    }

    fn select_first(&mut self) {
        if !self.results.is_empty() {
            self.selected = 0;
            self.list_state.select(Some(self.selected));
        }
    }

    fn select_last(&mut self) {
        if !self.results.is_empty() {
            self.selected = self.results.len() - 1;
            self.list_state.select(Some(self.selected));
        }
    }

    fn open_help_search(&mut self) {
        self.show_help_search = true;
        self.help_search_manual_scroll = false;
        self.update_help_search_matches();
    }

    fn close_help_search(&mut self) {
        self.show_help_search = false;
        self.help_search_scroll = 0;
        self.help_search_manual_scroll = false;
    }

    fn push_help_search_char(&mut self, ch: char) {
        self.help_search_query.push(ch);
        self.help_search_manual_scroll = false;
        self.update_help_search_matches();
    }

    fn pop_help_search_char(&mut self) {
        self.help_search_query.pop();
        self.help_search_manual_scroll = false;
        self.update_help_search_matches();
    }

    fn update_help_search_matches(&mut self) {
        self.help_search_highlights.clear();
        if self.help_search_query.is_empty() {
            self.help_search_scroll = 0;
            return;
        }

        let needle = self.help_search_query.to_lowercase();
        let mut first_line = None;

        for (line_idx, line) in self.help_lines.iter().enumerate() {
            let line_lower = line.to_lowercase();
            if line_lower.is_empty() {
                continue;
            }

            let mut start_idx = 0;
            while let Some(pos) = line_lower[start_idx..].find(&needle) {
                let absolute = start_idx + pos;
                let end_char = absolute + needle.len();
                self.help_search_highlights
                    .push((line_idx, absolute, end_char.min(line.len())));
                if first_line.is_none() {
                    first_line = Some(line_idx as u16);
                }
                start_idx = absolute + 1;
                if start_idx >= line_lower.len() {
                    break;
                }
            }
        }

        if !self.help_search_manual_scroll {
            if let Some(line) = first_line {
                self.help_search_scroll = line;
            } else {
                self.help_search_scroll = 0;
            }
        }
    }

    fn help_text_widget(&self) -> Text<'static> {
        let lines: Vec<Line> = self
            .help_lines
            .iter()
            .map(|line| {
                if line.trim_start().starts_with('$') {
                    Line::styled(line.clone(), Style::default().fg(Color::Yellow))
                } else {
                    Line::raw(line.clone())
                }
            })
            .collect();
        Text::from(lines)
    }

    fn help_search_text_widget(&self) -> Text<'static> {
        if self.help_lines.is_empty() {
            return Text::raw("");
        }

        if self.help_search_query.is_empty() {
            return Text::raw(self.help_lines.join("\n"));
        }

        let mut lines = Vec::new();
        for (idx, line) in self.help_lines.iter().enumerate() {
            let mut spans = Vec::new();
            let mut cursor = 0;
            let mut highlights: Vec<(usize, usize)> = self
                .help_search_highlights
                .iter()
                .filter(|(line_idx, _, _)| *line_idx == idx)
                .map(|(_, start, end)| (*start, *end))
                .collect();
            highlights.sort_by_key(|&(start, _)| start);

            for (start, end) in highlights {
                if start > cursor {
                    spans.push(Span::raw(line[cursor..start].to_string()));
                }
                let highlight_end = end.min(line.len());
                spans.push(Span::styled(
                    line[start..highlight_end].to_string(),
                    Style::default()
                        .fg(Color::Black)
                        .bg(Color::Yellow)
                        .add_modifier(Modifier::BOLD),
                ));
                cursor = highlight_end;
            }

            if cursor < line.len() {
                spans.push(Span::raw(line[cursor..].to_string()));
            }
            if spans.is_empty() {
                spans.push(Span::raw(line.clone()));
            }
            lines.push(Line::from(spans));
        }

        Text::from(lines)
    }

    fn open_help_modal(&mut self) {
        self.show_help_modal = true;
        self.help_modal_scroll = 0;
    }

    fn close_help_modal(&mut self) {
        self.show_help_modal = false;
        self.help_modal_scroll = 0;
    }

    fn scroll_help_modal_up(&mut self) {
        self.help_modal_scroll = self.help_modal_scroll.saturating_sub(1);
    }

    fn scroll_help_modal_down(&mut self) {
        let max_scroll = HELP_TEXT.len().saturating_sub(1) as u16;
        if self.help_modal_scroll < max_scroll {
            self.help_modal_scroll += 1;
        }
    }
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
