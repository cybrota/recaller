use std::sync::Arc;

use anyhow::{Result, anyhow};

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategies::remove_overstrike;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct GitHelpStrategy {
    runner: Arc<CommandRunner>,
}

impl GitHelpStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for GitHelpStrategy {
    fn priority(&self) -> i32 {
        2
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        base_cmd == "git"
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        if !command.has_sub_command(1) {
            return self.runner.run_git("git", &["help"]);
        }

        let sub_cmd = command.get_sub_command(0).unwrap();
        if let Ok(output) =
            self.runner
                .run_with_env("git", &["help", sub_cmd], &[("GIT_PAGER", "cat")])
        {
            return Ok(remove_overstrike(&output));
        }

        if command.has_sub_command(2) {
            let mut args: Vec<&str> = command.sub_cmds().iter().map(|s| s.as_str()).collect();
            args.push("--help");
            if let Ok(output) = self.runner.run_git("git", &args) {
                return Ok(remove_overstrike(&output));
            }
        }

        Err(anyhow!(
            "failed to get Git help for {}",
            command.full_name()
        ))
    }
}
