use std::sync::Arc;

use anyhow::Result;

use crate::help_system::runner::CommandRunner;
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct KubectlHelpStrategy {
    runner: Arc<CommandRunner>,
}

impl KubectlHelpStrategy {
    pub fn new(runner: Arc<CommandRunner>) -> Self {
        Self { runner }
    }
}

impl HelpStrategy for KubectlHelpStrategy {
    fn priority(&self) -> i32 {
        2
    }

    fn supports_command(&self, base_cmd: &str) -> bool {
        base_cmd == "kubectl"
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        if !command.has_sub_command(1) {
            return self.runner.run("kubectl", &["--help"]);
        }

        let mut args: Vec<&str> = command.sub_cmds().iter().map(|s| s.as_str()).collect();
        args.push("--help");
        self.runner.run("kubectl", &args)
    }
}
