use std::env;
use std::io::{self, Write};
use std::path::PathBuf;

use anyhow::{Context, Result, bail};
use directories::BaseDirs;

use crate::cli::{
    FsArgs, FsCleanArgs, FsCommand, FsIndexArgs, HistoryArgs, SettingsArgs, SettingsCommand,
};
use crate::config;
use crate::fs::{CleanupOptions, FilesystemIndexer};
use crate::help::usage_text;
use crate::history::get_suggestions;
use crate::state::AppState;
use crate::ui;
use crate::version::VERSION;

pub fn handle_run(state: &mut AppState) -> Result<()> {
    ui::history::run(state)
}

pub fn handle_usage() {
    println!("{}", usage_text());
}

pub fn handle_history(state: &mut AppState, args: &HistoryArgs) -> Result<()> {
    let enable_fuzzing = state.config.history.enable_fuzzing;
    let index = state.history_index()?;
    let query = args.matcher.trim();
    let suggestions = get_suggestions(index, query, enable_fuzzing);

    if suggestions.is_empty() {
        println!("No matching history entries found.");
    } else {
        for suggestion in suggestions {
            println!("{}", suggestion);
        }
    }

    Ok(())
}

pub fn handle_fs(state: &mut AppState, args: &FsArgs) -> Result<()> {
    if !state.config.filesystem.enabled {
        println!(
            "❌ Filesystem search is disabled. Enable it in ~/.recaller.yaml or via 'recaller settings list'."
        );
        return Ok(());
    }

    match &args.command {
        Some(FsCommand::Index(index_args)) => handle_fs_index(state, index_args),
        Some(FsCommand::Clean(clean_args)) => handle_fs_clean(state, clean_args),
        Some(FsCommand::Refresh) => handle_fs_refresh(state),
        None => handle_fs_launch(state),
    }
}

fn handle_fs_launch(state: &mut AppState) -> Result<()> {
    let mut indexer = FilesystemIndexer::new(state.config.filesystem.clone());
    indexer.load_or_create_index(!state.config.quiet)?;

    if !indexer.has_indexed_files() {
        println!(
            "📂 No files found in index. Run 'recaller fs index [path]' to index directories first."
        );
        return Ok(());
    }

    if !indexer.get_root_paths().is_empty() {
        if let Err(err) = indexer.refresh_index(!state.config.quiet) {
            println!("⚠️ Re-indexing completed with errors: {err}");
        }
    }

    println!("📊 {}", indexer.get_index_stats());
    ui::filesystem::run(state, &mut indexer)
}

fn handle_fs_index(state: &AppState, args: &FsIndexArgs) -> Result<()> {
    let raw_paths: Vec<PathBuf> = if args.paths.is_empty() {
        vec![PathBuf::from(".")]
    } else {
        args.paths.clone()
    };

    let mut valid_paths = Vec::new();
    for path in raw_paths {
        match normalize_path(&path) {
            Ok(resolved) => {
                if resolved.exists() {
                    valid_paths.push(resolved);
                } else {
                    println!("❌ Path does not exist: {}", resolved.display());
                }
            }
            Err(err) => println!("❌ Failed to resolve {}: {err}", path.display()),
        }
    }

    if valid_paths.is_empty() {
        bail!("no valid paths to index");
    }

    let mut indexer = FilesystemIndexer::new(state.config.filesystem.clone());
    indexer.load_or_create_index(!state.config.quiet)?;

    if valid_paths.len() == 1 {
        println!(
            "🔍 Starting filesystem indexing for {}",
            valid_paths[0].display()
        );
        indexer.index_directory_with_progress(&valid_paths[0], !state.config.quiet)?;
    } else {
        println!(
            "🔍 Starting filesystem indexing for {} directories",
            valid_paths.len()
        );
        indexer.index_directories_with_progress(&valid_paths, !state.config.quiet)?;
    }

    println!("\n💾 Saving index to disk...");
    indexer.persist_index(!state.config.quiet)?;
    println!("{}", indexer.get_index_stats());
    println!("\n💡 Run 'recaller fs' to launch the search UI.");
    Ok(())
}

fn handle_fs_clean(state: &AppState, args: &FsCleanArgs) -> Result<()> {
    let mut indexer = FilesystemIndexer::new(state.config.filesystem.clone());
    indexer.load_or_create_index(!state.config.quiet)?;

    if !indexer.has_indexed_files() {
        println!("📂 No files found in index.");
        return Ok(());
    }

    let index_size = indexer.get_index_file_size().unwrap_or(0);
    println!("📊 {}", indexer.get_index_stats());
    if index_size > 0 {
        println!("💾 Index file size: {:.2} KB", index_size as f64 / 1024.0);
    }

    if args.clear {
        if args.dry_run {
            println!(
                "🔍 [DRY RUN] Would clear entire index ({} entries)",
                indexer.entry_count()
            );
            return Ok(());
        }

        print!("⚠️ This will clear the entire filesystem index. Continue? [y/N]: ");
        io::stdout().flush().ok();
        let mut input = String::new();
        io::stdin().read_line(&mut input).ok();
        if !matches!(input.trim().to_lowercase().as_str(), "y" | "yes") {
            println!("❌ Operation cancelled.");
            return Ok(());
        }

        indexer.clear_index();
        indexer.persist_index(!state.config.quiet)?;
        println!("✅ Index cleared successfully!");
        return Ok(());
    }

    let path_filter = match &args.path {
        Some(path) => Some(normalize_path(path)?.to_string_lossy().to_string()),
        None => None,
    };

    let options = CleanupOptions {
        path: path_filter,
        remove_stale: args.stale,
        older_than_days: args.older_than as i32,
        show_progress: !args.dry_run && !state.config.quiet,
    };

    if args.dry_run {
        println!("🔍 [DRY RUN] Analyzing what would be cleaned...");
    } else {
        println!("🧹 Starting cleanup...");
    }

    let stats = indexer.cleanup_index(options)?;
    println!("\n📈 Cleanup Results:");
    println!("   Total entries: {}", stats.total_entries);
    println!("   Removed entries: {}", stats.removed_entries);
    if stats.stale_files > 0 {
        println!("   Stale files removed: {}", stats.stale_files);
    }
    if stats.old_files > 0 {
        println!("   Old entries removed: {}", stats.old_files);
    }
    if stats.freed_kb > 0.0 {
        println!("   Memory freed: {:.2} KB", stats.freed_kb);
    }

    if !args.dry_run && stats.removed_entries > 0 {
        println!("\n💾 Saving cleaned index...");
        indexer.persist_index(!state.config.quiet)?;
        println!("✅ Done");
    } else if args.dry_run {
        println!("\n💡 Run without --dry-run to apply these changes.");
    } else {
        println!("\n✅ No cleanup needed - index is already clean!");
    }

    Ok(())
}

fn handle_fs_refresh(state: &AppState) -> Result<()> {
    let mut indexer = FilesystemIndexer::new(state.config.filesystem.clone());
    indexer.load_or_create_index(!state.config.quiet)?;
    indexer.refresh_index(!state.config.quiet)?;
    println!("✔️ Refresh completed successfully!");
    Ok(())
}

pub fn handle_settings(args: &SettingsArgs) -> Result<()> {
    match args.command {
        SettingsCommand::List => {
            config::display_settings()?;
        }
    }
    Ok(())
}

pub fn handle_version() {
    println!("{}", VERSION);
}

fn normalize_path(path: &PathBuf) -> Result<PathBuf> {
    let path_str = path.to_string_lossy().to_string();
    let candidate = if let Some(stripped) = path_str.strip_prefix("~/") {
        let dirs = BaseDirs::new().context("failed to determine home directory")?;
        dirs.home_dir().join(stripped)
    } else {
        PathBuf::from(&path_str)
    };

    if candidate.is_absolute() {
        Ok(candidate)
    } else {
        Ok(env::current_dir()?.join(candidate))
    }
}
