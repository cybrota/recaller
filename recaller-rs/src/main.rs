mod banner;
mod cli;
mod commands;
mod config;
mod constants;
mod fs;
mod help;
mod help_system;
mod history;
mod platform;
mod state;
mod ui;
mod version;

use clap::Parser;

use crate::cli::{Cli, Commands};
use crate::commands::{
    handle_fs, handle_history, handle_run, handle_settings, handle_usage, handle_version,
};
use crate::config::Config;
use crate::state::AppState;

fn main() {
    let cli = Cli::parse();

    let config = match config::load_config() {
        Ok(cfg) => cfg,
        Err(err) => {
            config::print_config_error(&err);
            Config::default()
        }
    };

    let mut state = AppState::new(config);
    let command = cli.command.unwrap_or(Commands::Run);

    let result = match command {
        Commands::Run => handle_run(&mut state),
        Commands::Usage => {
            handle_usage();
            Ok(())
        }
        Commands::History(args) => handle_history(&mut state, &args),
        Commands::Fs(args) => handle_fs(&mut state, &args),
        Commands::Settings(args) => handle_settings(&args),
        Commands::Version => {
            handle_version();
            Ok(())
        }
    };

    if let Err(err) = result {
        eprintln!("❌ {err}");
        std::process::exit(1);
    }
}
