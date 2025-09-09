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

// HelpStrategyManager manages different help strategies
type HelpStrategyManager struct {
	strategies []HelpStrategy
	cmdRunner  *CommandRunner
}

// NewHelpStrategyManager creates a new strategy manager with all strategies
func NewHelpStrategyManager() *HelpStrategyManager {
	cmdRunner := NewCommandRunner()

	manager := &HelpStrategyManager{
		cmdRunner: cmdRunner,
	}

	// Register strategies in order of preference
	// TLDR is registered first as it provides cleaner, more practical examples
	manager.RegisterStrategy(&TldrStrategy{})
	manager.RegisterStrategy(NewGitHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewGoHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewKubectlHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewCargoHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewNpmHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewAwsHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewDockerHelpStrategy(cmdRunner))
	manager.RegisterStrategy(NewManPageStrategy(cmdRunner))
	manager.RegisterStrategy(NewGenericHelpStrategy(cmdRunner))

	return manager
}

// RegisterStrategy registers a new help strategy
func (hsm *HelpStrategyManager) RegisterStrategy(strategy HelpStrategy) {
	hsm.strategies = append(hsm.strategies, strategy)
}

// GetHelp gets help for a command using the best available strategy
func (hsm *HelpStrategyManager) GetHelp(cmdParts []string) (string, error) {
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmd := NewCommand(cmdParts)

	// Try TLDR first as it provides cleaner, more practical examples
	tldrStrategy := &TldrStrategy{}
	if help, err := tldrStrategy.GetHelp(cmdParts); err == nil && help != "" {
		return help, nil
	}

	// Find other strategies that support this command (excluding TLDR since we tried it first)
	var supportedStrategies []HelpStrategy
	for _, strategy := range hsm.strategies {
		if _, isTldr := strategy.(*TldrStrategy); isTldr {
			continue // Skip TLDR since we already tried it
		}
		if strategy.SupportsCommand(cmd.BaseCmd) {
			supportedStrategies = append(supportedStrategies, strategy)
		}
	}

	// Try strategies in priority order
	var lastErr error
	for _, strategy := range supportedStrategies {
		if help, err := strategy.GetHelp(cmdParts); err == nil && help != "" {
			return help, nil
		} else {
			lastErr = err
		}
	}

	if len(supportedStrategies) == 0 && lastErr == nil {
		return "", fmt.Errorf("no help strategy found for command %q", cmd.FullName)
	}

	return "", fmt.Errorf("failed to get help for command %q: %v", cmd.FullName, lastErr)
}
