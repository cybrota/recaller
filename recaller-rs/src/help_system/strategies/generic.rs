use std::sync::Arc;

use anyhow::{Result, anyhow};

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct GenericHelpStrategy {
    runner: Arc<CommandRunner>,
}

impl GenericHelpStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for GenericHelpStrategy {
    fn priority(&self) -> i32 {
        8
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        self.runner.command_exists(base_cmd)
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        let base = command
            .base_cmd()
            .ok_or_else(|| anyhow!("missing command"))?;
        let args: Vec<&str> = command.sub_cmds().iter().map(|s| s.as_str()).collect();
        let help_flags = ["-h", "--help", "help"];

        for flag in &help_flags {
            let mut attempt = args.clone();
            attempt.push(flag);
            if let Ok(out) = self.runner.run(base, &attempt) {
                if !out.trim().is_empty() {
                    return Ok(out);
                }
            }
        }

        Err(anyhow!("no help found for {}", command.full_name()))
    }
}
