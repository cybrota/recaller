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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	// Initialize color system early
	InitializeColors()
	Green, Info, Warning, Error, Reset = GetANSIColors()

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

	asciiLogo = fmt.Sprintf(asciiLogo, Info, version, Reset)

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
				config = &Config{History: HistoryConfig{EnableFuzzing: true}}
			}

			res := getSuggestions(cmd.Flag("match").Value.String(), tree, config.History.EnableFuzzing)
			fmt.Println(strings.Join(res, "\n"))
		},
	}

	cmdHistory.Flags().String("match", "", "match string prefix to look in history")

	var cmdFs = &cobra.Command{
		Use:   "fs",
		Short: "Filesystem search commands",
		Long:  fmt.Sprintf("%s\n%s", asciiLogo, `Launch filesystem search UI using existing index, or use subcommands to manage the index. Use 'recaller fs index [path]' to index directories first.`),
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			config, err := LoadConfig()
			if err != nil {
				log.Printf("Failed to load configuration: %v. Using default settings.", err)
				config = &defaultConfig
			}

			if !config.Filesystem.Enabled {
				fmt.Printf("❌ Filesystem search is disabled. Enable it in configuration:\n")
				fmt.Printf("Edit ~/.recaller.yaml and set:\n")
				fmt.Printf("filesystem:\n  enabled: true\n\n")
				fmt.Printf("Or run: recaller settings list\n")
				return
			}

			// Create filesystem indexer
			fsIndexer := NewFilesystemIndexer(config.Filesystem)

			// Load existing index
			if err := fsIndexer.LoadOrCreateIndex(!config.Quiet); err != nil {
				fmt.Printf("❌ Failed to load filesystem index: %v\n", err)
				fmt.Printf("💡 Run 'recaller fs index [path]' to create an index first.\n")
				return
			}

			// Check if index has any data
			if !fsIndexer.HasIndexedFiles() {
				fmt.Printf("📂 No files found in index.\n")
				fmt.Printf("💡 Run 'recaller fs index [path]' to index directories first.\n")
				return
			}

			// Auto re-index existing paths to discover new files
			if len(fsIndexer.GetRootPaths()) > 0 {
				if err := fsIndexer.RefreshIndex(!config.Quiet, false); err != nil {
					log.Printf("Warning: Re-indexing completed with errors: %v", err)
				}
			}

			// Show index statistics
			fmt.Printf("📊 %s\n", fsIndexer.GetIndexStats())

			// Launch filesystem search UI
			fmt.Printf("🚀 Launching filesystem search UI...\n")
			runFilesystemSearch(fsIndexer, config)
		},
	}

	var cmdFsIndex = &cobra.Command{
		Use:   "index [path1] [path2] ...",
		Short: "Index directories for filesystem search",
		Long:  `Index one or more directories for filesystem search without launching the UI. Optional paths to index (defaults to current directory if none provided).`,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			config, err := LoadConfig()
			if err != nil {
				log.Printf("Failed to load configuration: %v. Using default settings.", err)
				config = &defaultConfig
			}

			if !config.Filesystem.Enabled {
				fmt.Printf("❌ Filesystem search is disabled. Enable it in configuration:\n")
				fmt.Printf("Edit ~/.recaller.yaml and set:\n")
				fmt.Printf("filesystem:\n  enabled: true\n\n")
				fmt.Printf("Or run: recaller settings list\n")
				return
			}

			// Determine paths to index
			pathsToIndex := []string{"."}
			if len(args) > 0 {
				pathsToIndex = args
			}

			// Process each path: expand tilde, convert to absolute path, and verify existence
			var validPaths []string
			for _, pathToIndex := range pathsToIndex {
				// Expand tilde in path
				if strings.HasPrefix(pathToIndex, "~/") {
					homeDir, err := os.UserHomeDir()
					if err == nil {
						pathToIndex = filepath.Join(homeDir, pathToIndex[2:])
					}
				}

				// Convert to absolute path
				absPath, err := filepath.Abs(pathToIndex)
				if err != nil {
					fmt.Printf("❌ Invalid path: %s\n", pathToIndex)
					continue
				}

				// Verify path exists
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					fmt.Printf("❌ Path does not exist: %s\n", absPath)
					continue
				}

				validPaths = append(validPaths, absPath)
			}

			if len(validPaths) == 0 {
				fmt.Printf("❌ No valid paths to index\n")
				return
			}

			// Create filesystem indexer
			fsIndexer := NewFilesystemIndexer(config.Filesystem)

			// Load existing index if available
			if err := fsIndexer.LoadOrCreateIndex(!config.Quiet); err != nil {
				log.Printf("Failed to load filesystem index: %v", err)
			}

			// Index the specified directories with progress
			if len(validPaths) == 1 {
				fmt.Printf("🔍 Starting filesystem indexing for: %s\n", validPaths[0])
				if err := fsIndexer.IndexDirectoryWithProgress(validPaths[0], true); err != nil {
					if err.Error() == "max indexed files limit reached" {
						fmt.Printf("⚠️  Reached maximum file limit (%d files)\n", config.Filesystem.MaxIndexedFiles)
					} else {
						log.Printf("Warning: Indexing completed with errors: %v", err)
					}
				}
			} else {
				fmt.Printf("🔍 Starting filesystem indexing for %d directories:\n", len(validPaths))
				for i, path := range validPaths {
					fmt.Printf("  %d. %s\n", i+1, path)
				}
				fmt.Println()
				if err := fsIndexer.IndexDirectoriesWithProgress(validPaths, true); err != nil {
					if err.Error() == "max indexed files limit reached" {
						fmt.Printf("⚠️  Reached maximum file limit (%d files)\n", config.Filesystem.MaxIndexedFiles)
					} else {
						log.Printf("Warning: Indexing completed with errors: %v", err)
					}
				}
			}

			// Persist the index
			fmt.Printf("\n💾 Saving index to disk...")
			if err := fsIndexer.PersistIndex(!config.Quiet); err != nil {
				log.Printf("Warning: Failed to persist index: %v", err)
			} else {
				fmt.Printf(" ✅\n")
			}

			// Show index statistics
			fmt.Printf("\n%s\n", fsIndexer.GetIndexStats())
			fmt.Printf("\n💡 Run 'recaller fs' to launch the search UI.\n")
		},
	}

	var cmdFsClean = &cobra.Command{
		Use:   "clean [path]",
		Short: "Clean filesystem index",
		Long:  `Clean the filesystem index by removing stale entries, old entries, or entries matching a specific path.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			config, err := LoadConfig()
			if err != nil {
				log.Printf("Failed to load configuration: %v. Using default settings.", err)
				config = &defaultConfig
			}

			if !config.Filesystem.Enabled {
				fmt.Printf("❌ Filesystem search is disabled. Enable it first.\n")
				return
			}

			// Create filesystem indexer
			fsIndexer := NewFilesystemIndexer(config.Filesystem)

			// Load existing index
			if err := fsIndexer.LoadOrCreateIndex(!config.Quiet); err != nil {
				fmt.Printf("❌ Failed to load filesystem index: %v\n", err)
				return
			}

			// Get current index stats
			initialSize, _ := fsIndexer.GetIndexFileSize()
			fmt.Printf("📊 Current index: %s\n", fsIndexer.GetIndexStats())
			if initialSize > 0 {
				fmt.Printf("💾 Index file size: %.2f KB\n\n", float64(initialSize)/1024)
			}

			// Parse flags
			removeStale, _ := cmd.Flags().GetBool("stale")
			olderThanDays, _ := cmd.Flags().GetInt("older-than")
			clearAll, _ := cmd.Flags().GetBool("clear")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			if clearAll {
				if dryRun {
					fmt.Printf("🔍 [DRY RUN] Would clear entire index (%d entries)\n", len(fsIndexer.pathRecords))
					return
				}

				fmt.Printf("⚠️  This will clear the entire filesystem index. Continue? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Printf("❌ Operation cancelled.\n")
					return
				}

				if err := fsIndexer.ClearIndex(); err != nil {
					fmt.Printf("❌ Failed to clear index: %v\n", err)
					return
				}

				if err := fsIndexer.PersistIndex(!config.Quiet); err != nil {
					fmt.Printf("❌ Failed to persist cleared index: %v\n", err)
					return
				}

				fmt.Printf("✅ Index cleared successfully!\n")
				return
			}

			// Determine cleanup options
			var pathPrefix string
			if len(args) > 0 {
				pathPrefix = args[0]
				// Expand tilde and convert to absolute path
				if strings.HasPrefix(pathPrefix, "~/") {
					if homeDir, err := os.UserHomeDir(); err == nil {
						pathPrefix = filepath.Join(homeDir, pathPrefix[2:])
					}
				}
				if absPath, err := filepath.Abs(pathPrefix); err == nil {
					pathPrefix = absPath
				}
			}

			options := CleanupOptions{
				Path:          pathPrefix,
				RemoveStale:   removeStale,
				OlderThanDays: olderThanDays,
				ShowProgress:  !dryRun, // Show progress only for actual cleanup
			}

			// Perform dry run first if requested
			if dryRun {
				fmt.Printf("🔍 [DRY RUN] Analyzing what would be cleaned...\n")
				options.ShowProgress = false
			} else {
				fmt.Printf("🧹 Starting cleanup...\n")
			}

			// Run cleanup
			stats, err := fsIndexer.CleanupIndex(options)
			if err != nil {
				fmt.Printf("❌ Cleanup failed: %v\n", err)
				return
			}

			// Display results
			fmt.Printf("\n📈 Cleanup Results:\n")
			fmt.Printf("   Total entries: %d\n", stats.TotalEntries)
			if stats.StaleFiles > 0 {
				fmt.Printf("   Stale files removed: %d\n", stats.StaleFiles)
			}
			if stats.OldFiles > 0 {
				fmt.Printf("   Old entries removed: %d\n", stats.OldFiles)
			}
			if pathPrefix != "" {
				fmt.Printf("   Path-filtered entries removed: %d\n", stats.RemovedEntries)
			}
			fmt.Printf("   Total removed: %d entries\n", stats.RemovedEntries)
			fmt.Printf("   Memory freed: %.2f KB\n", stats.FreedKB)

			if !dryRun && stats.RemovedEntries > 0 {
				// Persist changes
				fmt.Printf("\n💾 Saving cleaned index...")
				if err := fsIndexer.PersistIndex(!config.Quiet); err != nil {
					fmt.Printf(" ❌ Failed: %v\n", err)
				} else {
					fmt.Printf(" ✅\n")

					// Show new stats
					newSize, _ := fsIndexer.GetIndexFileSize()
					fmt.Printf("\n📊 Updated index: %s\n", fsIndexer.GetIndexStats())
					if initialSize > 0 && newSize > 0 {
						freed := float64(initialSize-newSize) / 1024
						fmt.Printf("💾 Disk space freed: %.2f KB\n", freed)
					}
				}
			} else if dryRun {
				fmt.Printf("\n💡 Run without --dry-run to actually perform the cleanup.\n")
			} else {
				fmt.Printf("\n✅ No cleanup needed - index is already clean!\n")
			}
		},
	}

	// Add flags for clean command
	cmdFsClean.Flags().Bool("stale", false, "Remove entries for files that no longer exist")
	cmdFsClean.Flags().Int("older-than", 0, "Remove entries older than N days")
	cmdFsClean.Flags().Bool("clear", false, "Clear the entire index (requires confirmation)")
	cmdFsClean.Flags().Bool("dry-run", false, "Show what would be cleaned without making changes")

	var cmdFsRefresh = &cobra.Command{
		Use:   "refresh",
		Short: "Re-index all tracked paths to discover new files",
		Long:  `Re-index all previously indexed directories to discover new files and directories without launching the search UI. This is useful for manually updating your index.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			config, err := LoadConfig()
			if err != nil {
				log.Printf("Failed to load configuration: %v. Using default settings.", err)
				config = &defaultConfig
			}

			if !config.Filesystem.Enabled {
				fmt.Printf("❌ Filesystem search is disabled. Enable it in configuration:\n")
				fmt.Printf("Edit ~/.recaller.yaml and set:\n")
				fmt.Printf("filesystem:\n  enabled: true\n\n")
				fmt.Printf("Or run: recaller settings list\n")
				return
			}

			// Create filesystem indexer
			fsIndexer := NewFilesystemIndexer(config.Filesystem)

			// Load existing index
			if err := fsIndexer.LoadOrCreateIndex(!config.Quiet); err != nil {
				fmt.Printf("❌ Failed to load filesystem index: %v\n", err)
				fmt.Printf("💡 Run 'recaller fs index [path]' to create an index first.\n")
				return
			}

			// Refresh the index using the shared function
			if err := fsIndexer.RefreshIndex(!config.Quiet, true); err != nil {
				if err.Error() == "no tracked paths found in index" {
					fmt.Printf("📂 No tracked paths found in index.\n")
					fmt.Printf("💡 Run 'recaller fs index [path]' to index directories first.\n")
				} else if err.Error() == "max indexed files limit reached" {
					fmt.Printf("⚠️  Reached maximum file limit (%d files)\n", config.Filesystem.MaxIndexedFiles)
				} else {
					fmt.Printf("❌ Refresh failed: %v\n", err)
				}
				return
			}

			fmt.Printf("✔️ Refresh completed successfully!\n")
		},
	}

	var cmdSettingsList = &cobra.Command{
		Use:   "list",
		Short: "List current configuration settings",
		Long:  "Display all current configuration settings with their values",
		Run: func(cmd *cobra.Command, args []string) {
			displaySettings()
		},
	}

	var cmdSettings = &cobra.Command{
		Use:   "settings",
		Short: "Manage Recaller configuration settings",
		Long:  "Commands for viewing and managing Recaller configuration",
	}

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

	cmdSettings.AddCommand(cmdSettingsList)
	cmdFs.AddCommand(cmdFsIndex, cmdFsClean, cmdFsRefresh)
	rootCmd.AddCommand(cmdRun, cmdUsage, cmdVersion, cmdHistory, cmdFs, cmdSettings)
	rootCmd.Execute()
}
