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
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ManPageStrategy handles standard man pages
type ManPageStrategy struct {
	cmdRunner *CommandRunner
}

func NewManPageStrategy(cmdRunner *CommandRunner) *ManPageStrategy {
	return &ManPageStrategy{cmdRunner: cmdRunner}
}

func (m *ManPageStrategy) SupportsCommand(baseCmd string) bool {
	// Check if man page exists
	ctx, cancel := context.WithTimeout(context.Background(), FastCmdTimeout)
	defer cancel()
	manCheck := exec.CommandContext(ctx, "man", "-w", baseCmd)
	return manCheck.Run() == nil
}

func (m *ManPageStrategy) Priority() int {
	return 5 // Lower priority than specific strategies
}

func (m *ManPageStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if output, err := m.cmdRunner.Run("man", cmd.BaseCmd); err == nil {
		// Handle minimal environments where man prints a placeholder message
		if strings.Contains(output, "No manual entry") || strings.Contains(output, "has been minimized") {
			return "", fmt.Errorf("man page not found for command %q", cmd.BaseCmd)
		}
		return RemoveOverstrike(output), nil
	}

	return "", fmt.Errorf("failed to get man page for %q", cmd.BaseCmd)
}
