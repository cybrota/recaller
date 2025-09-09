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

// NpmHelpStrategy handles npm commands
type NpmHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewNpmHelpStrategy(cmdRunner *CommandRunner) *NpmHelpStrategy {
	return &NpmHelpStrategy{cmdRunner: cmdRunner}
}

func (n *NpmHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "npm"
}

func (n *NpmHelpStrategy) Priority() int {
	return 2
}

func (n *NpmHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return n.cmdRunner.Run("npm", "help")
	}

	subCmd := cmd.GetSubCommand(0)
	if out, err := n.cmdRunner.Run("npm", "help", subCmd); err == nil {
		return RemoveOverstrike(out), nil
	}

	// Fallback to npm <subcommand> --help
	return n.cmdRunner.Run("npm", subCmd, "--help")
}
