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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mattn/go-shellwords"
)

// ============================================================================
// CONSTANTS AND CONFIGURATION
// ============================================================================

const (
	defaultCmdTimeout = 30 * time.Second
	fastCmdTimeout    = 5 * time.Second
	gitCmdTimeout     = 15 * time.Second
	httpTimeout       = 10 * time.Second
	maxOutputSize     = 1024 * 1024 // 1MB
	maxTldrSize       = 512 * 1024  // 512KB
)

// ============================================================================
// STRATEGY PATTERN INTERFACES
// ============================================================================

// HelpStrategy defines the interface for different command help strategies
type HelpStrategy interface {
	GetHelp(cmdParts []string) (string, error)
	SupportsCommand(baseCmd string) bool
	Priority() int // Lower number = higher priority
}

// Command represents a parsed command with its parts
type Command struct {
	Parts    []string
	BaseCmd  string
	SubCmds  []string
	FullName string
}

// NewCommand creates a new Command from command parts
func NewCommand(parts []string) *Command {
	if len(parts) == 0 {
		return &Command{Parts: parts}
	}

	return &Command{
		Parts:    parts,
		BaseCmd:  parts[0],
		SubCmds:  parts[1:],
		FullName: strings.Join(parts, " "),
	}
}

// HasSubCommand checks if command has at least n sub-commands
func (c *Command) HasSubCommand(n int) bool {
	return len(c.SubCmds) >= n
}

// GetSubCommand returns the nth sub-command (0-indexed)
func (c *Command) GetSubCommand(n int) string {
	if n >= len(c.SubCmds) {
		return ""
	}
	return c.SubCmds[n]
}

// ============================================================================
// HELP STRATEGY MANAGER
// ============================================================================

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
	manager.RegisterStrategy(&GitHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&GoHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&KubectlHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&CargoHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&NpmHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&AwsHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&DockerHelpStrategy{cmdRunner})
	manager.RegisterStrategy(&ManPageStrategy{cmdRunner})
	manager.RegisterStrategy(&GenericHelpStrategy{cmdRunner})

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

// ============================================================================
// COMMAND RUNNER UTILITY
// ============================================================================

// CommandRunner handles command execution with timeouts and size limits
type CommandRunner struct{}

// NewCommandRunner creates a new command runner
func NewCommandRunner() *CommandRunner {
	return &CommandRunner{}
}

// RunWithTimeout runs a command with specified timeout and size limit
func (cr *CommandRunner) RunWithTimeout(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	var buf bytes.Buffer
	limitedWriter := &limitedWriter{w: &buf, limit: maxOutputSize}
	cmd.Stdout = limitedWriter
	cmd.Stderr = limitedWriter

	err := cmd.Run()
	result := buf.String()

	if limitedWriter.truncated {
		result += "\n[OUTPUT TRUNCATED - Size limit exceeded]"
	}

	return result, err
}

// Run runs a command with default timeout
func (cr *CommandRunner) Run(name string, args ...string) (string, error) {
	return cr.RunWithTimeout(defaultCmdTimeout, name, args...)
}

// RunFast runs a command with short timeout
func (cr *CommandRunner) RunFast(name string, args ...string) (string, error) {
	return cr.RunWithTimeout(fastCmdTimeout, name, args...)
}

// CheckCommandExists checks if a command exists using "which" or similar
func (cr *CommandRunner) CheckCommandExists(cmd string) bool {
	_, err := cr.RunFast("which", cmd)
	return err == nil
}

// ============================================================================
// CONCRETE HELP STRATEGIES
// ============================================================================

// TldrStrategy fetches help from TLDR pages - prioritized for cleaner examples
type TldrStrategy struct{}

func (t *TldrStrategy) SupportsCommand(baseCmd string) bool {
	return true // Supports any command as it's a universal fallback
}

func (t *TldrStrategy) Priority() int {
	return 0 // Highest priority - try first for better user experience
}

func (t *TldrStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	baseUrl := "https://raw.githubusercontent.com/tldr-pages/tldr/refs/heads/main/pages/common"
	var fullURL string

	// Support up to 2 levels of sub-commands for TLDR
	if cmd.HasSubCommand(1) {
		subCmd := cmd.GetSubCommand(0)
		fullURL = fmt.Sprintf("%s/%s-%s.md", baseUrl, cmd.BaseCmd, subCmd)
	} else {
		fullURL = fmt.Sprintf("%s/%s.md", baseUrl, cmd.BaseCmd)
	}

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch TLDR page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TLDR page not found (HTTP %d)", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, maxTldrSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read TLDR response: %v", err)
	}

	content := string(body)
	if content != "" {
		content = "📚 TLDR Documentation:\n\n" + content
	}

	return content, nil
}

// GitHelpStrategy handles Git commands with up to 3 levels of sub-commands
type GitHelpStrategy struct {
	cmdRunner *CommandRunner
}

func (g *GitHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "git"
}

func (g *GitHelpStrategy) Priority() int {
	return 2
}

func (g *GitHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return g.cmdRunner.RunWithTimeout(gitCmdTimeout, "git", "help")
	}

	// Handle git subcommand help
	subCmd := cmd.GetSubCommand(0)

	// Try git help <subcommand> first
	if out, err := g.runGitHelp(subCmd); err == nil {
		return removeOverstrike(out), nil
	}

	// For complex sub-commands like "git config --global", try git <subcommand> --help
	if cmd.HasSubCommand(2) {
		args := append(cmd.SubCmds, "--help")
		if out, err := g.cmdRunner.RunWithTimeout(gitCmdTimeout, "git", args...); err == nil {
			return removeOverstrike(out), nil
		}
	}

	return "", fmt.Errorf("failed to get Git help for %q", cmd.FullName)
}

func (g *GitHelpStrategy) runGitHelp(subCmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "help", subCmd)
	cmd.Env = append(os.Environ(), "GIT_PAGER=cat")

	var buf bytes.Buffer
	limitedWriter := &limitedWriter{w: &buf, limit: maxOutputSize}
	cmd.Stdout = limitedWriter
	cmd.Stderr = limitedWriter

	err := cmd.Run()
	result := buf.String()

	if limitedWriter.truncated {
		result += "\n[OUTPUT TRUNCATED - Size limit exceeded]"
	}

	return result, err
}

// GoHelpStrategy handles Go commands
type GoHelpStrategy struct {
	cmdRunner *CommandRunner
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

// KubectlHelpStrategy handles kubectl commands with sub-commands
type KubectlHelpStrategy struct {
	cmdRunner *CommandRunner
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

// CargoHelpStrategy handles Cargo commands
type CargoHelpStrategy struct {
	cmdRunner *CommandRunner
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

// NpmHelpStrategy handles npm commands
type NpmHelpStrategy struct {
	cmdRunner *CommandRunner
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
		return removeOverstrike(out), nil
	}

	// Fallback to npm <subcommand> --help
	return n.cmdRunner.Run("npm", subCmd, "--help")
}

// AwsHelpStrategy handles AWS CLI commands with multiple sub-command levels
type AwsHelpStrategy struct {
	cmdRunner *CommandRunner
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
		return removeOverstrike(out), nil
	}

	return "", fmt.Errorf("AWS command %q is invalid or not found", cmd.FullName)
}

// DockerHelpStrategy handles Docker commands
type DockerHelpStrategy struct {
	cmdRunner *CommandRunner
}

func (d *DockerHelpStrategy) SupportsCommand(baseCmd string) bool {
	return baseCmd == "docker"
}

func (d *DockerHelpStrategy) Priority() int {
	return 2
}

func (d *DockerHelpStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	if !cmd.HasSubCommand(1) {
		return d.cmdRunner.Run("docker", "--help")
	}

	// Handle docker subcommand help
	args := append(cmd.SubCmds, "--help")
	return d.cmdRunner.Run("docker", args...)
}

// ManPageStrategy handles standard man pages
type ManPageStrategy struct {
	cmdRunner *CommandRunner
}

func (m *ManPageStrategy) SupportsCommand(baseCmd string) bool {
	// Check if man page exists
	ctx, cancel := context.WithTimeout(context.Background(), fastCmdTimeout)
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
		return removeOverstrike(output), nil
	}

	return "", fmt.Errorf("failed to get man page for %q", cmd.BaseCmd)
}

// GenericHelpStrategy tries common help flags
type GenericHelpStrategy struct {
	cmdRunner *CommandRunner
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

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// removeOverstrike removes the common overstrike pattern from strings
func removeOverstrike(input string) string {
	runes := []rune(input)
	var output []rune

	for i := 0; i < len(runes); i++ {
		// Check if the current rune is part of an overstrike sequence
		if i+2 < len(runes) && runes[i+1] == '\b' {
			output = append(output, runes[i+2])
			i += 2
		} else {
			output = append(output, runes[i])
		}
	}
	return string(output)
}

// splitCommand splits a full command string into parts
func splitCommand(fullCmd string) ([]string, error) {
	args, err := shellwords.Parse(fullCmd)
	if err != nil {
		return nil, nil
	}
	return args, nil
}

// limitedWriter implements io.Writer with size limiting
type limitedWriter struct {
	w         io.Writer
	limit     int64
	written   int64
	truncated bool
}

func (lw *limitedWriter) Write(p []byte) (n int, err error) {
	if lw.written >= lw.limit {
		lw.truncated = true
		return len(p), nil
	}

	remaining := lw.limit - lw.written
	if int64(len(p)) > remaining {
		lw.truncated = true
		n, err = lw.w.Write(p[:remaining])
		lw.written += int64(n)
		return len(p), err
	}

	n, err = lw.w.Write(p)
	lw.written += int64(n)
	return n, err
}

// ============================================================================
// PUBLIC API
// ============================================================================

var globalHelpManager *HelpStrategyManager

func init() {
	globalHelpManager = NewHelpStrategyManager()
}

// getCommandHelp is the main entry point for getting command help
func getCommandHelp(cmdParts []string) (string, error) {
	return globalHelpManager.GetHelp(cmdParts)
}
