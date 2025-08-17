// app.go

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
	"time"

	"github.com/atotto/clipboard"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	tb "github.com/nsf/termbox-go"
	"github.com/patrickmn/go-cache"
)

// DisableMouseInput in termbox-go. This should be called after ui.Init()
func DisableMouseInput() {
	tb.SetInputMode(tb.InputEsc)
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
		ui.NewRow(0.95,
			ui.NewCol(0.3,
				ui.NewRow(0.2, inputPara),
				ui.NewRow(0.8, suggestionList),
			),
			ui.NewCol(0.7, aiResponsePara),
		),
		ui.NewRow(0.05, keyboardList),
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
		ui.NewRow(0.95,
			ui.NewCol(0.3,
				ui.NewRow(0.2, inputPara),
				ui.NewRow(0.8, suggestionList),
			),
			ui.NewCol(0.7, helpList),
		),
		ui.NewRow(0.05, keyboardList),
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
	keyboardList.Text = `[<enter>](fg:green) Execute command  [<ctrl+r>](fg:green) Reset input  [<tab>](fg:green) Switch panels  [<up/down>](fg:green) Navigate  [<ctrl+u>](fg:green) Insert command  [<ctrl+j/k>](fg:green) Jump first/last  [<F1>](fg:green) Show help  [<ctrl+z>](fg:green) Copy text  [<esc>](fg:green) Quit`
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
		matches := SearchWithRanking(tree, query)
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
			ui.Close()
			if len(suggestionList.Rows) > 0 {
				selectedCommand := suggestionList.Rows[selectedIndex]
				fmt.Println(fmt.Sprintf("Trying to run command: %s", selectedCommand))
				execCommandInPTY(selectedCommand)
			} else {
				execCommandInPTY(inputBuffer)
			}
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

// getSuggestions searches through file tree and returns list of macthes
// of commandRecommendLimit length
func getSuggestions(searchStr string, tree *AVLTree) []string {
	matches := SearchWithRanking(tree, searchStr)
	results := []string{}

	count := 0
	for _, node := range matches {
		results = append(results, fmt.Sprintf("%s", node.Command))
		count++
	}

	return results
}
