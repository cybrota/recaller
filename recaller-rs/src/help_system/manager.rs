use std::sync::Arc;

use anyhow::{Result, anyhow};

use crate::help_system::cache::HelpCache;
use crate::help_system::runner::CommandRunner;
use crate::help_system::strategies::{
    AwsHelpStrategy, CargoHelpStrategy, DockerHelpStrategy, GenericHelpStrategy, GitHelpStrategy,
    GoHelpStrategy, KubectlHelpStrategy, ManPageStrategy, NpmHelpStrategy, TldrStrategy,
};
use crate::help_system::strategy::{CommandParts, HelpStrategy};

pub struct HelpManager {
    cache: HelpCache,
    strategies: Vec<Box<dyn HelpStrategy>>, // excluding TLDR
    tldr_strategy: TldrStrategy,
}

impl HelpManager {
    pub fn new() -> Self {
        let runner = Arc::new(CommandRunner::new());
        let mut strategies: Vec<Box<dyn HelpStrategy>> = Vec::new();
        strategies.push(Box::new(GitHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(GoHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(KubectlHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(CargoHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(NpmHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(AwsHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(DockerHelpStrategy::new(runner.clone())));
        strategies.push(Box::new(ManPageStrategy::new(runner.clone())));
        strategies.push(Box::new(GenericHelpStrategy::new(runner)));

        strategies.sort_by_key(|s| s.priority());

        Self {
            cache: HelpCache::new(),
            strategies,
            tldr_strategy: TldrStrategy::new(),
        }
    }

    pub fn get_help(&self, command: &[String]) -> Result<String> {
        if command.is_empty() {
            return Err(anyhow!("no command provided"));
        }

        let command_key = command.join(" ");
        if let Some(cached) = self.cache.get(&command_key) {
            return Ok(cached);
        }

        let parts = CommandParts::new(command.to_vec());
        let mut last_err = None;

        if let Ok(help) = self.tldr_strategy.get_help(&parts) {
            if !help.trim().is_empty() {
                self.cache.insert(&command_key, &help);
                return Ok(help);
            }
        }

        let base_cmd = parts.base_cmd().unwrap_or("");
        for strategy in self
            .strategies
            .iter()
            .filter(|s| s.supports_command(base_cmd))
        {
            match strategy.get_help(&parts) {
                Ok(help) if !help.trim().is_empty() => {
                    self.cache.insert(&command_key, &help);
                    return Ok(help);
                }
                Ok(_) => continue,
                Err(err) => last_err = Some(err),
            }
        }

        Err(last_err.unwrap_or_else(|| anyhow!("no help strategy succeeded")))
    }
}
