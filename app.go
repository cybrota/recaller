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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	tb "github.com/nsf/termbox-go"
	"github.com/patrickmn/go-cache"
)

const commandRecommendLimit = 5

// DisableMouseInput in termbox-go. This should be called after ui.Init()
func DisableMouseInput() {
	tb.SetInputMode(tb.InputEsc)
}

// getBanner creates a datetime message
func getBanner(t time.Time) string {
	d := DaysToWeekend()
	msg := ""

	switch d {
	case 0:
		msg = "Enjoy your weekend! ☕"
	case 1:
		msg = fmt.Sprintf("%d day to Weekend! 🌴", d)
	default:
		msg = fmt.Sprintf("%d day to Weekend! 🌴", d)
	}
	return fmt.Sprintf("%s. %s", FormatDateTime(t), msg)
}

// getPaddedQuote adds before and after padding to a quote
func getPaddedQuote(quote string) string {
	return " " + quote + " "
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
	l.Rows = strings.Split(helpTxt, "\n")
}

func showAIWidget(
	grid *ui.Grid,
	inputPara *widgets.Paragraph,
	suggestionList *widgets.List,
	helpList *widgets.List,
	aiResponsePara *widgets.Paragraph,
) {
	helpList.Rows = []string{}
	grid.Set(
		ui.NewCol(0.3,
			ui.NewRow(0.2, inputPara),
			ui.NewRow(0.8, suggestionList),
		),
		ui.NewCol(0.7,
			ui.NewCol(1, aiResponsePara),
		),
	)
}

func showHelpWidget(
	grid *ui.Grid,
	inputPara *widgets.Paragraph,
	suggestionList *widgets.List,
	helpList *widgets.List,
	aiResponsePara *widgets.Paragraph,
) {
	aiResponsePara.Text = ""
	grid.Set(
		ui.NewCol(0.3,
			ui.NewRow(0.2, inputPara),
			ui.NewRow(0.8, suggestionList),
		),
		ui.NewCol(0.7,
			ui.NewCol(1, helpList),
		),
	)
}

func execCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Command error:", err)
		os.Exit(-1)
	}
	os.Exit(0)
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

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	DisableMouseInput()
	defer ui.Close()

	datetimeRowList := widgets.NewList()
	datetimeRowList.Title = " Today "
	datetimeRowList.Rows = []string{
		getBanner(time.Now()),
		"",
		getPaddedQuote(GetRandomQuote()),
	}
	datetimeRowList.SelectedRow = 2
	datetimeRowList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorBlue)
	datetimeRowList.WrapText = true

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
	keyboardList.Text = `[<enter>](fg:green) -> Execute a selected command and quit
[<ctrl> + r](fg:green) -> Reset command input
[<tab>](fg:green) -> Switch b/w call history and Help
[<up>/<down>](fg:green) -> Move up or down to select content and view help text
[<ctrl> + u](fg:green) -> Insert selected command to edit
[<esc>](fg:green) or [<ctrl> + c](fg:green) -> Quit Recaller`

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
	helpList.Title = " Command Doc "
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
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara)
	// 4. Render initial UI
	ui.Render(grid)

	focusOnHelp := false
	uiEvents := ui.PollEvents()
	inputBuffer := "" // We'll store typed characters here
	selectedIndex := 0

	dateTi := time.NewTicker(1 * time.Second)
	quoteTi := time.NewTicker(60 * time.Second)

	// Start a ticker to update clock on the app
	go func() {
		for {
			select {
			case <-done:
				return
			case t, _ := <-dateTi.C:
				datetimeRowList.Rows[0] = getBanner(t)
				ui.Render(datetimeRowList)
			case <-quoteTi.C:
				datetimeRowList.Rows[2] = getPaddedQuote(GetRandomQuote())
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
		case "<Space>":
			// Specifically handle space
			inputBuffer += " "
		case "<Enter>":
			ui.Close()
			if len(suggestionList.Rows) > 0 {
				selectedCommand := suggestionList.Rows[selectedIndex]
				execCommand(selectedCommand)
			} else {
				execCommand(inputBuffer)
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
					selectedCmd := suggestionList.Rows[selectedIndex]
					// Reset Help page to Top
					helpList.SelectedRow = 0
					repaintHelpWidget(hc, helpList, selectedCmd)
					showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara)
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
					selectedCmd := suggestionList.Rows[selectedIndex]
					// Reset Help page to Top
					helpList.SelectedRow = 0
					repaintHelpWidget(hc, helpList, selectedCmd)
					showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara)
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
			showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara)
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
				suggestionList.SelectedRow = len(suggestionList.Rows) - 1
			} else {
				if len(helpList.Rows) > 0 {
					helpList.SelectedRow = len(helpList.Rows) - 1
				}
			}
		// case "<C-e>":
		// 	showAIWidget(grid, inputPara, suggestionList, helpList, aiResponsePara)
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
				suggestionList.SelectedRow = 0
			} else {
				if len(helpList.Rows) > 0 {
					helpList.SelectedRow = 0
				}
			}
		case "<Resize>":
			// Re-render all widgets
			ui.Render(grid)
		default:
			// Typically a typed character
			if !focusOnHelp {
				if e.Type == ui.KeyboardEvent && len(e.ID) == 1 {
					// Add typed character to input
					inputBuffer += e.ID
				}
			}

			if len(suggestionList.Rows) > 0 {
				repaintHelpWidget(hc, helpList, suggestionList.Rows[0])
				showHelpWidget(grid, inputPara, suggestionList, helpList, aiResponsePara)
			}
		}

		// Update the paragraph to show the current input
		inputPara.Text = inputBuffer

		// Perform a new prefix search whenever input changes (or arrows, etc.)
		matches := SearchWithRanking(tree, inputBuffer)
		suggestionList.Rows = []string{}
		for _, node := range matches {
			suggestionList.Rows = append(suggestionList.Rows, fmt.Sprintf("%s", node.Command))
		}

		// Make sure the selectedIndex is still valid
		if selectedIndex >= len(suggestionList.Rows) {
			selectedIndex = len(suggestionList.Rows) - 1
		}
		if selectedIndex < 0 {
			selectedIndex = 0
		}
		suggestionList.SelectedRow = selectedIndex

		// Re-render all widgets
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
		if count == commandRecommendLimit {
			break
		}
		results = append(results, fmt.Sprintf("%s", node.Command))
		count++
	}

	return results
}
