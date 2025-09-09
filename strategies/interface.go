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

package strategies

import "strings"

// HelpStrategy defines the interface for different command help strategies
type HelpStrategy interface {
	GetHelp(cmdParts []string) (string, error)
	SupportsCommand(baseCmd string) bool
	Priority() int // Lower number = higher priority
}

// Command represents a parsed command with its parts
type Command struct {
	Parts    []string
	BaseCmd  string
	SubCmds  []string
	FullName string
}

// NewCommand creates a new Command from command parts
func NewCommand(parts []string) *Command {
	if len(parts) == 0 {
		return &Command{Parts: parts}
	}

	return &Command{
		Parts:    parts,
		BaseCmd:  parts[0],
		SubCmds:  parts[1:],
		FullName: strings.Join(parts, " "),
	}
}

// HasSubCommand checks if command has at least n sub-commands
func (c *Command) HasSubCommand(n int) bool {
	return len(c.SubCmds) >= n
}

// GetSubCommand returns the nth sub-command (0-indexed)
func (c *Command) GetSubCommand(n int) string {
	if n >= len(c.SubCmds) {
		return ""
	}
	return c.SubCmds[n]
}
