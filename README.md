# Recaller

![Recaller Logo](logo.png)

> **Fast, private command history search with instant documentation**

Recaller searches your shell history locally with smart ranking, instant help lookup, and terminal integration. All processing happens on your machine - your command history never leaves your computer.

[![Go](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](#license)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey.svg)](#supported-platforms)

## ✨ Features

- **Smart Search**: Commands ranked by frequency and recency with fuzzy matching
- **Instant Help**: View man pages and command documentation without leaving the interface
- **Terminal Integration**: Copy to clipboard or execute in new terminal tabs
- **Privacy First**: All processing happens locally - your history stays on your machine
- **Keyboard-Driven**: Full keyboard navigation with intuitive shortcuts
- **Cross-Platform**: Works on macOS and Linux with automatic terminal detection

## 🚀 Installation

**Install Script (Recommended)**
```bash
curl -sf https://raw.githubusercontent.com/cybrota/recaller/refs/heads/main/install.sh | sh
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

**Usage**
```bash
recaller              # Launch interactive search
recaller history      # View history with filtering
recaller version      # Check version
```

## ⌨️ Keyboard Shortcuts

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `Enter` | Copy to clipboard | `Ctrl+E` | Execute in terminal |
| `↑/↓` | Navigate | `Tab` | Switch panels |
| `F1` | Show help | `Esc` | Quit |

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

## 👨‍💻 Author

**Naren Yellavula**
- GitHub: [@narenaryan](https://github.com/narenaryan)
- Website: [https://github.com/narenaryan](https://github.com/narenaryan)

## 🙏 Acknowledgments

- Built with [termui](https://github.com/gizak/termui) for the beautiful terminal interface
- Uses [clipboard](https://github.com/atotto/clipboard) for cross-platform clipboard support
- Inspired by the need for better command-line productivity tools

---

<div align="center">

**Star ⭐ this repository if you find it useful!**

[Report Bug](https://github.com/cybrota/recaller/issues) · [Request Feature](https://github.com/cybrota/recaller/issues) · [Documentation](https://github.com/cybrota/recaller/wiki)

</div>
