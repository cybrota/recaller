use rustc_version_runtime::version;

use crate::version::VERSION;

pub fn usage_text() -> String {
    let rust_version = version();

    format!(
        r#"

 **Recaller {version}**

Easily access & re-run your most frequent shell history with blazing-fast search and documentation help.
No more cycling back with <bck-isearch>. See the latest history for any shell command.

Built with Rust {rust_version}

# 1. Features
* Recall your shell commands based on recency & frequency
* Fast access to documentation within shell for your favorite commands (Ex: kubectl, terraform, AWS CLI, cargo, go, npm, all man pages etc.)
* Elegant Terminal UI to quickly see history & associated help pages

# 2. Supported Platforms
* Linux/Unix
* Mac OSX

# Supported Terminals
* Bash
* Zshell (Zsh)

# Please be aware
* Copy to clipboard feature on Linux or Unix requires 'xclip' or 'xsel' command to be installed

# License
Licensed under the Apache License, Version 2.0
Copyright © 2025 Naren Yellavula

"#,
        version = VERSION,
        rust_version = rust_version,
    )
}
