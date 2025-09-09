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

// CargoHelpStrategy handles Cargo commands
type CargoHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewCargoHelpStrategy(cmdRunner *CommandRunner) *CargoHelpStrategy {
	return &CargoHelpStrategy{cmdRunner: cmdRunner}
}

func (c *CargoHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "cargo"
}

func (c *CargoHelpStrategy) Priority() int {
	return 2
}

func (c *CargoHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return c.cmdRunner.Run("cargo", "--help")
	}

	subCmd := cmd.GetSubCommand(0)
	return c.cmdRunner.Run("cargo", subCmd, "--help")
}
