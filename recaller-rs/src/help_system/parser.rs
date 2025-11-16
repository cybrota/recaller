use anyhow::Result;

pub fn split_command(command: &str) -> Result<Vec<String>> {
    let parts = shell_words::split(command)?;
    Ok(parts)
}
