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
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
	"golang.org/x/net/html"
)

// getGitCommandHelp fetches the Git documentation page for the given command
// and returns the text content of the DOM element with ID "main".
func getGitCommandHelp(command string) (string, error) {
	// Construct the URL for the specific Git command.
	url := fmt.Sprintf("https://git-scm.com/docs/git-%s", command)

	// Send the HTTP GET request.
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Ensure we received a successful response.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	// Parse the HTML document.
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the element with ID "main".
	mainNode := getElementByID(doc, "main")
	if mainNode == nil {
		return "", fmt.Errorf("element with id 'main' not found")
	}

	// Extract and return the text content.
	content := extractText(mainNode)
	return content, nil
}

// removeOverstrike removes the common overstrike pattern (a character followed by a backspace and then the same or another character) from a string.
// Ex: N\bNA\bAM\bME\bE
func removeOverstrike(input string) string {
	runes := []rune(input)
	var output []rune

	for i := 0; i < len(runes); i++ {
		// Check if the current rune is part of an overstrike sequence:
		// it should have a following backspace and then another character.
		if i+2 < len(runes) && runes[i+1] == '\b' {
			// Instead of writing both, just append the character after the backspace.
			// This effectively removes the overstrike.
			output = append(output, runes[i+2])
			i += 2 // Skip over the next two characters (backspace and the repeated character)
		} else {
			// Otherwise, just append the current rune.
			output = append(output, runes[i])
		}
	}
	return string(output)
}

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
		helpCmd := exec.Command("git", "help", subCmd)
		helpCmd.Env = append(os.Environ(), "GIT_PAGER=cat")
		if out, err := helpCmd.CombinedOutput(); err == nil {
			return removeOverstrike(string(out)), nil
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
	if baseCmd == "npm" {
		subCmd := cmdParts[1]
		if len(cmdParts) >= 2 {
			if out, err := runCmd("npm", "help", subCmd); err == nil {
				return removeOverstrike(out), nil
			}
		} else {
			if out, err := runCmd("npm", subCmd); err == nil {
				return removeOverstrike(out), nil
			}
		}
	}

	// Check if a man page exists using "man -w"
	manCheck := exec.Command("man", "-w", baseCmd)

	if err := manCheck.Run(); err == nil {
		// Run "man <command>" and pipe it through "col -b" to remove backspaces.
		manCmd := exec.Command("man", "-P", baseCmd)
		colCmd := exec.Command("cat")

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
		return removeOverstrike(buf.String()), nil
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
		return nil, nil
	}
	return args, nil
}
