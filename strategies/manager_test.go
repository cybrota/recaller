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
	"strings"
	"testing"
)

func TestHelpStrategyManager(t *testing.T) {
	manager := NewHelpStrategyManager()

	// Test with a command that should be handled by Git strategy
	cmdParts := []string{"git", "status"}
	help, err := manager.GetHelp(cmdParts)

	// We expect either TLDR help (which would come first) or Git help
	// The test should not fail if git is not installed
	if err != nil && !strings.Contains(err.Error(), "failed to get help for command") {
		t.Errorf("Unexpected error getting help: %v", err)
	}

	// If we got help, it should not be empty
	if err == nil && help == "" {
		t.Errorf("Expected non-empty help text")
	}
}

func TestCommand(t *testing.T) {
	cmdParts := []string{"git", "config", "--global"}
	cmd := NewCommand(cmdParts)

	if cmd.BaseCmd != "git" {
		t.Errorf("Expected BaseCmd to be 'git', got '%s'", cmd.BaseCmd)
	}

	if !cmd.HasSubCommand(2) {
		t.Errorf("Expected command to have at least 2 sub-commands")
	}

	if cmd.GetSubCommand(0) != "config" {
		t.Errorf("Expected first sub-command to be 'config', got '%s'", cmd.GetSubCommand(0))
	}

	if cmd.GetSubCommand(1) != "--global" {
		t.Errorf("Expected second sub-command to be '--global', got '%s'", cmd.GetSubCommand(1))
	}

	if cmd.FullName != "git config --global" {
		t.Errorf("Expected FullName to be 'git config --global', got '%s'", cmd.FullName)
	}
}
