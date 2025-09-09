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

// KubectlHelpStrategy handles kubectl commands with sub-commands
type KubectlHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewKubectlHelpStrategy(cmdRunner *CommandRunner) *KubectlHelpStrategy {
	return &KubectlHelpStrategy{cmdRunner: cmdRunner}
}

func (k *KubectlHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "kubectl"
}

func (k *KubectlHelpStrategy) Priority() int {
	return 2
}

func (k *KubectlHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return k.cmdRunner.Run("kubectl", "--help")
	}

	// Handle kubectl subcommand help - supports multiple levels
	args := append(cmd.SubCmds, "--help")
	return k.cmdRunner.Run("kubectl", args...)
}
