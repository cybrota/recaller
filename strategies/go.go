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

// GoHelpStrategy handles Go commands
type GoHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewGoHelpStrategy(cmdRunner *CommandRunner) *GoHelpStrategy {
	return &GoHelpStrategy{cmdRunner: cmdRunner}
}

func (g *GoHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "go"
}

func (g *GoHelpStrategy) Priority() int {
	return 2
}

func (g *GoHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return g.cmdRunner.Run("go", "help")
	}

	subCmd := cmd.GetSubCommand(0)
	return g.cmdRunner.Run("go", "help", subCmd)
}
