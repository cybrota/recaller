use anyhow::Result;

#[derive(Clone, Debug)]
pub struct CommandParts {
    pub parts: Vec<String>,
}

impl CommandParts {
    pub fn new(parts: Vec<String>) -> Self {
        Self { parts }
    }

    pub fn base_cmd(&self) -> Option<&str> {
        self.parts.get(0).map(|s| s.as_str())
    }

    pub fn sub_cmds(&self) -> &[String] {
        if self.parts.len() > 1 {
            &self.parts[1..]
        } else {
            &[]
        }
    }

    pub fn has_sub_command(&self, count: usize) -> bool {
        self.sub_cmds().len() >= count
    }

    pub fn get_sub_command(&self, idx: usize) -> Option<&str> {
        self.sub_cmds().get(idx).map(|s| s.as_str())
    }

    pub fn full_name(&self) -> String {
        self.parts.join(" ")
    }
}

pub trait HelpStrategy: Send + Sync {
    fn priority(&self) -> i32 {
        100
    }

    fn supports_command(&self, base_cmd: &str) -> bool;

    fn get_help(&self, command: &CommandParts) -> Result<String>;
}
