// command_help.go

/**
 * Copyright (C) Naren Yellavula - All Rights Reserved
 *
 * This source code is protected under international copyright law.  All rights
 * reserved and protected by the copyright holders.
 * This file is confidential and only available to authorized individuals with the
 * permission of the copyright holders.  If you encounter this file and do not have
 * permission, please contact the copyright holders and delete this file.
 */

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
)

// getCommandHelp attempts to retrieve help text for a given command.
// For git commands, if the base command is "git" and a subcommand is provided,
// it adjusts the arguments accordingly.
func getCommandHelp(cmdParts []string) (string, error) {
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	baseCmd := cmdParts[0]

	// Create the full command name for logging or man page lookup.
	fullCmdName := strings.Join(cmdParts, " ")

	// For Git commands (or other commands where help flags might conflict with file arguments),
	// it's best to avoid passing file arguments.
	// If the base command is "git" and there are additional elements, assume the first subcommand is the target.
	// For example, given ["git", "add", "command_help.go", "main.go"], we discard file arguments when showing help.
	if baseCmd == "git" && len(cmdParts) >= 2 {
		// Extract the subcommand from Git.
		gitSubCmd := cmdParts[1]
		// Use "git help <subcommand>" instead.
		helpCmd := exec.Command("git", "help", gitSubCmd)
		out, err := helpCmd.Output()
		if err == nil {
			return string(out), nil
		}
		// Fall back to using help flags if "git help <subcommand>" fails.
		// Try with "--help".
		helpCmd = exec.Command("git", gitSubCmd, "--help")
		out, err = helpCmd.Output()
		if err == nil {
			return string(out), nil
		}
		// Lastly, try with "-h".
		helpCmd = exec.Command("git", gitSubCmd, "-h")
		out, err = helpCmd.Output()
		if err == nil {
			return string(out), nil
		}
		return "", fmt.Errorf("failed to get help for command %q", fullCmdName)
	}

	// For Go commands
	if baseCmd == "go" && len(cmdParts) >= 2 {
		// Extract the subcommand from Go.
		gitSubCmd := cmdParts[1]
		// Use "go help <subcommand>" instead.
		helpCmd := exec.Command("go", "help", gitSubCmd)
		out, err := helpCmd.Output()
		if err == nil {
			return string(out), nil
		} else {
			return "The selected command is invalid", nil
		}
	}

	// For Kubectl commands
	if baseCmd == "kubectl" && len(cmdParts) >= 2 {
		// Extract the subcommand from Go.
		gitSubCmd := cmdParts[1]
		// Use "go help <subcommand>" instead.
		helpCmd := exec.Command("kubectl", gitSubCmd, "--help")
		out, err := helpCmd.Output()
		if err == nil {
			return string(out), nil
		}
	}

	// For Rust commands
	if baseCmd == "cargo" && len(cmdParts) >= 2 {
		// Extract the subcommand from Go.
		gitSubCmd := cmdParts[1]
		// Use "go help <subcommand>" instead.
		helpCmd := exec.Command("cargo", gitSubCmd, "--help")
		out, err := helpCmd.Output()
		if err == nil {
			return string(out), nil
		}
	}

	// For NPM commands
	if baseCmd == "npm" && len(cmdParts) >= 2 {
		// Extract the subcommand from Go.
		gitSubCmd := cmdParts[1]
		// Use "go help <subcommand>" instead.
		helpCmd := exec.Command("npm", "help", gitSubCmd)
		out, err := helpCmd.Output()
		if err == nil {
			return string(out), nil
		}
	}

	// Check for a man page using "man -w"
	checkMan := exec.Command("man", "-w", baseCmd)
	if err := checkMan.Run(); err == nil {
		// If a man page is found, we run "man <fullCmdName>" and pipe it through "col -b"
		manCmd := exec.Command("man", baseCmd)
		colCmd := exec.Command("col", "-b")
		r, w := io.Pipe()
		manCmd.Stdout = w
		colCmd.Stdin = r

		var buf bytes.Buffer
		colCmd.Stdout = &buf

		if err := manCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start man command: %v", err)
		}
		if err := colCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start col command: %v", err)
		}
		if err := manCmd.Wait(); err != nil {
			return "", fmt.Errorf("man command failed: %v", err)
		}
		if err := w.Close(); err != nil {
			return "", fmt.Errorf("failed to close pipe to col: %v", err)
		}
		if err := colCmd.Wait(); err != nil {
			return "", fmt.Errorf("col command failed: %v", err)
		}
		return buf.String(), nil
	}

	// For non-Git commands or single-root commands, try help flags.
	tryHelp := func(flag string) (string, error) {
		args := append(cmdParts[1:], flag)
		helpCmd := exec.Command(baseCmd, args...)
		out, err := helpCmd.Output()
		return string(out), err
	}

	if out, err := tryHelp("-h"); err == nil {
		return out, nil
	}
	if out, err := tryHelp("--help"); err == nil {
		return out, nil
	}
	if out, err := tryHelp("help"); err == nil {
		return out, nil
	}

	return "", fmt.Errorf("no help found for command %q", fullCmdName)
}

// extractCommandName fetches a command from full-command
func splitCommand(fullCmd string) ([]string, error) {
	// Split on whitespace
	args, err := shellwords.Parse(fullCmd)
	if err != nil {
		errors.New(fmt.Sprintf("failed to parse command: %s", fullCmd))
	}
	return args, nil
}
