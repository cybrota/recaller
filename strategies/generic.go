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

import "fmt"

// GenericHelpStrategy tries common help flags
type GenericHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewGenericHelpStrategy(cmdRunner *CommandRunner) *GenericHelpStrategy {
	return &GenericHelpStrategy{cmdRunner: cmdRunner}
}

func (g *GenericHelpStrategy) SupportsCommand(baseCmd string) bool {
	return g.cmdRunner.CheckCommandExists(baseCmd)
}

func (g *GenericHelpStrategy) Priority() int {
	return 8 // Lower priority than specific strategies
}

func (g *GenericHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	// Try different help flags
	helpFlags := []string{"-h", "--help", "help"}

	for _, flag := range helpFlags {
		args := append(cmd.SubCmds, flag)
		if out, err := g.cmdRunner.Run(cmd.BaseCmd, args...); err == nil && out != "" {
			return out, nil
		}
	}

	return "", fmt.Errorf("no help found for command %q", cmd.FullName)
}
