# Recaller
[![Go](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](#license)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey.svg)](#supported-platforms)

<picture width="500">
  <source
    width="100%"
    media="(prefers-color-scheme: dark)"
    src="https://github.com/cybrota/recaller/blob/main/recaller-logo.png"
    alt="Recaller logo (dark)"
  />
  <img
    width="100%"
    src="https://github.com/cybrota/recaller/blob/main/logo.png"
    alt="Recaller logo (light)"
  />
</picture>

Recaller searches your files and shell history locally with smart ranking, instant help lookup, and terminal integration. All processing happens on your machine - your command history never leaves your computer.

Install Recaller easily with this script!
```bash
curl -sf https://raw.githubusercontent.com/cybrota/recaller/refs/heads/main/install.sh | sh
```

Key Features of Recaller:

* **Smart Command Search**: Commands ranked by frequency and recency with configurable search modes.
* **Instant Help**: View man pages and command documentation without leaving the interface.
* **Terminal Integration**: Copy to clipboard or execute in new terminal tabs.
* **Multi-Directory Indexing**: Index multiple directories simultaneously for comprehensive file search.
* **Auto Re-indexing**: Automatically discovers new files when launching the search UI.
* **Smart File Ranking**: Files ranked by access frequency and recency.
* **Privacy First**: All processing happens locally - your history stays on your machine.

## Why use Recaller?

Tired of forgetting complex commands or searching for files across different directories? Recaller is here to help. It provides a fast, private, and efficient way to search your command history and files. With its smart ranking and instant help features, you can boost your productivity and streamline your workflow. And since all processing happens locally, your data remains private and secure.

## Getting Started

### Installing Recaller

**Install Script (Recommended)**
```bash
curl -sf https://raw.githubusercontent.com/cybrota/recaller/refs/heads/main/install.sh | sh
```

**Or via HomeBrew (Ideal for macOS users)**
```bash
brew tap cybrota/cybrota
brew install recaller
```

**Or build from source** (requires Go 1.18+)
```bash
git clone https://github.com/cybrota/recaller.git
cd recaller && go build -o recaller . && sudo mv recaller /usr/local/bin/
```

### Configuring Your Project

**Shell Configuration** (Required for Bash users)
- **Bash**: Follow [setup guide](docs/setup-bash.md) to enable timestamped history
- **Zsh**: Works out of the box, see [setup guide](docs/setup-zsh.md) for optimization

**Configuration** (Optional)
Create `~/.recaller.yaml` to customize search behavior:
```yaml
history:
  # Default: true (fuzzy search - matches substring anywhere)
  enable_fuzzing: true
  # Set to false for prefix-based search only
  # enable_fuzzing: false

filesystem:
  # Enable filesystem search functionality
  enabled: true
  # Maximum number of files to index (default: 100000)
  max_indexed_files: 100000
  # Bloom filter settings for memory efficiency
  bloom_filter_size: 1000000
  bloom_filter_hashes: 5
  # Patterns to ignore during indexing
  ignore_patterns:
    - "*.tmp"
    - "*.log"
    - ".git"
    - "node_modules"
    - ".DS_Store"

# Reduce the verbosity of app. Default is false.
quiet: true
```

## Usage

### Command History Search
```bash
recaller                    # Launch interactive command history search
recaller run                # Same as above
recaller history            # View history with filtering
```

### Filesystem Search
```bash
# Index directories for filesystem search
recaller fs index                    # Index current directory recursively
recaller fs index ~/Documents        # Index specific directory recursively
recaller fs index /usr/local ~/code  # Index multiple directories recursively

# Launch filesystem search UI
recaller fs                          # Launch search UI (auto re-indexes tracked paths)

# Manage filesystem index
recaller fs clean --stale            # Remove entries for deleted files
recaller fs clean --older-than 30    # Remove entries older than 30 days
recaller fs clean --clear            # Clear entire index
recaller fs clean --dry-run          # Preview what would be cleaned
```

### Configuration
```bash
recaller settings list      # View current configuration settings
recaller version            # Check version
```

## 🤝 Contributing

Contributions welcome! Areas for improvement:
- Shell support (Fish, PowerShell)
- Terminal emulator support
- Performance optimizations
- Test coverage

```bash
git clone https://github.com/cybrota/recaller.git
cd recaller && go mod tidy && go run .
```

## 📝 License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

Copyright © 2025 [Naren Yellavula](https://github.com/narenaryan)
