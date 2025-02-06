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
	"os"
	"os/exec"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	tb "github.com/nsf/termbox-go"
	"github.com/patrickmn/go-cache"
)

func main() {
	helpCache := cache.New(cache.DefaultExpiration, cache.DefaultExpiration)

	tree := NewAVLTree()
	if err := readHistoryAndPopulateTree(tree); err != nil {
		log.Fatalf("Error reading history: %v", err)
	}
	run(tree, helpCache)
	// res, _ := getCommandHelp("aws")

	// fmt.Println(res)
}

// DisableMouseInput in termbox-go. This should be called after ui.Init()
func DisableMouseInput() {
	tb.SetInputMode(tb.InputEsc)
}

// getBanner creates a datetime message
func getBanner(t time.Time) string {
	d := DaysToWeekend()
	msg := ""

	if d == 0 {
		msg = "Enjoy your weekend!"
	} else {
		msg = fmt.Sprintf("%d days to Weekend! 🌴", d)
	}
	return fmt.Sprintf("%s. %s", FormatDateTime(t), msg)
}

// getPaddedQuote adds before and after padding to a quote
func getPaddedQuote(quote string) string {
	return " " + quote + " "
}

func repaintHelpWidget(g *ui.Grid, c *cache.Cache, l *widgets.List, cmd string) {
	help, err := splitCommand(cmd)
	if err != nil {
		log.Fatalf("Cannot repaint the widget due to: %v", err)
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

	l.Rows = strings.Split(helpTxt, "\n")
	// Re-render the help widget (along with others)
	ui.Render(g)
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

func run(tree *AVLTree, hc *cache.Cache) {
	// Done channel for ticker
	done := make(chan bool)

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	DisableMouseInput()
	defer ui.Close()

	datetimeRowList := widgets.NewList()
	datetimeRowList.Title = "Today"
	datetimeRowList.Rows = []string{
		getBanner(time.Now()),
		"",
		getPaddedQuote(GetRandomQuote()),
	}
	datetimeRowList.SelectedRow = 2
	datetimeRowList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorBlue)

	// 1. Create the input paragraph
	inputPara := widgets.NewParagraph()
	inputPara.Title = " Type Command "
	inputPara.Text = ""

	// List to show matching results
	suggestionList := widgets.NewList()
	suggestionList.Title = " Recalled From History 🍔 "
	suggestionList.Rows = []string{}
	suggestionList.SelectedRow = 0
	suggestionList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorGreen)

	// Create a widget to show help text of a command
	helpList := widgets.NewList()
	helpList.Title = " Help Text "
	helpList.Rows = []string{"Press <F1> or <fn + 1> for help on the selected command."}
	helpList.SelectedRow = 0
	helpList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorYellow)
	helpList.BorderStyle = ui.NewStyle(ui.ColorCyan)

	// === Layout with Grid ===
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	// We create 1 row with 2 columns: 40% for suggestions, 60% for help
	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(0.4, ui.NewRow(0.1, inputPara), ui.NewRow(0.9, suggestionList)),
			ui.NewCol(0.6, ui.NewRow(0.10, datetimeRowList), ui.NewRow(0.90, helpList)),
		),
	)

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
		case "<Tab>", "<Shift>":
			// CHANGED: Press Tab or Shift to toggle focus
			focusOnHelp = !focusOnHelp
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
					repaintHelpWidget(grid, hc, helpList, selectedCmd)
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
					repaintHelpWidget(grid, hc, helpList, selectedCmd)
				}
			}
		case "<PageUp>":
			if focusOnHelp && len(helpList.Rows) > 0 {
				// Jump to top of help list
				helpList.SelectedRow = 0
			}
		case "<PageDown>":
			if focusOnHelp && len(helpList.Rows) > 0 {
				// Jump to bottom of help list
				helpList.SelectedRow = len(helpList.Rows) - 1
			}
		case "<F1>":
			// Fetch help for the highlighted command
			if len(suggestionList.Rows) > 0 {
				selectedCmd := suggestionList.Rows[selectedIndex]
				repaintHelpWidget(grid, hc, helpList, selectedCmd)
			}
		case "<Resize>":
			// If you need to handle resizing, do so here
		default:
			// Typically a typed character
			if e.Type == ui.KeyboardEvent && len(e.ID) == 1 {
				// Add typed character to input
				inputBuffer += e.ID
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
