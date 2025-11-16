use anyhow::{Result, anyhow};
use std::sync::Arc;

use crate::config::Config;
use crate::help_system::manager::HelpManager;
use crate::history::{HistoryIndex, load_history_index};

pub struct AppState {
    pub config: Config,
    history_index: Option<HistoryIndex>,
    help_manager: Option<Arc<HelpManager>>,
}

impl AppState {
    pub fn new(config: Config) -> Self {
        Self {
            config,
            history_index: None,
            help_manager: None,
        }
    }

    pub fn history_index(&mut self) -> Result<&HistoryIndex> {
        if self.history_index.is_none() {
            let index = load_history_index().map_err(|err| anyhow!(err))?;
            self.history_index = Some(index);
        }

        Ok(self
            .history_index
            .as_ref()
            .expect("history index initialized"))
    }

    pub fn help_manager(&mut self) -> Arc<HelpManager> {
        if self.help_manager.is_none() {
            self.help_manager = Some(Arc::new(HelpManager::new()));
        }
        self.help_manager
            .as_ref()
            .expect("help manager initialized")
            .clone()
    }
}
