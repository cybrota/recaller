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

package main

import (
	"github.com/cybrota/recaller/strategies"
	"github.com/mattn/go-shellwords"
)

// ============================================================================
// GLOBAL HELP MANAGER
// ============================================================================

var globalHelpManager *strategies.HelpStrategyManager

func init() {
	globalHelpManager = strategies.NewHelpStrategyManager()
}

// ============================================================================
// PUBLIC API
// ============================================================================

// getCommandHelp is the main entry point for getting command help
func getCommandHelp(cmdParts []string) (string, error) {
	return globalHelpManager.GetHelp(cmdParts)
}

// splitCommand splits a full command string into parts
func splitCommand(fullCmd string) ([]string, error) {
	args, err := shellwords.Parse(fullCmd)
	if err != nil {
		return nil, nil
	}
	return args, nil
}
