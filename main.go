// main.go

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
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

const asciiLogo = `
██████╗ ███████╗ ██████╗ █████╗ ██╗     ██╗     ███████╗██████╗
██╔══██╗██╔════╝██╔════╝██╔══██╗██║     ██║     ██╔════╝██╔══██╗
██████╔╝█████╗  ██║     ███████║██║     ██║     █████╗  ██████╔╝
██╔══██╗██╔══╝  ██║     ██╔══██║██║     ██║     ██╔══╝  ██╔══██╗
██║  ██║███████╗╚██████╗██║  ██║███████╗███████╗███████╗██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝

Copyright @ Naren Yellavula (https://github.com/narenaryan)
`

func main() {
	var cmdRun = &cobra.Command{
		Use:   "run",
		Short: "Launches recaller UI for search & documentation",
		Long:  fmt.Sprintf("%s\n%s", asciiLogo, `Run command opens Recaller UI with search from history`),
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// Parse the command-line flags
			helpCache := NewOptimizedHelpCache()

			tree := NewAVLTree()
			if err := readHistoryAndPopulateTree(tree); err != nil {
				log.Fatalf("Error reading history: %v", err)
			}
			run(tree, helpCache)
		},
	}

	var cmdUsage = &cobra.Command{
		Use:   "usage",
		Short: "Print Recaller usage guide",
		Long:  fmt.Sprintf("%s\n%s", asciiLogo, `Usage displays the recaller CLI usage guide`),
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(getHelpMessage())
		},
	}

	var cmdHistory = &cobra.Command{
		Use:   "history",
		Short: "Print Recaller usage guide",
		Long:  fmt.Sprintf("%s\n%s", asciiLogo, "Suggest list of past %d most frequently used commands"),
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			// Parse the command-line flags
			tree := NewAVLTree()
			if err := readHistoryAndPopulateTree(tree); err != nil {
				log.Fatalf("Error reading history: %v", err)
			}
			res := getSuggestions(cmd.Flag("match").Value.String(), tree)
			fmt.Println(strings.Join(res, "\n"))
		},
	}

	cmdHistory.Flags().String("match", "", "match string prefix to look in history")

	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Print Recaller version",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}

	var rootCmd = &cobra.Command{
		Use:  "recaller",
		Long: asciiLogo,
		Run: func(cmd *cobra.Command, args []string) {
			// Default to run command when no subcommand is provided
			helpCache := NewOptimizedHelpCache()

			tree := NewAVLTree()
			if err := readHistoryAndPopulateTree(tree); err != nil {
				log.Fatalf("Error reading history: %v", err)
			}
			run(tree, helpCache)
		},
	}
	rootCmd.AddCommand(cmdRun, cmdUsage, cmdVersion, cmdHistory)
	rootCmd.Execute()
}
