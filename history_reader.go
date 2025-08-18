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
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("zsh history file not found. Run some commands in zsh to create %s, then try again", zshHistoryPath)
		}
		return nil, err
	}
	defer file.Close()

	// Pre-allocate history slice with estimated capacity
	var history []HistoryEntry
	if stat, err := file.Stat(); err == nil {
		// Estimate ~50 bytes per line average
		estimatedLines := int(stat.Size() / 50)
		history = make([]HistoryEntry, 0, estimatedLines)
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for better performance with large history files
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
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
// Run `history -w` to store history to .bash_history file (or) close the shell and re-launch
// in ~/.bash_profile to read epoch timestamps correctly
func readBashHistoryWithEpoch() ([]HistoryEntry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	historyPath := filepath.Join(homeDir, ".bash_history")

	file, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("bash history file not found. Run 'history -w' to create %s, then try again", historyPath)
		}
		return nil, err
	}
	defer file.Close()

	// Pre-allocate history slice with estimated capacity
	var history []HistoryEntry
	if stat, err := file.Stat(); err == nil {
		// Estimate ~30 bytes per line average for bash
		estimatedLines := int(stat.Size() / 30)
		history = make([]HistoryEntry, 0, estimatedLines)
	}
	var lastTimestamp *time.Time

	scanner := bufio.NewScanner(file)
	// Increase buffer size for better performance with large history files
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
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
		// Default to bash when SHELL is not set
		return "bash", nil
	}

	// Extract shell name from executable path (e.g., "/bin/zsh" -> "zsh")
	currentShell := filepath.Base(currentShellPath)
	return currentShell, nil
}

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

	if err != nil {
		return err
	}

	// Optimize: Pre-allocate frequency map with estimated capacity
	// and track most recent timestamp per command for efficiency
	freqMap := make(map[string]int, len(history)/4) // Estimate unique commands
	lastTimestamp := make(map[string]*time.Time, len(history)/4)

	// Process history in reverse to get most recent timestamps efficiently
	for i := len(history) - 1; i >= 0; i-- {
		hist := history[i]
		if hist.Timestamp != nil && hist.Command != "" {
			// Update frequency count
			freqMap[hist.Command]++

			// Keep only the most recent timestamp per command
			if lastTimestamp[hist.Command] == nil || hist.Timestamp.After(*lastTimestamp[hist.Command]) {
				lastTimestamp[hist.Command] = hist.Timestamp
			}
		}
	}

	// Insert into AVL tree with optimized metadata (single pass)
	for command, frequency := range freqMap {
		metadata := CommandMetadata{
			Command:   command,
			Timestamp: lastTimestamp[command],
			Frequency: frequency,
		}
		tree.Insert(command, metadata)
	}

	return nil
}
