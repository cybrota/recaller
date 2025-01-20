// history_reader.go

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
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HistoryEntry holds the optional timestamp and the command
type HistoryEntry struct {
	Command   string
	Timestamp *time.Time
}

// readZshHistoryWithEpoch reads ~/.zsh_history file.
func readZshHistoryWithEpoch() ([]HistoryEntry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	zshHistoryPath := filepath.Join(homeDir, ".zsh_history")

	file, err := os.Open(zshHistoryPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var history []HistoryEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, ": ") {
			// line doesn't have a zsh metadata prefix, might be older or partial
			// So just treat it as a plain command
			history = append(history, HistoryEntry{Timestamp: nil, Command: line})
			continue
		}

		// Example line: ": 1673291850:0;ls -la"
		// Break on the first 2 colons (split into 3 parts)
		parts := strings.SplitN(line, ":", 3)
		// parts[0] = ""
		// parts[1] = " 1673291850"
		// parts[2] = "0;ls -la"

		if len(parts) < 3 {
			// If the format is unexpected, skip
			continue
		}

		// Clean up the timestamp part
		timeStr := strings.TrimSpace(parts[1]) // "1673291850"

		epoch, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			// If we fail, skip or store nil timestamp
			history = append(history, HistoryEntry{Timestamp: nil, Command: line})
			continue
		}
		t := time.Unix(epoch, 0)

		// The command part will be in parts[2], but it has "0;" or "1;" etc. at the beginning
		// We can split at the semicolon
		subParts := strings.SplitN(parts[2], ";", 2)
		// subParts[0] = "0"  (the return status or extended info)
		// subParts[1] = "ls -la"
		if len(subParts) < 2 {
			// No command found
			history = append(history, HistoryEntry{Timestamp: &t, Command: ""})
			continue
		}

		command := subParts[1]
		history = append(history, HistoryEntry{Timestamp: &t, Command: command})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

// readBashHistoryWithEpoch reads ~/.bash_history file.
// Set export HISTTIMEFORMAT="%s "
// in ~/.bash_profile to read epoch timestamps correctly
func readBashHistoryWithEpoch() ([]HistoryEntry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	historyPath := filepath.Join(homeDir, ".bash_history")

	file, err := os.Open(historyPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var history []HistoryEntry
	var lastTimestamp *time.Time

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Lines starting with '#' are epoch timestamps if HISTTIMEFORMAT was ever enabled
		if strings.HasPrefix(line, "#") {
			// Remove '#' and parse the remainder as an integer (epoch seconds)
			epochStr := strings.TrimPrefix(line, "#")
			epochStr = strings.TrimSpace(epochStr)
			epoch, err := strconv.ParseInt(epochStr, 10, 64)
			if err == nil {
				t := time.Unix(epoch, 0)
				lastTimestamp = &t
			} else {
				lastTimestamp = nil
			}
		} else {
			// This line is a command
			entry := HistoryEntry{
				Timestamp: lastTimestamp,
				Command:   line,
			}
			history = append(history, entry)
			// Reset the timestamp so it won't affect subsequent commands
			lastTimestamp = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

// detectCurrentShell detects the type of Unix shell: Bash, Zshell etc.
func detectCurrentShell() (string, error) {
	currentShellPath, ok := os.LookupEnv("SHELL")
	if !ok {
		return "", fmt.Errorf("SHELL environment variable not set")
	}

	// Extract shell name from executable path (e.g., "/bin/zsh" -> "zsh")
	currentShell := filepath.Base(currentShellPath)
	return currentShell, nil
}

// readHistoryAndPopulateTree reads existing history into memory as an AVL tree
func readHistoryAndPopulateTree(tree *AVLTree) error {
	s, err := detectCurrentShell()
	if err != nil {
		log.Fatalf("Error while resolving the path: %v", err)
	}

	var history []HistoryEntry
	switch s {
	case "zsh":
		history, err = readZshHistoryWithEpoch()
	case "bash":
		history, err = readBashHistoryWithEpoch()
	default:
		log.Fatalf("Unknown shell: %s detected. Aborting.", s)
	}

	for _, hist := range history {
		tree.Insert(hist.Command, hist.Timestamp)
	}

	return nil
}
