use std::sync::Arc;

use anyhow::Result;

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategies::remove_overstrike;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct NpmHelpStrategy {
    runner: Arc<CommandRunner>,
}

impl NpmHelpStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for NpmHelpStrategy {
    fn priority(&self) -> i32 {
        2
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        base_cmd == "npm"
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        if !command.has_sub_command(1) {
            return self.runner.run("npm", &["help"]);
        }

        let sub_cmd = command.get_sub_command(0).unwrap();
        if let Ok(out) = self.runner.run("npm", &["help", sub_cmd]) {
            return Ok(remove_overstrike(&out));
        }

        self.runner.run("npm", &[sub_cmd, "--help"])
    }
}
