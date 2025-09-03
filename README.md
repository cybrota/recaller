# Recaller

![Recaller Logo](logo.png)

> **Fast, private command history search with instant documentation**

Recaller searches your shell history locally with smart ranking, instant help lookup, and terminal integration. All processing happens on your machine - your command history never leaves your computer.

[![Go](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](#license)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey.svg)](#supported-platforms)

## ✨ Features

### 🔍 Command History Search
- **Smart Search**: Commands ranked by frequency and recency with configurable search modes
  - **Fuzzy Search**: Substring matching anywhere in commands (default)
  - **Prefix Search**: Fast matching of command beginnings (configurable)
- **Instant Help**: View man pages and command documentation without leaving the interface
- **Terminal Integration**: Copy to clipboard or execute in new terminal tabs

### 📁 Filesystem Search
- **Multi-Directory Indexing**: Index multiple directories simultaneously for comprehensive file search
- **Auto Re-indexing**: Automatically discovers new files when launching the search UI
- **Smart File Ranking**: Files ranked by access frequency and recency
- **Filter Toggle**: Instantly switch between showing all files, directories only, or files only
- **Fast Search**: Blazing-fast search through indexed files and directories
- **File Operations**: Open files with default applications or copy paths to clipboard

### 🔒 Privacy & Performance  
- **Privacy First**: All processing happens locally - your history stays on your machine
- **Keyboard-Driven**: Full keyboard navigation with intuitive shortcuts
- **Cross-Platform**: Works on macOS and Linux with automatic terminal detection

## 🚀 Installation

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

## 🔧 Setup

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
```

## 💻 Usage

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

> **🔄 Auto Re-indexing**: When you launch `recaller fs`, it automatically re-indexes all previously indexed directories to discover new files and folders that were added since the last indexing. This keeps your search results up-to-date without manual intervention.

### Configuration
```bash
recaller settings list      # View current configuration settings
recaller version            # Check version
```

## 🔍 Search Modes

**Fuzzy Search** (Default)
- Matches commands containing your search query **anywhere**
- More intuitive and finds commands with keywords in any position
- Example: `commit` matches `git commit -m "fix"`, `pre-commit run`, etc.

**Prefix Search** (Configurable)
- Matches commands that **start with** your search query
- Fast and efficient for finding commands by their beginning
- Example: `git` matches `git status`, `git commit`, etc.

## ⌨️ Keyboard Shortcuts

### Command History Search
| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `Enter` | Copy to clipboard | `Ctrl+E` | Execute in terminal |
| `↑/↓` | Navigate | `Tab` | Switch panels |
| `F1` | Show help | `Ctrl+R` | Reset input |
| `Ctrl+U` | Insert command | `Ctrl+Z` | Copy text |
| `Ctrl+J/K` | Jump first/last | `Esc` | Quit |

### Filesystem Search
| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `Enter` | Open file | `Ctrl+X` | Copy path |
| `↑/↓` | Navigate | `Tab` | Switch panels |
| `Ctrl+T` | **Toggle filter** | `Ctrl+R` | Reset input |
| `Ctrl+J/K` | Jump first/last | `Esc` | Quit |

> **New**: `Ctrl+T` cycles through filter modes: **All** (📁📄) → **Directories** (📁) → **Files** (📄)

## 🔒 Privacy & Security

**Your data stays local**: Recaller processes your command history entirely on your machine. No data is sent to external servers or cloud services. Your command history remains private and secure.

## 📋 Requirements

- **OS**: macOS 10.12+ or Linux
- **Clipboard**: Linux users need `xclip` (`sudo apt install xclip`)
- **Terminals**: Auto-detects Terminal.app, iTerm2, GNOME Terminal, Konsole, and others


## 🔄 Shell Support

| Shell | Support | Setup |
|-------|---------|-------|
| **Bash** | ✅ Full | [Required config](docs/setup-bash.md) |
| **Zsh** | ✅ Full | [Optional config](docs/setup-zsh.md) |
| **Fish** | 🔄 Planned | - |

> **⚠️ Bash users**: Requires timestamped history. See [setup guide](docs/setup-bash.md).

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

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright © 2025 [Naren Yellavula](https://github.com/narenaryan)

## 🙏 Acknowledgments

- Built with [termui](https://github.com/gizak/termui) for the beautiful terminal interface
- Uses [clipboard](https://github.com/atotto/clipboard) for cross-platform clipboard support
- Inspired by the need for better command-line productivity tools

---

<div align="center">

**Star ⭐ this repository if you find it useful!**

[Report Bug](https://github.com/cybrota/recaller/issues) · [Request Feature](https://github.com/cybrota/recaller/issues) · [Documentation](https://github.com/cybrota/recaller/wiki)

</div>
