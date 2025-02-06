package main

import (
	"testing"
)

// TestGetCommandHelpEmpty verifies that calling getCommandHelp with an empty slice returns an error.
func TestGetCommandHelpEmpty(t *testing.T) {
	_, err := getCommandHelp([]string{})
	if err == nil {
		t.Error("Expected error for empty command slice, got nil")
	}
}

// TestGetCommandHelpNonexistent verifies that a non-existent command returns an error.
func TestGetCommandHelpNonexistent(t *testing.T) {
	// Use a command name that is very unlikely to exist.
	_, err := getCommandHelp([]string{"nonexistent_command"})
	if err == nil {
		t.Error("Expected error for nonexistent command, got nil")
	}
}

// TestSplitCommand verifies that splitCommand correctly tokenizes a command string.
func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"go build", []string{"go", "build"}},
		{`git commit -m "Initial commit"`, []string{"git", "commit", "-m", "Initial commit"}},
		{"npm install package", []string{"npm", "install", "package"}},
		{`echo "hello world"`, []string{"echo", "hello world"}},
	}

	for _, tc := range tests {
		parts, err := splitCommand(tc.input)
		if err != nil {
			t.Errorf("splitCommand(%q) returned error: %v", tc.input, err)
			continue
		}
		if len(parts) != len(tc.expected) {
			t.Errorf("splitCommand(%q): expected %v, got %v", tc.input, tc.expected, parts)
			continue
		}
		for i := range parts {
			if parts[i] != tc.expected[i] {
				t.Errorf("splitCommand(%q): expected %v, got %v", tc.input, tc.expected, parts)
				break
			}
		}
	}
}
