use std::io::Read;
use std::time::Duration;

use anyhow::{Context, Result, anyhow};

use crate::help_system::strategy::{CommandParts, HelpStrategy};

const TLDR_TIMEOUT: Duration = Duration::from_secs(10);
const MAX_TLDR_SIZE: usize = 512 * 1024;

pub struct TldrStrategy;

impl TldrStrategy {
    pub fn new() -> Self {
        Self
    }
}

impl HelpStrategy for TldrStrategy {
    fn priority(&self) -> i32 {
        0
    }

    fn supports_command(&self, _: &str) -> bool {
        true
    }

    fn get_help(&self, command: &CommandParts) -> Result<String> {
        let base = command
            .base_cmd()
            .ok_or_else(|| anyhow!("no base command provided"))?;

        let base_url =
            "https://raw.githubusercontent.com/tldr-pages/tldr/refs/heads/main/pages/common";
        let path = if command.has_sub_command(1) {
            format!(
                "{}/{}-{}.md",
                base_url,
                base,
                command.get_sub_command(0).unwrap()
            )
        } else {
            format!("{}/{}.md", base_url, base)
        };

        let response = ureq::get(&path)
            .timeout(TLDR_TIMEOUT)
            .call()
            .with_context(|| format!("failed to fetch TLDR page from {path}"))?;

        if response.status() != 200 {
            return Err(anyhow!(
                "TLDR page not found for {base} (HTTP {})",
                response.status()
            ));
        }

        let mut reader = response.into_reader();
        let mut buffer = Vec::new();
        let mut chunk = [0u8; 8192];
        while buffer.len() < MAX_TLDR_SIZE {
            let read_len = std::cmp::min(chunk.len(), MAX_TLDR_SIZE - buffer.len());
            match reader.read(&mut chunk[..read_len]) {
                Ok(0) => break,
                Ok(n) => buffer.extend_from_slice(&chunk[..n]),
                Err(err) => return Err(anyhow!("failed to read TLDR response: {err}")),
            }
        }

        let mut content = String::from_utf8_lossy(&buffer).to_string();
        if content.is_empty() {
            return Err(anyhow!("TLDR page empty"));
        }
        content = format!("📚 TLDR Documentation:\n\n{}", content);
        Ok(content)
    }
}
