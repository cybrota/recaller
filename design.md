# Recaller Design Doc

### Overview
- **Purpose**: Provide a globally available command suggestion tool for Linux & Unix developers, leveraging shell history and triggered by customizable keystrokes.
- **Target Audience**: Developers working on Linux & Unix systems.

### Functional Requirements

1. **Cross-shell Compatibility**
    * Support for Bash, Zsh, Fish, and tcsh.
2. **Configurable Keystroke Trigger**
    * Default: `Ctrl + Shift + Space`
    * User-configurable via configuration file or CLI flag.
3. **Command Suggestion Logic**
    * **Primary**: Partial Matching with Recency Bias
    * **Secondary**: Frequency of Use (as a tiebreaker)
4. **Shell History Integration**
    * Utilize shell's built-in history file, with an option to merge across shells.
    * Maintain a tool-specific history cache (in addition to shell history)
5. **User Interface**
    * **Initial Implementation**: Inline Autocomplete
    * **Future Enhancement**: Optional Dropdown List or Sidebar

### Non-Functional Requirements

1. **Performance**
    * Minimize latency in suggestion prompts (<200ms).
2. **Security**
    * Handle sensitive command histories securely (e.g., encryption for stored histories).
    * Future improvements: Use masking to hide sensitive information.
3. **Testability**
    * Unit tests for suggestion logic and integration tests for shell interactions.
4. **Simplicity**
    * Prioritize a simple, intuitive user experience.

### Technical Details

1. **Programming Language**: Go
2. **Dependencies**:
    * `github.com/gizak/termui/v3` for terminal UI (subject to change based on developer preference)
    * `github.com/howeyc/fsnotify` for monitoring history file changes (if directly reading from shell's history)
