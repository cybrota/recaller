use std::process::Command;
use std::sync::Arc;

use anyhow::{Result, anyhow};

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategies::remove_overstrike;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct ManPageStrategy {
    runner: Arc<CommandRunner>,
}

impl ManPageStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for ManPageStrategy {
    fn priority(&self) -> i32 {
        5
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        if base_cmd.is_empty() {
            return false;
        }
        Command::new("man")
            .arg("-w")
            .arg(base_cmd)
            .output()
            .map(|out| out.status.success())
            .unwrap_or(false)
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        let base = command
            .base_cmd()
            .ok_or_else(|| anyhow!("missing command"))?;
        let output = self.runner.run("man", &[base])?;
        if output.contains("No manual entry") || output.contains("has been minimized") {
            return Err(anyhow!("man page not found for {base}"));
        }
        Ok(remove_overstrike(&output))
    }
}
