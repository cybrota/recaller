use std::io::Read;
use std::process::{Command, Stdio};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use anyhow::{Context, Result, anyhow};
use wait_timeout::ChildExt;

const DEFAULT_CMD_TIMEOUT: Duration = Duration::from_secs(30);
const GIT_CMD_TIMEOUT: Duration = Duration::from_secs(15);
const MAX_OUTPUT_SIZE: usize = 1024 * 1024; // 1MB

pub struct CommandRunner;

impl CommandRunner {
    pub fn new() -> Self {
        Self
    }

    pub fn run(&self, program: &str, args: &[&str]) -> Result<String> {
        self.run_with_timeout(DEFAULT_CMD_TIMEOUT, program, args, &[])
    }

    pub fn run_git(&self, program: &str, args: &[&str]) -> Result<String> {
        self.run_with_timeout(GIT_CMD_TIMEOUT, program, args, &[])
    }

    pub fn run_with_env(
        &self,
        program: &str,
        args: &[&str],
        env: &[(&str, &str)],
    ) -> Result<String> {
        self.run_with_timeout(DEFAULT_CMD_TIMEOUT, program, args, env)
    }

    pub fn command_exists(&self, program: &str) -> bool {
        which::which(program).is_ok()
    }

    fn run_with_timeout(
        &self,
        timeout: Duration,
        program: &str,
        args: &[&str],
        env: &[(&str, &str)],
    ) -> Result<String> {
        let mut cmd = Command::new(program);
        cmd.args(args);
        cmd.stdout(Stdio::piped());
        cmd.stderr(Stdio::piped());
        for (key, value) in env {
            cmd.env(key, value);
        }

        let mut child = cmd
            .spawn()
            .with_context(|| format!("failed to spawn {program}"))?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| anyhow!("failed to capture stdout"))?;
        let stderr = child
            .stderr
            .take()
            .ok_or_else(|| anyhow!("failed to capture stderr"))?;

        let buffer = Arc::new(Mutex::new(LimitedBuffer::new(MAX_OUTPUT_SIZE)));
        let buffer_stdout = buffer.clone();
        let stdout_handle = std::thread::spawn(move || read_stream(stdout, buffer_stdout));
        let buffer_stderr = buffer.clone();
        let stderr_handle = std::thread::spawn(move || read_stream(stderr, buffer_stderr));

        let status = match child.wait_timeout(timeout)? {
            Some(status) => status,
            None => {
                child.kill().ok();
                child.wait().ok();
                stdout_handle.join().ok();
                stderr_handle.join().ok();
                return Err(anyhow!("command timed out"));
            }
        };

        stdout_handle.join().ok();
        stderr_handle.join().ok();

        let buf = buffer.lock().unwrap();
        let mut output = String::from_utf8_lossy(&buf.buf).to_string();
        if buf.truncated {
            output.push_str("\n[OUTPUT TRUNCATED - Size limit exceeded]");
        }

        if status.success() {
            Ok(output)
        } else if output.trim().is_empty() {
            Err(anyhow!("command exited with status {status}"))
        } else {
            Err(anyhow!(output))
        }
    }
}

fn read_stream(mut reader: impl Read + Send + 'static, buffer: Arc<Mutex<LimitedBuffer>>) {
    let mut chunk = [0u8; 8192];
    loop {
        match reader.read(&mut chunk) {
            Ok(0) => break,
            Ok(n) => {
                let mut buf = buffer.lock().unwrap();
                buf.write(&chunk[..n]);
                if buf.is_full() {
                    break;
                }
            }
            Err(_) => break,
        }
    }
}

struct LimitedBuffer {
    buf: Vec<u8>,
    limit: usize,
    truncated: bool,
}

impl LimitedBuffer {
    fn new(limit: usize) -> Self {
        Self {
            buf: Vec::with_capacity(limit.min(4096)),
            limit,
            truncated: false,
        }
    }

    fn write(&mut self, data: &[u8]) {
        if self.buf.len() >= self.limit {
            self.truncated = true;
            return;
        }

        let remaining = self.limit - self.buf.len();
        let to_copy = remaining.min(data.len());
        self.buf.extend_from_slice(&data[..to_copy]);
        if to_copy < data.len() {
            self.truncated = true;
        }
    }

    fn is_full(&self) -> bool {
        self.buf.len() >= self.limit
    }
}
