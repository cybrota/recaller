// Copyright 2025 Naren Yellavula
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"runtime"

	markdown "github.com/MichaelMure/go-term-markdown"
)

func getHelpMessage() string {
	message := fmt.Sprintf(`

 **Recaller %s**

Easily access & re-run your most frequent shell history with blazing-fast search and documentation help.
No more cycling back with <bck-isearch>. See the latest history for any shell command.

Built with Go %s

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

`, version, runtime.Version())
	result := markdown.Render(string(message), 80, 3)
	return string(result)
}
