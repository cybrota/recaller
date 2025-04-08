// command_help.go

/**
 * Copyright 2025 (C) Naren Yellavula - All Rights Reserved
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
)

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

func getCOmmandHelpTLDRPage(cmdParts []string) (string, error) {
	baseCmd := cmdParts[0]
	baseUrl := "https://raw.githubusercontent.com/tldr-pages/tldr/refs/heads/main/pages/common"
	fullURL := ""

	if len(cmdParts) >= 2 {
		subCmd := cmdParts[1]
		fullURL = fmt.Sprintf("%s/%s-%s.md", baseUrl, baseCmd, subCmd)
	} else {
		fullURL = fmt.Sprintf("%s/%s.md", baseUrl, baseCmd)
	}

	resp, err := http.Get(fullURL)
	if err != nil {
		fmt.Println("Error fetching TLDR page:", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", err
	}

	return string(body), nil
}

// getCommandHelp attempts to retrieve help text for a given command.
// It makes some adjustments for commands like git that use subcommands.
func getCommandHelp(cmdParts []string) (string, error) {
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	// out, err := getCOmmandHelpTLDRPage(cmdParts)

	// if err != nil {
	// 	return "Cannot fetch TLDR doc", err
	// }

	// if out != "" {
	// 	return out, nil
	// }

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

	if baseCmd == "aws" {
		if len(cmdParts) >= 2 {
			subCmd := cmdParts[1]
			if out, err := runCmd("aws", subCmd, "help"); err == nil {
				return removeOverstrike(out), nil
			}
		} else {
			return "", fmt.Errorf("Given AWS command is invalid")
		}
	}

	// Check if a man page exists using "man -w"
	manCheck := exec.Command("man", "-w", baseCmd)
	if err := manCheck.Run(); err == nil {
		// Run "man <command>"
		manCmd := exec.Command("man", baseCmd)

		var buf bytes.Buffer
		manCmd.Stdout = &buf
		// Start both commands.
		if err := manCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start man command: %v", err)
		}
		// Wait for the man command to finish, then close the writer.
		if err := manCmd.Wait(); err != nil {
			return "", fmt.Errorf("man command failed: %v", err)
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

	return "", fmt.Errorf("\nNo help found for command %q", fullCmdName)
}

// splitCommand splits a full command string into parts.
func splitCommand(fullCmd string) ([]string, error) {
	args, err := shellwords.Parse(fullCmd)
	if err != nil {
		return nil, nil
	}
	return args, nil
}
