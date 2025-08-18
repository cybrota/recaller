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

	// Add timeout to HTTP request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		fmt.Println("Error fetching TLDR page:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Limit response body size
	limitedReader := io.LimitReader(resp.Body, 512*1024) // 512KB limit
	body, err := io.ReadAll(limitedReader)
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

	// Helper function with configurable timeout
	runCmdWithTimeout := func(name string, timeout time.Duration, args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, name, args...)

		// Limit output size to prevent memory exhaustion
		var buf bytes.Buffer
		limitedWriter := &limitedWriter{w: &buf, limit: 1024 * 1024} // 1MB limit
		cmd.Stdout = limitedWriter
		cmd.Stderr = limitedWriter

		err := cmd.Run()
		if limitedWriter.truncated {
			return buf.String() + "\n[OUTPUT TRUNCATED - Size limit exceeded]", err
		}
		return buf.String(), err
	}

	// Helper function to run a command with timeout and size limits
	runCmd := func(name string, args ...string) (string, error) {
		return runCmdWithTimeout(name, 30*time.Second, args...)
	}

	// Special handling for Git commands:
	if baseCmd == "git" && len(cmdParts) >= 2 {

		subCmd := cmdParts[1]
		// Create git help command with proper environment
		gitHelpCmd := func() (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "git", "help", subCmd)
			cmd.Env = append(os.Environ(), "GIT_PAGER=cat")

			var buf bytes.Buffer
			limitedWriter := &limitedWriter{w: &buf, limit: 1024 * 1024}
			cmd.Stdout = limitedWriter
			cmd.Stderr = limitedWriter

			err := cmd.Run()
			if limitedWriter.truncated {
				return buf.String() + "\n[OUTPUT TRUNCATED - Size limit exceeded]", err
			}
			return buf.String(), err
		}

		if out, err := gitHelpCmd(); err == nil {
			return removeOverstrike(out), nil
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

	// Check if a man page exists using "man -w" with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manCheck := exec.CommandContext(ctx, "man", "-w", baseCmd)
	if err := manCheck.Run(); err == nil {
		// Run "man <command>" with timeout and size limits
		if output, err := runCmdWithTimeout("man", 30*time.Second, baseCmd); err == nil {
			// Handle minimal environments where man prints a placeholder message
			if strings.Contains(output, "No manual entry") || strings.Contains(output, "has been minimized") {
				return "", fmt.Errorf("man page not found for command %q", baseCmd)
			}
			return removeOverstrike(output), nil
		}
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
		return len(p), nil // Pretend we wrote it to avoid errors
	}

	remaining := lw.limit - lw.written
	if int64(len(p)) > remaining {
		lw.truncated = true
		n, err = lw.w.Write(p[:remaining])
		lw.written += int64(n)
		return len(p), err // Return original length to avoid errors
	}

	n, err = lw.w.Write(p)
	lw.written += int64(n)
	return n, err
}
