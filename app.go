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
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	tb "github.com/nsf/termbox-go"
	"github.com/patrickmn/go-cache"
)

const (
	Green = "\033[32m"
	Reset = "\033[0m"
)

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
	// Escape double quotes for AppleScript
	escapedCommand := strings.ReplaceAll(command, `"`, `\"`)

	// Try Terminal.app first (most common)
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

	// If Terminal.app fails, try iTerm2
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
	// Try different terminal emulators in order of preference
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

	// First, try to detect which terminal is available
	for _, terminal := range terminals {
		if _, err := exec.LookPath(terminal.name); err == nil {
			cmd := exec.Command(terminal.cmd[0], terminal.cmd[1:]...)
			return cmd.Start() // Use Start() instead of Run() to avoid blocking
		}
	}

	return fmt.Errorf("no supported terminal emulator found")
}

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
				ui.NewRow(0.82, suggestionList), // 0.82 for fill empty padding forced by keyboard widget
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
				ui.NewRow(0.82, suggestionList), // 0.82 for fill empty padding forced by keyboard widget
			),
			ui.NewCol(0.7, helpList),
		),
		ui.NewRow(0.07, keyboardList),
	)
}

// toggleBorders toggles borders of given widgets b/w White & Cyan
func toggleBorders(w1 *widgets.List, w2 *widgets.List) {
	if w1.BorderStyle.Fg == ui.ColorCyan {
		w1.BorderStyle = ui.NewStyle(ui.ColorWhite)
		w2.BorderStyle = ui.NewStyle(ui.ColorCyan)
	} else {
		w1.BorderStyle = ui.NewStyle(ui.ColorCyan)
		w2.BorderStyle = ui.NewStyle(ui.ColorWhite)
	}
}

func run(tree *AVLTree, hc *cache.Cache) {
	// co_key := os.Getenv("COHERE_API_KEY")
	// client := cohereclient.NewClient(cohereclient.WithToken(co_key))

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to load configuration: %v. Using default settings.", err)
		config = &Config{EnableFuzzing: false}
	}

	// Done channel for ticker
	done := make(chan bool)

	// Debouncing for search operations
	searchDebouncer := time.NewTimer(0)
	searchDebouncer.Stop()
	const debounceDelay = 100 * time.Millisecond

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	DisableMouseInput()
	defer ui.Close()

	rows, _ := ReadFilesAndDirs("green")
	fileTreeRowTbl := widgets.NewTable()
	fileTreeRowTbl.Title = "Files & Directories"
	fileTreeRowTbl.Rows = [][]string{
		{"Name"},
	}
	for _, item := range rows {
		fileTreeRowTbl.Rows = append(fileTreeRowTbl.Rows, []string{item[0]})
	}
	fileTreeRowTbl.TextStyle = ui.NewStyle(ui.ColorWhite)
	fileTreeRowTbl.FillRow = true

	keyboardList := widgets.NewParagraph()
	keyboardList.Title = " Keyboard Shortcuts "
	keyboardList.Text = `[<enter>](fg:green) Copy command  [<ctrl+e>](fg:green) Send to terminal  [<ctrl+r>](fg:green) Reset input  [<tab>](fg:green) Switch panels  [<up/down>](fg:green) Navigate  [<ctrl+u>](fg:green) Insert command  [<ctrl+j/k>](fg:green) Jump first/last  [<F1>](fg:green) Show help  [<ctrl+z>](fg:green) Copy text  [<esc>](fg:green) Quit`
	keyboardList.TextStyle.Fg = ui.ColorWhite
	keyboardList.BorderStyle.Fg = ui.ColorWhite

	// 1. Create the input paragraph
	inputPara := widgets.NewParagraph()
	inputPara.Title = " Type Command "
	inputPara.Text = ""
	inputPara.TextStyle.Bg = ui.ColorBlue
	inputPara.TextStyle.Fg = ui.ColorBlack
	inputPara.BorderStyle = ui.NewStyle(ui.ColorYellow)

	// List to show matching results
	suggestionList := widgets.NewList()
	suggestionList.Title = " Recalled From History ⚡ "
	suggestionList.Rows = []string{}
	suggestionList.SelectedRow = 0
	suggestionList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorGreen)
	suggestionList.BorderStyle = ui.NewStyle(ui.ColorCyan)

	// Create a widget to show help text of a command
	helpList := widgets.NewList()
	helpList.Title = " Help Doc "
	helpList.Rows = []string{"Select a command to display the help text"}
	helpList.SelectedRow = 0
	helpList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorYellow)
	helpList.WrapText = true

	// Create a widget for AI output
	aiResponsePara := widgets.NewParagraph()
	aiResponsePara.Title = " AI Doc "
	aiResponsePara.Text = ""
	aiResponsePara.TextStyle.Fg = ui.ColorWhite

	// === Layout with Grid ===
	termWidth, termHeight := ui.TerminalDimensions()
	grid := ui.NewGrid()
	grid.SetRect(0, 0, termWidth, termHeight)

	showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
	// 4. Render initial UI
	ui.Render(grid)

	focusOnHelp := false
	uiEvents := ui.PollEvents()
	inputBuffer := "" // We'll store typed characters here
	selectedIndex := 0
	lastSearchQuery := "" // Cache last search to avoid redundant operations

	// Helper function to update search results
	updateSearchResults := func(query string) {
		if query == lastSearchQuery {
			return // Skip if query hasn't changed
		}
		lastSearchQuery = query
		matches := SearchWithRanking(tree, query, config.EnableFuzzing)
		suggestionList.Rows = suggestionList.Rows[:0] // Reuse slice to reduce allocations
		for _, node := range matches {
			suggestionList.Rows = append(suggestionList.Rows, node.Command)
		}

		// Update selectedIndex bounds
		if selectedIndex >= len(suggestionList.Rows) {
			selectedIndex = 0
		}
		if selectedIndex < 0 {
			selectedIndex = 0
		}
		suggestionList.SelectedRow = selectedIndex

		// Auto-load help text for the selected command
		if len(suggestionList.Rows) > 0 {
			selectedCmd := suggestionList.Rows[selectedIndex]
			helpList.SelectedRow = 0 // Reset help scroll to top
			repaintHelpWidget(hc, helpList, selectedCmd)
		}

		ui.Render(grid)
	}

	// Perform initial search
	updateSearchResults(inputBuffer)
	// Start a ticker to update clock on the app
	go func() {
		for {
			select {
			case <-done:
				return
			case <-searchDebouncer.C:
				// Debounced search execution
				updateSearchResults(inputBuffer)
			}
		}
	}()

	for {
		e := <-uiEvents
		switch e.ID {
		case "<C-c>", "<Escape>":
			// Ctrl-C or Escape to exit
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
			// CHANGED: Press Tab or Shift to toggle focus
			focusOnHelp = !focusOnHelp
			toggleBorders(suggestionList, helpList)
		case "<Backspace>":
			// Remove the last character from input
			if len(inputBuffer) > 0 {
				inputBuffer = inputBuffer[:len(inputBuffer)-1]
			}
			// Reset and restart debounce timer
			searchDebouncer.Reset(debounceDelay)
		case "<Space>":
			// Specifically handle space
			inputBuffer += " "
			// Reset and restart debounce timer
			searchDebouncer.Reset(debounceDelay)
		case "<Enter>":
			var commandToCopy string
			if len(suggestionList.Rows) > 0 {
				commandToCopy = suggestionList.Rows[selectedIndex]
			} else {
				commandToCopy = inputBuffer
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
			// Ctrl+E to send command directly to terminal
			var commandToSend string
			if len(suggestionList.Rows) > 0 {
				commandToSend = suggestionList.Rows[selectedIndex]
			} else {
				commandToSend = inputBuffer
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
			if focusOnHelp {
				// Scroll helpList up
				if helpList.SelectedRow > 0 {
					helpList.SelectedRow--
				}
			} else {
				// Move selection up in suggestionList
				if selectedIndex > 0 {
					selectedIndex--
					suggestionList.SelectedRow = selectedIndex
					selectedCmd := suggestionList.Rows[selectedIndex]
					// Reset Help page to Top
					helpList.SelectedRow = 0
					repaintHelpWidget(hc, helpList, selectedCmd)
					showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
				}
			}
		case "<Down>":
			if focusOnHelp {
				if helpList.SelectedRow < len(helpList.Rows)-1 {
					helpList.SelectedRow++
				}
			} else {
				// Move selection down in suggestionList
				if selectedIndex < len(suggestionList.Rows)-1 {
					selectedIndex++
					suggestionList.SelectedRow = selectedIndex
					selectedCmd := suggestionList.Rows[selectedIndex]
					// Reset Help page to Top
					helpList.SelectedRow = 0
					repaintHelpWidget(hc, helpList, selectedCmd)
					showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
				}
			}
		case "<F1>":
			var selectedCmd string
			// Fetch help for the highlighted command
			if len(suggestionList.Rows) > 0 {
				selectedCmd = suggestionList.Rows[selectedIndex]
			} else {
				selectedCmd = inputPara.Text
			}

			repaintHelpWidget(hc, helpList, selectedCmd)
			showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara, keyboardList)
		case "<C-u>":
			if !focusOnHelp {
				inputBuffer = suggestionList.Rows[selectedIndex]
			}
		case "<C-r>":
			if !focusOnHelp {
				inputBuffer = ""
			}
		case "<C-j>":
			// Go to the last line
			if !focusOnHelp {
				if len(suggestionList.Rows) > 0 {
					selectedIndex = len(suggestionList.Rows) - 1
					suggestionList.SelectedRow = selectedIndex
				}
			} else {
				if len(helpList.Rows) > 0 {
					helpList.SelectedRow = len(helpList.Rows) - 1
				}
			}
		// case "<C-e>":
		// 	showAIWidget(grid, inputPara, suggestionList, helpList, datetimePara, aiResponsePara)
		// 	ui.Render(grid)
		// 	helpList.Rows = []string{}
		// 	var sc string
		// 	if len(suggestionList.Rows) > 0 {
		// 		sc = suggestionList.Rows[selectedIndex]
		// 	} else {
		// 		sc = inputPara.Text
		// 	}

		// 	prompt := preparePrompt(&PromptVars{
		// 		SelectedCommand: sc,
		// 		HelpText:        GetOrfillCache(hc, sc),
		// 	})

		// 	var max_t int = 500
		// 	stream, err := client.ChatStream(
		// 		context.TODO(),
		// 		&cohere.ChatStreamRequest{
		// 			Message:   prompt,
		// 			MaxTokens: &max_t,
		// 		},
		// 	)
		// 	if err != nil {
		// 		fmt.Println(err)
		// 	}

		// 	// Make sure to close the stream when you're done reading.
		// 	// This is easily handled with defer.
		// 	defer stream.Close()
		// 	for {
		// 		message, err := stream.Recv()

		// 		if errors.Is(err, io.EOF) {
		// 			// An io.EOF error means the server is done sending messages
		// 			// and should be treated as a success.
		// 			break
		// 		}
		// 		if err != nil {
		// 			// The stream has encountered a non-recoverable error. Propagate the
		// 			// error by simply returning the error like usual.
		// 			fmt.Println(err)
		// 			break
		// 		}
		// 		aiResponsePara.Text = aiResponsePara.Text + message.TextGeneration.GetText()
		// 		ui.Render(aiResponsePara)
		// 	}

		case "<C-k>":
			// Go to the first line
			if !focusOnHelp {
				selectedIndex = 0
				suggestionList.SelectedRow = selectedIndex
			} else {
				if len(helpList.Rows) > 0 {
					helpList.SelectedRow = 0
				}
			}
		case "<Resize>":
			// Adjust layout when the terminal size changes
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
			// Typically a typed character
			if !focusOnHelp {
				if e.Type == ui.KeyboardEvent && len(e.ID) == 1 {
					// Add typed character to input
					inputBuffer += e.ID
					// Reset and restart debounce timer
					searchDebouncer.Reset(debounceDelay)
				}
			}
		}

		// Update the paragraph to show the current input
		inputPara.Text = inputBuffer

		// Re-render UI (search results updated via debouncer)
		ui.Render(grid)
	}
}

// getSuggestions searches through file tree and returns list of matches
// of commandRecommendLimit length
func getSuggestions(searchStr string, tree *AVLTree, enableFuzzing bool) []string {
	matches := SearchWithRanking(tree, searchStr, enableFuzzing)
	results := []string{}

	count := 0
	for _, node := range matches {
		results = append(results, fmt.Sprintf("%s", node.Command))
		count++
	}

	return results
}
