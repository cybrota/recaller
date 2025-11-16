use std::sync::Arc;

use anyhow::{Result, anyhow};

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategies::remove_overstrike;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct AwsHelpStrategy {
    runner: Arc<CommandRunner>,
}

impl AwsHelpStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for AwsHelpStrategy {
    fn priority(&self) -> i32 {
        2
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        base_cmd == "aws"
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        if !command.has_sub_command(1) {
            return self.runner.run("aws", &["help"]);
        }

        let mut args: Vec<&str> = command.sub_cmds().iter().map(|s| s.as_str()).collect();
        args.push("help");
        if let Ok(out) = self.runner.run("aws", &args) {
            return Ok(remove_overstrike(&out));
        }

        Err(anyhow!("AWS command {} is invalid", command.full_name()))
    }
}
