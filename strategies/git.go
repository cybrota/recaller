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

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// GitHelpStrategy handles Git commands with up to 3 levels of sub-commands
type GitHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewGitHelpStrategy(cmdRunner *CommandRunner) *GitHelpStrategy {
	return &GitHelpStrategy{cmdRunner: cmdRunner}
}

func (g *GitHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "git"
}

func (g *GitHelpStrategy) Priority() int {
	return 2
}

func (g *GitHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return g.cmdRunner.RunWithTimeout(GitCmdTimeout, "git", "help")
	}

	// Handle git subcommand help
	subCmd := cmd.GetSubCommand(0)

	// Try git help <subcommand> first
	if out, err := g.runGitHelp(subCmd); err == nil {
		return RemoveOverstrike(out), nil
	}

	// For complex sub-commands like "git config --global", try git <subcommand> --help
	if cmd.HasSubCommand(2) {
		args := append(cmd.SubCmds, "--help")
		if out, err := g.cmdRunner.RunWithTimeout(GitCmdTimeout, "git", args...); err == nil {
			return RemoveOverstrike(out), nil
		}
	}

	return "", fmt.Errorf("failed to get Git help for %q", cmd.FullName)
}

func (g *GitHelpStrategy) runGitHelp(subCmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "help", subCmd)
	cmd.Env = append(os.Environ(), "GIT_PAGER=cat")

	var buf bytes.Buffer
	limitedWriter := &LimitedWriter{w: &buf, limit: MaxOutputSize}
	cmd.Stdout = limitedWriter
	cmd.Stderr = limitedWriter

	err := cmd.Run()
	result := buf.String()

	if limitedWriter.truncated {
		result += "\n[OUTPUT TRUNCATED - Size limit exceeded]"
	}

	return result, err
}
