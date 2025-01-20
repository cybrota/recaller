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

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func main() {
	tree := NewAVLTree()
	if err := readHistoryAndPopulateTree(tree); err != nil {
		log.Fatalf("Error reading history: %v", err)
	}
	run(tree)
	// res, _ := getCommandHelp("aws")

	// fmt.Println(res)
}

func run(tree *AVLTree) {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// 1. Create the input paragraph
	inputPara := widgets.NewParagraph()
	inputPara.Title = " Type Command "
	inputPara.Text = ""
	inputPara.SetRect(0, 0, 100, 3) // x1, y1, x2, y2

	// List to show matching results
	suggestionList := widgets.NewList()
	suggestionList.Title = " Recalled From History 🍔 "
	suggestionList.Rows = []string{}
	suggestionList.SelectedRow = 0
	suggestionList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorGreen)
	suggestionList.SetRect(0, 3, 100, 15)

	// Create a widget to show help text of a command
	helpPara := widgets.NewParagraph()
	helpPara.Title = " Help Text "
	helpPara.Text = "Press <F1> or <fn + 1> for help on the selected command."
	// Position it on the right side or below the existing widgets
	helpPara.SetRect(105, 3, 200, 15)
	helpPara.BorderStyle = ui.NewStyle(ui.ColorCyan)

	// 4. Render initial UI
	ui.Render(inputPara, suggestionList, helpPara)

	// 5. Main event loop
	uiEvents := ui.PollEvents()
	inputBuffer := "" // We'll store typed characters here
	selectedIndex := 0

	for {
		e := <-uiEvents
		switch e.ID {
		case "<C-c>", "<Escape>":
			// Ctrl-C or Escape to exit
			return
		case "<Backspace>":
			// Remove the last character from input
			if len(inputBuffer) > 0 {
				inputBuffer = inputBuffer[:len(inputBuffer)-1]
			}
		case "<Space>":
			// Specifically handle space
			inputBuffer += " "
		case "<Enter>":
			if len(suggestionList.Rows) > 0 {
				selectedCommand := suggestionList.Rows[selectedIndex]

				// 1. Close termui so the terminal is back to normal
				ui.Close()

				// 2. Launch the command in a “shell-like” environment
				cmd := exec.Command("sh", "-c", selectedCommand)
				// If you want actual Bash + history appends, use:
				// cmd := exec.Command("bash", "-ic", selectedCommand+"; history -a")

				// 3. Attach stdio so user sees output, usage, etc.
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
		case "<Up>":
			// Move selection up
			if selectedIndex > 0 {
				selectedIndex--
			}

		case "<Down>":
			// Move selection down
			if selectedIndex < len(suggestionList.Rows)-1 {
				selectedIndex++
			}
		case "<F1>":
			// Fetch help for the highlighted command
			if len(suggestionList.Rows) > 0 {
				selectedCmd := suggestionList.Rows[selectedIndex]

				helpText, err := getCommandHelp(extractCommandName(selectedCmd))
				if err != nil {
					helpPara.Text = "Relax and take a deep breath. " + err.Error()
				} else {
					helpPara.Text = helpText
				}

				// Re-render the help widget (along with others)
				ui.Render(inputPara, suggestionList, helpPara)
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
		matches := tree.SearchPrefixMostRecent(inputBuffer)
		suggestionList.Rows = []string{}
		for _, node := range matches {
			suggestionList.Rows = append(suggestionList.Rows, node.Key)
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
		ui.Render(inputPara, suggestionList)
	}
}
