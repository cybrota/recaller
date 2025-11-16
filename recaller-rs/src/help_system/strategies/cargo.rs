use std::sync::Arc;

use anyhow::Result;

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct CargoHelpStrategy {
    runner: Arc<CommandRunner>,
}

impl CargoHelpStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for CargoHelpStrategy {
    fn priority(&self) -> i32 {
        2
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        base_cmd == "cargo"
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        if !command.has_sub_command(1) {
            return self.runner.run("cargo", &["--help"]);
        }

        let sub_cmd = command.get_sub_command(0).unwrap();
        self.runner.run("cargo", &[sub_cmd, "--help"])
    }
}
