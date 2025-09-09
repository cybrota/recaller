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

// AwsHelpStrategy handles AWS CLI commands with multiple sub-command levels
type AwsHelpStrategy struct {
	cmdRunner *CommandRunner
}

func NewAwsHelpStrategy(cmdRunner *CommandRunner) *AwsHelpStrategy {
	return &AwsHelpStrategy{cmdRunner: cmdRunner}
}

func (a *AwsHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "aws"
}

func (a *AwsHelpStrategy) Priority() int {
	return 2
}

func (a *AwsHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return a.cmdRunner.Run("aws", "help")
	}

	// AWS CLI supports help at multiple levels: aws s3 help, aws s3 cp help
	args := append(cmd.SubCmds, "help")
	if out, err := a.cmdRunner.Run("aws", args...); err == nil {
		return RemoveOverstrike(out), nil
	}

	return "", fmt.Errorf("AWS command %q is invalid or not found", cmd.FullName)
}
