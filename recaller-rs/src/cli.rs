use std::path::PathBuf;

use crate::version::VERSION;
use clap::{Args, Parser, Subcommand};

#[derive(Debug, Parser)]
#[command(
    name = "recaller",
    version = VERSION,
    about = "Blazing-fast command history search with instant documentation",
    arg_required_else_help = false,
    disable_help_subcommand = true
)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Option<Commands>,
}

#[derive(Debug, Subcommand)]
pub enum Commands {
    /// Launch the Recaller UI for search & documentation
    Run,
    /// Print the Recaller usage guide
    Usage,
    /// Fetch history sorted by time and frequency
    History(HistoryArgs),
    /// Filesystem search commands
    Fs(FsArgs),
    /// Manage Recaller configuration settings
    Settings(SettingsArgs),
    /// Print the current Recaller version
    Version,
}

#[derive(Debug, Args)]
pub struct HistoryArgs {
    /// Match string prefix to look in history
    #[arg(long = "match", default_value = "")]
    pub matcher: String,
}

#[derive(Debug, Args)]
pub struct FsArgs {
    #[command(subcommand)]
    pub command: Option<FsCommand>,
}

#[derive(Debug, Subcommand)]
pub enum FsCommand {
    /// Index directories for filesystem search
    Index(FsIndexArgs),
    /// Clean filesystem index
    Clean(FsCleanArgs),
    /// Re-index tracked paths to discover new files
    Refresh,
}

#[derive(Debug, Args)]
pub struct FsIndexArgs {
    /// Paths to index (defaults to current directory)
    pub paths: Vec<PathBuf>,
}

#[derive(Debug, Args)]
pub struct FsCleanArgs {
    /// Optional path to clean
    pub path: Option<PathBuf>,
    /// Remove entries for files that no longer exist
    #[arg(long)]
    pub stale: bool,
    /// Remove entries older than N days
    #[arg(long = "older-than", default_value_t = 0)]
    pub older_than: u32,
    /// Clear the entire index (requires confirmation)
    #[arg(long)]
    pub clear: bool,
    /// Show what would be cleaned without making changes
    #[arg(long = "dry-run")]
    pub dry_run: bool,
}

#[derive(Debug, Args)]
pub struct SettingsArgs {
    #[command(subcommand)]
    pub command: SettingsCommand,
}

#[derive(Debug, Subcommand)]
pub enum SettingsCommand {
    /// List current configuration settings
    List,
}
