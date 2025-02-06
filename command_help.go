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
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
)

// getCommandHelp attempts to retrieve help text for a given command.
// It makes some adjustments for commands like git that use subcommands.
func getCommandHelp(cmdParts []string) (string, error) {
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	baseCmd := cmdParts[0]
	fullCmdName := strings.Join(cmdParts, " ")

	// Helper function to run a command and return its output.
	runCmd := func(name string, args ...string) (string, error) {
		cmd := exec.Command(name, args...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// Special handling for Git commands:
	if baseCmd == "git" && len(cmdParts) >= 2 {
		subCmd := cmdParts[1]
		// Try "git help <subcommand>" with GIT_PAGER=cat to force output.
		helpCmd := exec.Command("git", "help", subCmd)
		helpCmd.Env = append(os.Environ(), "GIT_PAGER=cat")
		if out, err := helpCmd.CombinedOutput(); err == nil {
			return string(out), nil
		}
		// Fallback: try "--help"
		if out, err := runCmd("git", subCmd, "--help"); err == nil {
			return out, nil
		}
		// Fallback: try "-h"
		if out, err := runCmd("git", subCmd, "-h"); err == nil {
			return out, nil
		}
		return "", fmt.Errorf("failed to get help for command %q", fullCmdName)
	}

	// Special handling for Go commands:
	if baseCmd == "go" && len(cmdParts) >= 2 {
		subCmd := cmdParts[1]
		if out, err := runCmd("go", "help", subCmd); err == nil {
			return out, nil
		}
		return "The selected command is invalid", nil
	}

	// Special handling for kubectl commands:
	if baseCmd == "kubectl" && len(cmdParts) >= 2 {
		subCmd := cmdParts[1]
		if out, err := runCmd("kubectl", subCmd, "--help"); err == nil {
			return out, nil
		}
	}

	// Special handling for cargo commands:
	if baseCmd == "cargo" && len(cmdParts) >= 2 {
		subCmd := cmdParts[1]
		if out, err := runCmd("cargo", subCmd, "--help"); err == nil {
			return out, nil
		}
	}

	// Special handling for npm commands:
	if baseCmd == "npm" && len(cmdParts) >= 2 {
		subCmd := cmdParts[1]
		if out, err := runCmd("npm", "help", subCmd); err == nil {
			return out, nil
		}
	}

	// Check if a man page exists using "man -w"
	manCheck := exec.Command("man", "-w", baseCmd)
	if err := manCheck.Run(); err == nil {
		// Run "man <command>" and pipe it through "col -b" to remove backspaces.
		manCmd := exec.Command("man", baseCmd)
		colCmd := exec.Command("col", "-b")

		// Pipe the output of manCmd into colCmd.
		pipeReader, pipeWriter := io.Pipe()
		manCmd.Stdout = pipeWriter
		colCmd.Stdin = pipeReader

		var buf bytes.Buffer
		colCmd.Stdout = &buf

		// Start both commands.
		if err := manCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start man command: %v", err)
		}
		if err := colCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start col command: %v", err)
		}
		// Wait for the man command to finish, then close the writer.
		if err := manCmd.Wait(); err != nil {
			return "", fmt.Errorf("man command failed: %v", err)
		}
		pipeWriter.Close()
		// Wait for the col command to finish.
		if err := colCmd.Wait(); err != nil {
			return "", fmt.Errorf("col command failed: %v", err)
		}
		return buf.String(), nil
	}

	// For other commands, try common help flags.
	tryHelp := func(flag string) (string, error) {
		args := append(cmdParts[1:], flag)
		return runCmd(baseCmd, args...)
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

// splitCommand splits a full command string into parts.
func splitCommand(fullCmd string) ([]string, error) {
	args, err := shellwords.Parse(fullCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command %q: %v", fullCmd, err)
	}
	return args, nil
}
