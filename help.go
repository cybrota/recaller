package main

func getHelpMessage() string {
	message := `
Recaller

Easily access & re-run your most frequent shell history with blazing-fast search and documentation help.
No more cycling back with <bck-isearch>. See the latest history for any shell command.

# Features
* Recall your shell commands based on recency & frequency
* Fast access to documentation within shell for your favorite commands (Ex: kubectl, terraform, AWS CLI, cargo, go, npm, all man pages etc.)
* Elegant Terminal UI to quickly see history & associated help pages

# Supported Platforms
* Linux/Unix
* Mac OSX

# Supported Terminals
* Bash
* Zshell (Zsh)

# Pleas be aware
* Copy to cliboard feature on Linux or Unix requires 'xclip' or 'xsel' command to be installed
`
	return message
}
