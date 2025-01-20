// command_help_test.gp

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
	"strings"
	"testing"
)

func TestGetCommandHelp_ManPageAvailable(t *testing.T) {
	cmdName := "ls" // Choose a command likely to have a man page
	help, err := getCommandHelp(cmdName)
	if err != nil {
		t.Errorf("Unexpected error for %q: %v", cmdName, err)
	}
	if len(help) == 0 {
		t.Errorf("Expected non-empty help string for %q, got empty string", cmdName)
	}
}

func TestGetCommandHelp_NoManPageButHasHelpFlag(t *testing.T) {
	// Choose a command that doesn't have a man page but supports -h or --help
	cmdName := "aws" // Replace with an actual command for testing
	help, err := getCommandHelp(cmdName)
	if err != nil && !strings.Contains(err.Error(), "no help found") {
		t.Errorf("Unexpected error for %q: %v", cmdName, err)
	}
	if len(help) > 0 {
		t.Logf("Found help for %q, but expected none. Help: %s", cmdName, help)
	}
}

func TestGetCommandHelp_CommandDoesNotExist(t *testing.T) {
	cmdName := "nonexistentcommand123"
	_, err := getCommandHelp(cmdName)
	if err == nil {
		t.Errorf("Expected error for nonexistent command %q, got nil", cmdName)
	}
}

func TestExtractCommandName_EmptyString(t *testing.T) {
	fullCmd := ""
	name := extractCommandName(fullCmd)
	if name != "" {
		t.Errorf("Expected empty string for command name, got: %q", name)
	}
}

func TestExtractCommandName_SingleWordCommand(t *testing.T) {
	fullCmd := "ls"
	name := extractCommandName(fullCmd)
	if name != fullCmd {
		t.Errorf("Expected command name %q, got: %q", fullCmd, name)
	}
}

func TestExtractCommandName_CommandWithArgs(t *testing.T) {
	fullCmd := "ls -l /tmp"
	name := extractCommandName(fullCmd)
	expected := "ls"
	if name != expected {
		t.Errorf("Expected command name %q, got: %q", expected, name)
	}
}
