use anyhow::{Context, Result, anyhow};
use arboard::Clipboard;
use std::process::Command;

pub fn copy_to_clipboard(text: &str) -> Result<()> {
    let mut clipboard = Clipboard::new().context("failed to access clipboard")?;
    clipboard
        .set_text(text.to_string())
        .context("failed to copy text to clipboard")
}

pub fn send_to_terminal(command: &str) -> Result<()> {
    #[cfg(target_os = "macos")]
    {
        return send_to_terminal_macos(command);
    }

    #[cfg(target_os = "linux")]
    {
        return send_to_terminal_linux(command);
    }

    #[cfg(not(any(target_os = "macos", target_os = "linux")))]
    {
        Err(anyhow!(
            "terminal automation not supported on this platform"
        ))
    }
}

#[cfg(target_os = "macos")]
fn send_to_terminal_macos(command: &str) -> Result<()> {
    let escaped = command.replace('"', "\\\"");
    let script = format!(
        r#"tell application "Terminal"
    activate
    if (count of windows) = 0 then
        do script "{cmd}"
    else
        do script "{cmd}" in front window
    end if
end tell"#,
        cmd = escaped
    );

    let status = Command::new("osascript")
        .args(["-e", &script])
        .status()
        .context("failed to invoke osascript")?;

    if status.success() {
        return Ok(());
    }

    let iterm_script = format!(
        r#"tell application "iTerm2"
    tell current window
        create tab with default profile
        tell current session to write text "{cmd}"
    end tell
    activate
end tell"#,
        cmd = escaped
    );

    Command::new("osascript")
        .args(["-e", &iterm_script])
        .status()
        .context("failed to invoke osascript for iTerm")?
        .success()
        .then_some(())
        .ok_or_else(|| anyhow!("failed to send command to Terminal or iTerm"))
}

#[cfg(target_os = "linux")]
fn send_to_terminal_linux(command: &str) -> Result<()> {
    let wrapped = if command.trim_end().ends_with("exec bash") {
        command.to_string()
    } else {
        format!("{}; exec bash", command)
    };

    let terminals: &[(&str, &[&str])] = &[
        ("gnome-terminal", &["--tab", "--", "bash", "-lc"]),
        ("konsole", &["--new-tab", "-e", "bash", "-lc"]),
        ("xfce4-terminal", &["--tab", "-x", "bash", "-lc"]),
        ("tilix", &["-a", "session-add-down", "-e", "bash", "-lc"]),
        ("terminator", &["--new-tab", "-e", "bash", "-lc"]),
        ("alacritty", &["-e", "bash", "-lc"]),
        ("kitty", &["--tab", "bash", "-lc"]),
        ("xterm", &["-e", "bash", "-lc"]),
    ];

    for (term, args) in terminals {
        if which::which(term).is_ok() {
            let mut cmd = Command::new(term);
            for arg in *args {
                cmd.arg(arg);
            }
            cmd.arg(&wrapped);
            cmd.spawn()
                .with_context(|| format!("failed to launch {term}"))?;
            return Ok(());
        }
    }

    Err(anyhow!("no supported terminal emulator found"))
}

pub fn open_path(path: &str) -> Result<()> {
    #[cfg(target_os = "macos")]
    {
        Command::new("open")
            .arg(path)
            .spawn()
            .context("failed to spawn open")?;
        return Ok(());
    }

    #[cfg(target_os = "linux")]
    {
        Command::new("xdg-open")
            .arg(path)
            .spawn()
            .context("failed to spawn xdg-open")?;
        return Ok(());
    }

    #[cfg(not(any(target_os = "macos", target_os = "linux")))]
    {
        Err(anyhow!("opening files is not supported on this platform"))
    }
}
