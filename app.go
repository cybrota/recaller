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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	tb "github.com/nsf/termbox-go"
	"github.com/patrickmn/go-cache"
)

// ============================================================================
// CONSTANTS AND CONFIGURATION
// ============================================================================

var (
	Green, Info, Warning, Error, Reset string
)

const (
	debounceDelay     = 100 * time.Millisecond
	fsDebounceDelay   = 150 * time.Millisecond
	maxPathDisplayLen = 80
	fileSizeUnit      = 1024
)

// Filter modes for filesystem search
const (
	filterModeAll = iota
	filterModeDirs
	filterModeFiles
)

var (
	filterModes = []string{"All", "Dirs", "Files"}
	filterIcons = []string{"📁📄", "📁", "📄"}
)

// ============================================================================
// TERMINAL AND SYSTEM UTILITIES
// ============================================================================

// DisableMouseInput in termbox-go. This should be called after ui.Init()
func DisableMouseInput() {
	tb.SetInputMode(tb.InputEsc)
}

// sendToTerminal sends a command to the terminal (cross-platform)
func sendToTerminal(command string) error {
	switch runtime.GOOS {
	case "darwin":
		return sendToTerminalMacOS(command)
	case "linux":
		return sendToTerminalLinux(command)
	default:
		return fmt.Errorf("terminal automation not supported on %s", runtime.GOOS)
	}
}

// sendToTerminalMacOS sends command using AppleScript
func sendToTerminalMacOS(command string) error {
	escapedCommand := strings.ReplaceAll(command, `"`, `\"`)

	script := fmt.Sprintf(`tell application "Terminal"
		activate
		if (count of windows) = 0 then
			do script "%s"
		else
			set newTab to do script "%s" in front window
			set selected of newTab to true
		end if
	end tell`, escapedCommand, escapedCommand)

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()

	if err != nil {
		script = fmt.Sprintf(`tell application "iTerm2"
			tell current window
				set newSession to (create tab with default profile)
				tell current session to write text "%s"
			end tell
			activate
		end tell`, escapedCommand)
		cmd = exec.Command("osascript", "-e", script)
		err = cmd.Run()
	}

	return err
}

// sendToTerminalLinux sends command using Linux terminal emulators
func sendToTerminalLinux(command string) error {
	terminals := []struct {
		name string
		cmd  []string
	}{
		{"gnome-terminal", []string{"gnome-terminal", "--tab", "--", "bash", "-c", command + "; exec bash"}},
		{"konsole", []string{"konsole", "--new-tab", "-e", "bash", "-c", command + "; exec bash"}},
		{"xfce4-terminal", []string{"xfce4-terminal", "--tab", "-e", "bash -c '" + command + "; exec bash'"}},
		{"tilix", []string{"tilix", "-a", "session-add-down", "-e", "bash -c '" + command + "; exec bash'"}},
		{"terminator", []string{"terminator", "--new-tab", "-e", "bash -c '" + command + "; exec bash'"}},
		{"alacritty", []string{"alacritty", "-e", "bash", "-c", command + "; exec bash"}},
		{"kitty", []string{"kitty", "--tab", "bash", "-c", command + "; exec bash"}},
		{"xterm", []string{"xterm", "-e", "bash", "-c", command + "; exec bash"}},
	}

	for _, terminal := range terminals {
		if _, err := exec.LookPath(terminal.name); err == nil {
			cmd := exec.Command(terminal.cmd[0], terminal.cmd[1:]...)
			return cmd.Start()
		}
	}

	return fmt.Errorf("no supported terminal emulator found")
}

// openFileWithDefaultApp opens a file or directory with the system's default application
func openFileWithDefaultApp(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "linux":
		return exec.Command("xdg-open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// ============================================================================
// HELP AND CACHE UTILITIES
// ============================================================================

func GetOrfillCache(c *cache.Cache, cmd string) string {
	help, err := splitCommand(cmd)
	if err != nil {
		return ""
	}

	page := GetHelpPage(c, cmd)
	var helpTxt string

	if page == "" {
		helpTxt, err = getCommandHelp(help)
		if err != nil {
			helpTxt = fmt.Sprintf("Relax and take a deep breath.\n%s", err.Error())
		}
		CacheHelpPage(c, cmd, helpTxt)
	} else {
		helpTxt = page
	}

	return helpTxt
}

func repaintHelpWidget(c *cache.Cache, l *widgets.List, cmd string) {
	helpTxt := GetOrfillCache(c, cmd)
	lines := strings.Split(helpTxt, "\n")
	l.Rows = dedupeLines(lines)
}

// dedupeLines removes consecutive duplicate lines from a slice of strings.
func dedupeLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := []string{lines[0]}
	for _, ln := range lines[1:] {
		if ln != out[len(out)-1] {
			out = append(out, ln)
		}
	}
	return out
}

// ============================================================================
// UI LAYOUT AND WIDGET MANAGEMENT
// ============================================================================

func createKeyboardShortcutsWidget() *widgets.Paragraph {
	keyboardList := widgets.NewParagraph()
	keyboardList.Title = " Keyboard Shortcuts "
	keyboardList.Text = `[<enter>](fg:green) Copy command  [<ctrl+e>](fg:green) Send to terminal  [<ctrl+r>](fg:green) Reset input  [<tab>](fg:green) Switch panels  [<up/down>](fg:green) Navigate  [<ctrl+u>](fg:green) Insert command  [<ctrl+j/k>](fg:green) Jump first/last  [<F1>](fg:green) Show help  [<ctrl+z>](fg:green) Copy text  [<esc>](fg:green) Quit`
	keyboardList.TextStyle = StyleText()
	keyboardList.BorderStyle = StyleBorder(false)
	return keyboardList
}

func createInputWidget() *widgets.Paragraph {
	inputPara := widgets.NewParagraph()
	inputPara.Title = " Type Command "
	inputPara.Text = ""
	scheme := GetColorScheme()
	inputPara.TextStyle.Bg = scheme.Primary
	inputPara.TextStyle.Fg = scheme.OnPrimary
	inputPara.BorderStyle = StyleBorder(true)
	return inputPara
}

func createSuggestionListWidget() *widgets.List {
	suggestionList := widgets.NewList()
	suggestionList.Title = " Recalled From History ⚡ "
	suggestionList.Rows = []string{}
	suggestionList.SelectedRow = 0
	suggestionList.SelectedRowStyle = StyleSuccess()
	suggestionList.BorderStyle = StyleBorder(true)
	return suggestionList
}

func createHelpListWidget() *widgets.List {
	helpList := widgets.NewList()
	helpList.Title = " Help Doc "
	helpList.Rows = []string{"Select a command to display the help text"}
	helpList.SelectedRow = 0
	helpList.SelectedRowStyle = StyleWarning()
	helpList.WrapText = true
	helpList.BorderStyle = StyleBorder(false)
	return helpList
}

func showAIWidget(
	grid *ui.Grid,
	inputPara *widgets.Paragraph,
	suggestionList *widgets.List,
	helpList *widgets.List,
	aiResponsePara *widgets.Paragraph,
	keyboardList *widgets.Paragraph,
) {
	helpList.Rows = []string{}
	grid.Set(
		ui.NewRow(0.93,
			ui.NewCol(0.3,
				ui.NewRow(0.2, inputPara),
				ui.NewRow(0.82, suggestionList),
			),
			ui.NewCol(0.7, helpList),
		),
		ui.NewRow(0.07, keyboardList),
	)
}

func showHelpWidget(
	grid *ui.Grid,
	inputPara *widgets.Paragraph,
	suggestionList *widgets.List,
	helpList *widgets.List,
	aiResponsePara *widgets.Paragraph,
	keyboardList *widgets.Paragraph,
) {
	aiResponsePara.Text = ""
	grid.Set(
		ui.NewRow(0.93,
			ui.NewCol(0.3,
				ui.NewRow(0.2, inputPara),
				ui.NewRow(0.82, suggestionList),
			),
			ui.NewCol(0.7, helpList),
		),
		ui.NewRow(0.07, keyboardList),
	)
}

// toggleBorders toggles borders of given widgets b/w White & Cyan
func toggleBorders(w1 *widgets.List, w2 *widgets.List) {
	scheme := GetColorScheme()
	if w1.BorderStyle.Fg == scheme.BorderFocus {
		w1.BorderStyle = StyleBorder(false)
		w2.BorderStyle = StyleBorder(true)
	} else {
		w1.BorderStyle = StyleBorder(true)
		w2.BorderStyle = StyleBorder(false)
	}
}

// ============================================================================
// SEARCH AND SUGGESTION UTILITIES
// ============================================================================

// getSuggestions searches through file tree and returns list of matches
func getSuggestions(searchStr string, tree *AVLTree, enableFuzzing bool) []string {
	matches := SearchWithRanking(tree, searchStr, enableFuzzing)
	results := []string{}

	for _, node := range matches {
		results = append(results, fmt.Sprintf("%s", node.Command))
	}

	return results
}

// ============================================================================
// COMMAND HISTORY SEARCH UI
// ============================================================================

type historySearchState struct {
	inputBuffer     string
	selectedIndex   int
	lastSearchQuery string
	focusOnHelp     bool
}

func (state *historySearchState) updateSearchResults(tree *AVLTree, config *Config, suggestionList *widgets.List, helpList *widgets.List, hc *cache.Cache, grid *ui.Grid) {
	if state.inputBuffer == state.lastSearchQuery {
		return
	}
	state.lastSearchQuery = state.inputBuffer

	matches := SearchWithRanking(tree, state.inputBuffer, config.History.EnableFuzzing)
	suggestionList.Rows = suggestionList.Rows[:0]

	for _, node := range matches {
		suggestionList.Rows = append(suggestionList.Rows, node.Command)
	}

	if state.selectedIndex >= len(suggestionList.Rows) {
		state.selectedIndex = 0
	}
	if state.selectedIndex < 0 {
		state.selectedIndex = 0
	}
	suggestionList.SelectedRow = state.selectedIndex

	if len(suggestionList.Rows) > 0 {
		selectedCmd := suggestionList.Rows[state.selectedIndex]
		helpList.SelectedRow = 0
		repaintHelpWidget(hc, helpList, selectedCmd)
	}

	ui.Render(grid)
}

func (state *historySearchState) handleNavigation(direction string, suggestionList *widgets.List, helpList *widgets.List, hc *cache.Cache, grid *ui.Grid, inputPara *widgets.Paragraph, aiResponsePara *widgets.Paragraph, keyboardList *widgets.Paragraph) {
	if state.focusOnHelp {
		switch direction {
		case "up":
			if helpList.SelectedRow > 0 {
				helpList.SelectedRow--
			}
		case "down":
			if helpList.SelectedRow < len(helpList.Rows)-1 {
				helpList.SelectedRow++
			}
		case "first":
			if len(helpList.Rows) > 0 {
				helpList.SelectedRow = 0
			}
		case "last":
			if len(helpList.Rows) > 0 {
				helpList.SelectedRow = len(helpList.Rows) - 1
			}
		}
	} else {
		switch direction {
		case "up":
			if state.selectedIndex > 0 {
				state.selectedIndex--
				suggestionList.SelectedRow = state.selectedIndex
				selectedCmd := suggestionList.Rows[state.selectedIndex]
				helpList.SelectedRow = 0
				repaintHelpWidget(hc, helpList, selectedCmd)
				showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
			}
		case "down":
			if state.selectedIndex < len(suggestionList.Rows)-1 {
				state.selectedIndex++
				suggestionList.SelectedRow = state.selectedIndex
				selectedCmd := suggestionList.Rows[state.selectedIndex]
				helpList.SelectedRow = 0
				repaintHelpWidget(hc, helpList, selectedCmd)
				showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
			}
		case "first":
			state.selectedIndex = 0
			suggestionList.SelectedRow = state.selectedIndex
		case "last":
			if len(suggestionList.Rows) > 0 {
				state.selectedIndex = len(suggestionList.Rows) - 1
				suggestionList.SelectedRow = state.selectedIndex
			}
		}
	}
}

func run(tree *AVLTree, hc *cache.Cache) {
	// Initialize color system
	InitializeColors()
	Green, Info, Warning, Error, Reset = GetANSIColors()

	config, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to load configuration: %v. Using default settings.", err)
		config = &Config{History: HistoryConfig{EnableFuzzing: true}}
	}

	done := make(chan bool)
	searchDebouncer := time.NewTimer(0)
	searchDebouncer.Stop()

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	DisableMouseInput()
	defer ui.Close()

	// Create UI widgets
	keyboardList := createKeyboardShortcutsWidget()
	inputPara := createInputWidget()
	suggestionList := createSuggestionListWidget()
	helpList := createHelpListWidget()
	aiResponsePara := widgets.NewParagraph()
	aiResponsePara.Title = " AI Doc "
	aiResponsePara.Text = ""
	aiResponsePara.TextStyle = StyleText()
	aiResponsePara.BorderStyle = StyleBorder(false)

	// Setup grid layout
	termWidth, termHeight := ui.TerminalDimensions()
	grid := ui.NewGrid()
	grid.SetRect(0, 0, termWidth, termHeight)
	showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
	ui.Render(grid)

	// Initialize search state
	state := &historySearchState{
		inputBuffer:     "",
		selectedIndex:   0,
		lastSearchQuery: "",
		focusOnHelp:     false,
	}

	uiEvents := ui.PollEvents()

	// Start debouncer goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			case <-searchDebouncer.C:
				state.updateSearchResults(tree, config, suggestionList, helpList, hc, grid)
			}
		}
	}()

	// Perform initial search
	state.updateSearchResults(tree, config, suggestionList, helpList, hc, grid)

	for {
		e := <-uiEvents
		switch e.ID {
		case "<C-c>", "<Escape>":
			done <- true
			return
		case "<C-z>":
			selectedText := helpList.Rows[helpList.SelectedRow]
			if err := clipboard.WriteAll(selectedText); err != nil {
				log.Printf("Failed to copy text: %v", err)
			} else {
				log.Println("Text successfully copied to clipboard!")
			}
		case "<Tab>":
			state.focusOnHelp = !state.focusOnHelp
			toggleBorders(suggestionList, helpList)
		case "<Backspace>":
			if len(state.inputBuffer) > 0 {
				state.inputBuffer = state.inputBuffer[:len(state.inputBuffer)-1]
			}
			searchDebouncer.Reset(debounceDelay)
		case "<Space>":
			state.inputBuffer += " "
			searchDebouncer.Reset(debounceDelay)
		case "<Enter>":
			var commandToCopy string
			if len(suggestionList.Rows) > 0 {
				commandToCopy = suggestionList.Rows[state.selectedIndex]
			} else {
				commandToCopy = state.inputBuffer
			}
			if commandToCopy != "" {
				if err := clipboard.WriteAll(commandToCopy); err != nil {
					log.Printf("Failed to copy command to clipboard: %v", err)
				}
			}
			ui.Close()
			if commandToCopy != "" {
				fmt.Fprintf(os.Stderr, "📋 Copied %s%s%s to clipboard.\n", Green, commandToCopy, Reset)
			}
			return
		case "<C-e>":
			var commandToSend string
			if len(suggestionList.Rows) > 0 {
				commandToSend = suggestionList.Rows[state.selectedIndex]
			} else {
				commandToSend = state.inputBuffer
			}

			if commandToSend != "" {
				if err := sendToTerminal(commandToSend); err != nil {
					log.Printf("Failed to send command to terminal: %v", err)
				} else {
					fmt.Printf("⚡ Sent `%s` to terminal\n", commandToSend)
				}
			}
			ui.Close()
			return
		case "<Up>":
			state.handleNavigation("up", suggestionList, helpList, hc, grid, inputPara, aiResponsePara, keyboardList)
		case "<Down>":
			state.handleNavigation("down", suggestionList, helpList, hc, grid, inputPara, aiResponsePara, keyboardList)
		case "<F1>":
			var selectedCmd string
			if len(suggestionList.Rows) > 0 {
				selectedCmd = suggestionList.Rows[state.selectedIndex]
			} else {
				selectedCmd = inputPara.Text
			}
			repaintHelpWidget(hc, helpList, selectedCmd)
			showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
		case "<C-u>":
			if !state.focusOnHelp {
				state.inputBuffer = suggestionList.Rows[state.selectedIndex]
			}
		case "<C-r>":
			if !state.focusOnHelp {
				state.inputBuffer = ""
			}
		case "<C-j>":
			state.handleNavigation("last", suggestionList, helpList, hc, grid, inputPara, aiResponsePara, keyboardList)
		case "<C-k>":
			state.handleNavigation("first", suggestionList, helpList, hc, grid, inputPara, aiResponsePara, keyboardList)
		case "<Resize>":
			if payload, ok := e.Payload.(ui.Resize); ok {
				grid.SetRect(0, 0, payload.Width, payload.Height)
			} else {
				termWidth, termHeight := ui.TerminalDimensions()
				grid.SetRect(0, 0, termWidth, termHeight)
			}
			showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
			ui.Clear()
			ui.Render(grid)
		default:
			if !state.focusOnHelp {
				if e.Type == ui.KeyboardEvent && len(e.ID) == 1 {
					state.inputBuffer += e.ID
					searchDebouncer.Reset(debounceDelay)
				}
			}
		}

		inputPara.Text = state.inputBuffer
		ui.Render(grid)
	}
}

// ============================================================================
// FILESYSTEM SEARCH UTILITIES
// ============================================================================

// formatFileForDisplay formats a file path for display in the UI
func formatFileForDisplay(file RankedFile) string {
	var icon string
	if file.Metadata.IsDirectory {
		icon = "📁"
	} else {
		icon = "📄"
	}

	currentDir, _ := os.Getwd()
	displayPath := file.Path
	if relPath, err := filepath.Rel(currentDir, file.Path); err == nil && !strings.HasPrefix(relPath, "..") {
		displayPath = relPath
	}

	if len(displayPath) > maxPathDisplayLen {
		displayPath = "..." + displayPath[len(displayPath)-maxPathDisplayLen+3:]
	}

	return fmt.Sprintf("%s %s", icon, displayPath)
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	if size < fileSizeUnit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(fileSizeUnit), 0
	for n := size / fileSizeUnit; n >= fileSizeUnit; n /= fileSizeUnit {
		div *= fileSizeUnit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// ============================================================================
// FILESYSTEM SEARCH UI
// ============================================================================

type filesystemSearchState struct {
	inputBuffer     string
	selectedIndex   int
	lastSearchQuery string
	focusOnMetadata bool
	filterMode      int
	currentFiles    []RankedFile
}

func (state *filesystemSearchState) updateFileListTitle(fileList *widgets.List) {
	fileList.Title = fmt.Sprintf(" %s %s ", filterIcons[state.filterMode], filterModes[state.filterMode])
}

func (state *filesystemSearchState) updateMetadataDisplay(metadataList *widgets.List) {
	if len(state.currentFiles) == 0 || state.selectedIndex >= len(state.currentFiles) {
		metadataList.Rows = []string{"Select a file to view details"}
		return
	}

	file := state.currentFiles[state.selectedIndex]
	metadata := []string{
		fmt.Sprintf("📍 Path: %s", file.Path),
	}

	if file.Metadata.IsDirectory {
		metadata = append(metadata, "📁 Type: Directory")
	} else {
		ext := strings.ToUpper(filepath.Ext(file.Path))
		if ext == "" {
			ext = "FILE"
		} else {
			ext = ext[1:]
		}
		metadata = append(metadata, fmt.Sprintf("📄 Type: %s", ext))
	}

	if file.Metadata.Timestamp != nil {
		metadata = append(metadata, fmt.Sprintf("🕒 Last Accessed: %s", file.Metadata.Timestamp.Format("2006-01-02 15:04:05")))
	}
	metadata = append(metadata, fmt.Sprintf("📊 Access Count: %d", file.Metadata.AccessCount))
	metadata = append(metadata, fmt.Sprintf("⭐ Score: %.2f", file.Score))

	if !file.Metadata.IsDirectory && file.Metadata.Size > 0 {
		size := formatFileSize(file.Metadata.Size)
		metadata = append(metadata, fmt.Sprintf("💾 Size: %s", size))
	}

	if !file.Metadata.LastModified.IsZero() {
		metadata = append(metadata, fmt.Sprintf("✏️  Modified: %s", file.Metadata.LastModified.Format("2006-01-02 15:04:05")))
	}

	if file.Metadata.IsHidden {
		metadata = append(metadata, "🔒 Hidden file")
	}
	if file.Metadata.IsSymlink {
		metadata = append(metadata, "🔗 Symbolic link")
	}

	metadataList.Rows = metadata
	metadataList.SelectedRow = 0
}

func (state *filesystemSearchState) updateFileResults(fsIndexer *FilesystemIndexer, config *Config, fileList *widgets.List, metadataList *widgets.List, grid *ui.Grid) {
	if state.inputBuffer == state.lastSearchQuery {
		return
	}
	state.lastSearchQuery = state.inputBuffer

	if state.inputBuffer == "" {
		fileList.Rows = []string{"Type to search files and directories..."}
		state.currentFiles = []RankedFile{}
	} else {
		allFiles := fsIndexer.SearchFiles(state.inputBuffer, config.History.EnableFuzzing)
		filteredFiles := []RankedFile{}

		for _, file := range allFiles {
			switch state.filterMode {
			case filterModeAll:
				filteredFiles = append(filteredFiles, file)
			case filterModeDirs:
				if file.Metadata.IsDirectory {
					filteredFiles = append(filteredFiles, file)
				}
			case filterModeFiles:
				if !file.Metadata.IsDirectory {
					filteredFiles = append(filteredFiles, file)
				}
			}
		}

		state.currentFiles = filteredFiles
		fileList.Rows = fileList.Rows[:0]

		for _, file := range filteredFiles {
			fileList.Rows = append(fileList.Rows, formatFileForDisplay(file))
		}

		if len(fileList.Rows) == 0 {
			filterText := filterModes[state.filterMode]
			if state.filterMode == filterModeAll {
				fileList.Rows = []string{"No files found matching: " + state.inputBuffer}
			} else {
				fileList.Rows = []string{fmt.Sprintf("No %s found matching: %s", strings.ToLower(filterText), state.inputBuffer)}
			}
		}
	}

	if state.selectedIndex >= len(state.currentFiles) {
		state.selectedIndex = 0
	}
	if state.selectedIndex < 0 {
		state.selectedIndex = 0
	}
	fileList.SelectedRow = state.selectedIndex

	state.updateFileListTitle(fileList)
	state.updateMetadataDisplay(metadataList)
	ui.Render(grid)
}

func createFilesystemKeyboardWidget() *widgets.Paragraph {
	keyboardList := widgets.NewParagraph()
	keyboardList.Title = " Filesystem Search Shortcuts "
	keyboardList.Text = `[<enter>](fg:green) Open file  [<ctrl+x>](fg:green) Copy path  [<ctrl+r>](fg:green) Reset input  [<up/down>](fg:green) Navigate  [<ctrl+j/k>](fg:green) Jump first/last  [<ctrl+t>](fg:green) Toggle filter  [<tab>](fg:green) Switch panels  [<esc>](fg:green) Quit`
	keyboardList.TextStyle = StyleText()
	keyboardList.BorderStyle = StyleBorder(false)
	return keyboardList
}

func createFilesystemInputWidget() *widgets.Paragraph {
	inputPara := widgets.NewParagraph()
	inputPara.Title = " Search Files & Directories "
	inputPara.Text = ""
	scheme := GetColorScheme()
	inputPara.TextStyle.Bg = scheme.Primary
	inputPara.TextStyle.Fg = scheme.OnPrimary
	inputPara.BorderStyle = StyleBorder(true)
	return inputPara
}

func createFileListWidget() *widgets.List {
	fileList := widgets.NewList()
	fileList.Title = " 📁 Files & Directories "
	fileList.Rows = []string{"Type to search files and directories..."}
	fileList.SelectedRow = 0
	fileList.SelectedRowStyle = StyleSuccess()
	fileList.BorderStyle = StyleBorder(true)
	return fileList
}

func createMetadataListWidget() *widgets.List {
	metadataList := widgets.NewList()
	metadataList.Title = " 📋 File Info "
	metadataList.Rows = []string{"Select a file to view details"}
	metadataList.SelectedRow = 0
	metadataList.SelectedRowStyle = StyleInfo()
	metadataList.WrapText = true
	metadataList.BorderStyle = StyleBorder(false)
	return metadataList
}

// runFilesystemSearch launches the filesystem search UI
func runFilesystemSearch(fsIndexer *FilesystemIndexer, config *Config) {
	// Initialize color system
	InitializeColors()
	Green, Info, Warning, Error, Reset = GetANSIColors()

	searchDebouncer := time.NewTimer(0)
	searchDebouncer.Stop()

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	DisableMouseInput()
	defer ui.Close()

	// Create UI widgets
	keyboardList := createFilesystemKeyboardWidget()
	inputPara := createFilesystemInputWidget()
	fileList := createFileListWidget()
	metadataList := createMetadataListWidget()

	// Setup layout
	termWidth, termHeight := ui.TerminalDimensions()
	grid := ui.NewGrid()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(0.93,
			ui.NewCol(0.4,
				ui.NewRow(0.2, inputPara),
				ui.NewRow(0.8, fileList),
			),
			ui.NewCol(0.6, metadataList),
		),
		ui.NewRow(0.07, keyboardList),
	)

	ui.Render(grid)

	// Initialize search state
	state := &filesystemSearchState{
		inputBuffer:     "",
		selectedIndex:   0,
		lastSearchQuery: "",
		focusOnMetadata: false,
		filterMode:      filterModeAll,
		currentFiles:    []RankedFile{},
	}

	uiEvents := ui.PollEvents()
	done := make(chan bool)

	// Start debouncer goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			case <-searchDebouncer.C:
				state.updateFileResults(fsIndexer, config, fileList, metadataList, grid)
			}
		}
	}()

	// Set initial title and perform initial search
	state.updateFileListTitle(fileList)
	state.updateFileResults(fsIndexer, config, fileList, metadataList, grid)

	for {
		e := <-uiEvents
		switch e.ID {
		case "<C-c>", "<Escape>":
			done <- true
			return
		case "<Tab>":
			state.focusOnMetadata = !state.focusOnMetadata
			if state.focusOnMetadata {
				fileList.BorderStyle = StyleBorder(false)
				metadataList.BorderStyle = StyleBorder(true)
			} else {
				fileList.BorderStyle = StyleBorder(true)
				metadataList.BorderStyle = StyleBorder(false)
			}
		case "<Backspace>":
			if !state.focusOnMetadata && len(state.inputBuffer) > 0 {
				state.inputBuffer = state.inputBuffer[:len(state.inputBuffer)-1]
				searchDebouncer.Reset(fsDebounceDelay)
			}
		case "<Space>":
			if state.focusOnMetadata {
				if metadataList.SelectedRow < len(metadataList.Rows)-1 {
					metadataList.SelectedRow++
				}
			} else {
				state.inputBuffer += " "
				searchDebouncer.Reset(fsDebounceDelay)
			}
		case "<Enter>":
			if len(state.currentFiles) > state.selectedIndex && state.selectedIndex >= 0 {
				filePath := state.currentFiles[state.selectedIndex].Path
				fsIndexer.AddPath(filePath, time.Now())

				if err := openFileWithDefaultApp(filePath); err != nil {
					log.Printf("Failed to open file: %v", err)
				} else {
					fmt.Printf("🚀 Opened: %s\n", filePath)
				}

				go func() {
					if err := fsIndexer.PersistIndex(!config.Quiet); err != nil {
						log.Printf("Failed to persist index: %v", err)
					}
				}()
			}
			ui.Close()
			return
		case "<C-x>":
			if len(state.currentFiles) > state.selectedIndex && state.selectedIndex >= 0 {
				filePath := state.currentFiles[state.selectedIndex].Path
				if err := clipboard.WriteAll(filePath); err != nil {
					log.Printf("Failed to copy path: %v", err)
				}
				ui.Close()
				fmt.Printf("📋 Copied path: %s\n", filePath)
				return
			}
		case "<Up>":
			if state.focusOnMetadata {
				if metadataList.SelectedRow > 0 {
					metadataList.SelectedRow--
				}
			} else {
				if state.selectedIndex > 0 && len(state.currentFiles) > 0 {
					state.selectedIndex--
					fileList.SelectedRow = state.selectedIndex
					state.updateMetadataDisplay(metadataList)
				}
			}
		case "<Down>":
			if state.focusOnMetadata {
				if metadataList.SelectedRow < len(metadataList.Rows)-1 {
					metadataList.SelectedRow++
				}
			} else {
				if state.selectedIndex < len(state.currentFiles)-1 && len(state.currentFiles) > 0 {
					state.selectedIndex++
					fileList.SelectedRow = state.selectedIndex
					state.updateMetadataDisplay(metadataList)
				}
			}
		case "<C-r>":
			if !state.focusOnMetadata {
				state.inputBuffer = ""
				searchDebouncer.Reset(fsDebounceDelay)
			}
		case "<C-j>":
			if !state.focusOnMetadata {
				if len(state.currentFiles) > 0 {
					state.selectedIndex = len(state.currentFiles) - 1
					fileList.SelectedRow = state.selectedIndex
					state.updateMetadataDisplay(metadataList)
				}
			} else {
				if len(metadataList.Rows) > 0 {
					metadataList.SelectedRow = len(metadataList.Rows) - 1
				}
			}
		case "<C-k>":
			if !state.focusOnMetadata {
				state.selectedIndex = 0
				fileList.SelectedRow = state.selectedIndex
				state.updateMetadataDisplay(metadataList)
			} else {
				metadataList.SelectedRow = 0
			}
		case "<C-t>":
			state.filterMode = (state.filterMode + 1) % 3
			state.lastSearchQuery = ""
			state.updateFileResults(fsIndexer, config, fileList, metadataList, grid)
		case "<Resize>":
			if payload, ok := e.Payload.(ui.Resize); ok {
				grid.SetRect(0, 0, payload.Width, payload.Height)
			} else {
				termWidth, termHeight := ui.TerminalDimensions()
				grid.SetRect(0, 0, termWidth, termHeight)
			}
			ui.Clear()
			ui.Render(grid)
		default:
			if !state.focusOnMetadata && e.Type == ui.KeyboardEvent && len(e.ID) == 1 {
				state.inputBuffer += e.ID
				searchDebouncer.Reset(fsDebounceDelay)
			}
		}

		inputPara.Text = state.inputBuffer
		ui.Render(grid)
	}
}
