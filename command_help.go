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
	"os/exec"
	"strings"
)

// getCommandHelp attempts to retrieve help text for cmdName.
// 1) Checks if there is a man page via "man -w cmdName".
// 2) If no man page is found, tries "cmdName -h".
// 3) If that fails, tries "cmdName --help".
// Returns the help text or an error if none of the above produce results.

func getCommandHelp(cmdName string) (string, error) {
	// 1) Check if there's a man page via "man -w cmdName".
	checkMan := exec.Command("man", "-w", cmdName)
	if err := checkMan.Run(); err == nil {
		// Man page is available, so run "man cmdName" and pipe it through "col -b"
		// to remove backspaces and overstriking.
		manCmd := exec.Command("man", cmdName)
		colCmd := exec.Command("col", "-b")

		// Pipe: manCmd.Stdout -> colCmd.Stdin
		// Then capture colCmd.Stdout into a buffer.
		r, w := io.Pipe()
		manCmd.Stdout = w
		colCmd.Stdin = r

		var buf bytes.Buffer
		colCmd.Stdout = &buf

		// Start both commands.
		if err := manCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start man command: %v", err)
		}
		if err := colCmd.Start(); err != nil {
			return "", fmt.Errorf("failed to start col command: %v", err)
		}

		// Wait for manCmd to finish, then close the writer so colCmd sees EOF.
		if err := manCmd.Wait(); err != nil {
			return "", fmt.Errorf("man command failed: %v", err)
		}
		if err := w.Close(); err != nil {
			return "", fmt.Errorf("failed to close pipe to col: %v", err)
		}

		// Now wait for colCmd to finish, which will fill our buffer.
		if err := colCmd.Wait(); err != nil {
			return "", fmt.Errorf("col command failed: %v", err)
		}

		return buf.String(), nil
	}

	// 2) No man page found. Try "cmdName -h".
	helpCmd := exec.Command(cmdName, "-h")
	out, err := helpCmd.Output()
	if err == nil {
		return string(out), nil
	}

	// 3) If that fails, try "cmdName --help".
	helpCmd = exec.Command(cmdName, "--help")
	out, err = helpCmd.Output()
	if err == nil {
		return string(out), nil
	}

	// 4) No -h or --help options found. Try "cmdName help".
	helpCmd = exec.Command(cmdName, "help")
	out, err = helpCmd.Output()
	if err == nil {
		return string(out), nil
	}

	// Otherwise, we have no help text to display.
	return "", fmt.Errorf("no help found for command %q", cmdName)
}

// extractCommandName fetches a command from full-command
func extractCommandName(fullCmd string) string {
	// Split on whitespace
	parts := strings.Fields(fullCmd)
	if len(parts) == 0 {
		// No tokens at all (empty string, or just whitespace)
		return ""
	}
	return parts[0] // The command name
}
