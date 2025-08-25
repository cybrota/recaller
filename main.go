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
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	asciiLogo := `
██████╗ ███████╗ ██████╗ █████╗ ██╗     ██╗     ███████╗██████╗
██╔══██╗██╔════╝██╔════╝██╔══██╗██║     ██║     ██╔════╝██╔══██╗
██████╔╝█████╗  ██║     ███████║██║     ██║     █████╗  ██████╔╝
██╔══██╗██╔══╝  ██║     ██╔══██║██║     ██║     ██╔══╝  ██╔══██╗
██║  ██║███████╗╚██████╗██║  ██║███████╗███████╗███████╗██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝
Blazing-fast command history search with instant documentation and terminal execution [Version: %s%s%s]

Copyright @ Naren Yellavula (Please give us a star ⭐ here: https://github.com/cybrota/recaller)

`

	asciiLogo = fmt.Sprintf(asciiLogo, Green, version, Reset)

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

			// Load configuration for fuzzy search
			config, err := LoadConfig()
			if err != nil {
				log.Printf("Failed to load configuration: %v. Using default settings.", err)
				config = &Config{EnableFuzzing: false}
			}

			res := getSuggestions(cmd.Flag("match").Value.String(), tree, config.EnableFuzzing)
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
		Use:     "recaller",
		Version: version,
		Long:    asciiLogo,
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
