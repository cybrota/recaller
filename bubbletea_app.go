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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/patrickmn/go-cache"
)

// BubbleTeaMode represents different UI modes
type BubbleTeaMode int

const (
	ModeHistory BubbleTeaMode = iota
	ModeFilesystem
)

// Filter modes for filesystem search
const (
	FilterModeAll = iota
	FilterModeDirs
	FilterModeFiles
)

// Model represents the Bubble Tea application state
type Model struct {
	mode  BubbleTeaMode
	ready bool

	// History search components
	textInput       textinput.Model
	suggestionsList list.Model
	helpViewport    viewport.Model

	// Filesystem search components
	filesystemInput  textinput.Model
	filesList        list.Model
	metadataViewport viewport.Model

	// Data
	tree      *AVLTree
	helpCache *cache.Cache
	config    *Config
	fsIndexer *FilesystemIndexer

	// State
	focusIndex  int
	suggestions []string
	lastQuery   string
	focusOnHelp bool // True when help viewport is focused for navigation

	// Filesystem state
	filesystemFocusIndex int // 0: input, 1: files list, 2: metadata
	filterMode           int // FilterModeAll, FilterModeDirs, FilterModeFiles
	currentFiles         []RankedFile
	selectedFileIndex    int
	lastFilesystemQuery  string

	// Styling
	styles          *Styles
	glamourRenderer *glamour.TermRenderer

	// Dimensions
	width  int
	height int
}

// Styles holds all the styling for the application
type Styles struct {
	BorderFocused  lipgloss.Style
	BorderBlurred  lipgloss.Style
	Title          lipgloss.Style
	InputPrompt    lipgloss.Style
	HelpKey        lipgloss.Style
	HelpDesc       lipgloss.Style
	SuccessMessage lipgloss.Style
	ErrorMessage   lipgloss.Style
}

// NewStyles creates the default styles
func NewStyles() *Styles {
	return &Styles{
		BorderFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Bold(true),
		BorderBlurred: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // Bright cyan/blue, more visible on dark backgrounds
			Padding(0, 1).
			Bold(true),
		InputPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true),
		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")),
		SuccessMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		ErrorMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
	}
}

// suggestionItem represents an item in the suggestions list
type suggestionItem struct {
	command string
}

func (i suggestionItem) FilterValue() string { return i.command }
func (i suggestionItem) Title() string       { return i.command }
func (i suggestionItem) Description() string { return "" }

// fileItem represents an item in the files list
type fileItem struct {
	rankedFile RankedFile
}

func (i fileItem) FilterValue() string { return filepath.Base(i.rankedFile.Path) }
func (i fileItem) Title() string       { return filepath.Base(i.rankedFile.Path) }
func (i fileItem) Description() string {
	metadata := i.rankedFile.Metadata
	if metadata.IsDirectory {
		return fmt.Sprintf("📁 %s", filepath.Dir(i.rankedFile.Path))
	}
	return fmt.Sprintf("📄 %s", filepath.Dir(i.rankedFile.Path))
}

// InitialModel creates the initial model
func InitialModel(tree *AVLTree, hc *cache.Cache, fsIndexer *FilesystemIndexer, mode BubbleTeaMode) Model {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		config = &Config{History: HistoryConfig{EnableFuzzing: true}}
	}

	// Initialize text input for history search
	ti := textinput.New()
	ti.Placeholder = "Type command to search..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// Initialize suggestions list
	items := []list.Item{}
	suggestionsList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	suggestionsList.SetShowTitle(false) // Completely disable built-in title rendering
	suggestionsList.SetShowHelp(false)

	// Initialize help viewport
	helpViewport := viewport.New(0, 0)
	helpViewport.SetContent("Select a command to see help documentation...")

	// Initialize filesystem components
	fsInput := textinput.New()
	fsInput.Placeholder = "Type to search files and directories..."
	fsInput.CharLimit = 256
	fsInput.Width = 50

	fileItems := []list.Item{}
	filesList := list.New(fileItems, list.NewDefaultDelegate(), 0, 0)
	filesList.SetShowTitle(false) // Completely disable built-in title rendering
	filesList.SetShowHelp(false)

	metadataViewport := viewport.New(0, 0)
	metadataViewport.SetContent("Select a file to view details...")

	// Initialize glamour renderer with auto-detection
	glamourRenderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(72),
	)

	// Set focus based on mode
	if mode == ModeFilesystem {
		ti.Blur()
		fsInput.Focus()
	}

	model := Model{
		mode:                 mode,
		textInput:            ti,
		suggestionsList:      suggestionsList,
		helpViewport:         helpViewport,
		filesystemInput:      fsInput,
		filesList:            filesList,
		metadataViewport:     metadataViewport,
		tree:                 tree,
		helpCache:            hc,
		config:               config,
		fsIndexer:            fsIndexer,
		focusIndex:           0,
		filesystemFocusIndex: 0,
		filterMode:           FilterModeAll,
		currentFiles:         []RankedFile{},
		selectedFileIndex:    0,
		styles:               NewStyles(),
		glamourRenderer:      glamourRenderer,
		suggestions:          []string{},
		lastQuery:            "",
		lastFilesystemQuery:  "",
	}

	return model
}

// Init is called when the program starts
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all the I/O
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "f2":
			// Switch between modes
			if m.mode == ModeHistory {
				m.mode = ModeFilesystem
				m.textInput.Blur()
				m.filesystemInput.Focus()
			} else {
				m.mode = ModeHistory
				m.filesystemInput.Blur()
				m.textInput.Focus()
			}
			return m, nil
		}

		// Handle mode-specific key events
		if m.mode == ModeHistory {
			return m.updateHistoryMode(msg)
		} else {
			return m.updateFilesystemMode(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.ready = true
	}

	return m, tea.Batch(cmds...)
}

// updateHistoryMode handles key events for history search mode
func (m Model) updateHistoryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg.String() {
	case "tab":
		if m.focusOnHelp {
			// From help back to input (completing the cycle)
			m.focusOnHelp = false
			m.focusIndex = 0 // Back to input
		} else if m.focusIndex == 0 {
			// From input to suggestions
			m.focusIndex = 1
		} else {
			// From suggestions to help
			m.focusOnHelp = true
			// Keep focusIndex as 1 so we know we came from suggestions
		}
	case "enter":
		if m.focusIndex == 0 {
			// Handle search input - do nothing special, just let user continue typing
			return m, nil
		} else {
			// Handle command selection from list
			if len(m.suggestions) > 0 {
				selectedIndex := m.suggestionsList.Index()
				if selectedIndex >= 0 && selectedIndex < len(m.suggestions) {
					selectedCommand := m.suggestions[selectedIndex]
					// Copy command to clipboard and quit
					return m, tea.Sequence(
						func() tea.Msg {
							copyToClipboard(selectedCommand)
							return tea.Quit()
						},
					)
				}
			}
		}
	case "ctrl+e":
		// Send to terminal
		if len(m.suggestions) > 0 {
			selectedIndex := m.suggestionsList.Index()
			if selectedIndex >= 0 && selectedIndex < len(m.suggestions) {
				selectedCommand := m.suggestions[selectedIndex]
				return m, tea.Sequence(
					func() tea.Msg {
						sendToTerminal(selectedCommand)
						return tea.Quit()
					},
				)
			}
		}
	case "f1":
		// Show help for current command (like the original F1 functionality)
		var selectedCommand string
		if m.focusIndex == 0 {
			// Use the input text if focusing on input
			selectedCommand = m.textInput.Value()
		} else if len(m.suggestions) > 0 {
			// Use the selected suggestion
			selectedIndex := m.suggestionsList.Index()
			if selectedIndex >= 0 && selectedIndex < len(m.suggestions) {
				selectedCommand = m.suggestions[selectedIndex]
			}
		}
		if selectedCommand != "" {
			m.updateHelp(selectedCommand)
			m.focusOnHelp = true // Switch focus to help after showing it
		}
		return m, nil
	case "ctrl+z":
		// Copy selected help text (like original Ctrl+Z functionality)
		if m.focusOnHelp {
			helpContent := m.helpViewport.View()
			return m, tea.Sequence(
				func() tea.Msg {
					copyToClipboard(helpContent)
					return nil
				},
			)
		}
		return m, nil
	case "pgup":
		// Page up in help content
		if m.focusOnHelp {
			m.helpViewport.LineUp(m.helpViewport.Height)
			return m, nil
		}
	case "pgdown":
		// Page down in help content
		if m.focusOnHelp {
			m.helpViewport.LineDown(m.helpViewport.Height)
			return m, nil
		}
	case "home":
		// Go to top of help content
		if m.focusOnHelp {
			m.helpViewport.GotoTop()
			return m, nil
		}
	case "end":
		// Go to bottom of help content
		if m.focusOnHelp {
			m.helpViewport.GotoBottom()
			return m, nil
		}
	case "up", "k":
		if m.focusOnHelp {
			// Navigate help content
			m.helpViewport.LineUp(1)
			return m, nil
		} else if m.focusIndex == 1 && len(m.suggestions) > 0 {
			// Manual navigation for suggestions list
			if m.suggestionsList.Index() > 0 {
				m.suggestionsList.CursorUp()
			}
			// Update help when selection changes
			selectedIndex := m.suggestionsList.Index()
			if selectedIndex >= 0 && selectedIndex < len(m.suggestions) {
				m.updateHelp(m.suggestions[selectedIndex])
			}
			return m, nil
		}
	case "down", "j":
		if m.focusOnHelp {
			// Navigate help content
			m.helpViewport.LineDown(1)
			return m, nil
		} else if m.focusIndex == 1 && len(m.suggestions) > 0 {
			// Manual navigation for suggestions list
			if m.suggestionsList.Index() < len(m.suggestions)-1 {
				m.suggestionsList.CursorDown()
			}
			// Update help when selection changes
			selectedIndex := m.suggestionsList.Index()
			if selectedIndex >= 0 && selectedIndex < len(m.suggestions) {
				m.updateHelp(m.suggestions[selectedIndex])
			}
			return m, nil
		}
	}

	// Update components based on focus
	if m.focusOnHelp {
		// When help is focused, let viewport handle scrolling (already handled above)
		msgStr := msg.String()
		if msgStr != "up" && msgStr != "down" && msgStr != "k" && msgStr != "j" {
			m.helpViewport, cmd = m.helpViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else if m.focusIndex == 0 {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

		// Update suggestions when text changes
		currentQuery := m.textInput.Value()
		if currentQuery != m.lastQuery {
			m.updateSuggestions(currentQuery)
			m.lastQuery = currentQuery
		}
	} else {
		// Only let the list handle non-navigation keys
		msgStr := msg.String()
		if msgStr != "up" && msgStr != "down" && msgStr != "k" && msgStr != "j" {
			m.suggestionsList, cmd = m.suggestionsList.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateFilesystemMode handles key events for filesystem search mode
func (m Model) updateFilesystemMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg.String() {
	case "tab":
		m.filesystemFocusIndex = (m.filesystemFocusIndex + 1) % 3
	case "enter":
		if m.filesystemFocusIndex == 1 && len(m.currentFiles) > 0 {
			// Open selected file
			selectedFile := m.currentFiles[m.selectedFileIndex]
			m.fsIndexer.AddPath(selectedFile.Path, time.Now())

			return m, tea.Sequence(
				func() tea.Msg {
					if err := openFileWithDefaultApp(selectedFile.Path); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to open file: %v\n", err)
					} else {
						fmt.Printf("🚀 Opened: %s\n", selectedFile.Path)
					}
					// Persist index in background
					go func() {
						if err := m.fsIndexer.PersistIndex(!m.config.Quiet); err != nil {
							fmt.Fprintf(os.Stderr, "Failed to persist index: %v\n", err)
						}
					}()
					return tea.Quit()
				},
			)
		}
	case "ctrl+x":
		if m.filesystemFocusIndex == 1 && len(m.currentFiles) > 0 {
			// Copy selected file path
			selectedFile := m.currentFiles[m.selectedFileIndex]
			return m, tea.Sequence(
				func() tea.Msg {
					if err := copyToClipboard(selectedFile.Path); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to copy path: %v\n", err)
					} else {
						fmt.Printf("📋 Copied path: %s\n", selectedFile.Path)
					}
					return tea.Quit()
				},
			)
		}
	case "ctrl+t":
		// Toggle filter mode
		m.filterMode = (m.filterMode + 1) % 3
		m.updateFilesystemResults()
		m.updateFilesListTitle()
	case "ctrl+r":
		// Reset input
		if m.filesystemFocusIndex == 0 {
			m.filesystemInput.SetValue("")
		}
	case "up", "k":
		if m.filesystemFocusIndex == 1 {
			if m.selectedFileIndex > 0 {
				m.selectedFileIndex--
				// Sync the list cursor
				if m.filesList.Index() > 0 {
					m.filesList.CursorUp()
				}
				m.updateMetadataContent()
			}
		} else if m.filesystemFocusIndex == 2 {
			m.metadataViewport.LineUp(1)
		}
	case "down", "j":
		if m.filesystemFocusIndex == 1 {
			if m.selectedFileIndex < len(m.currentFiles)-1 {
				m.selectedFileIndex++
				// Sync the list cursor
				if m.filesList.Index() < len(m.currentFiles)-1 {
					m.filesList.CursorDown()
				}
				m.updateMetadataContent()
			}
		} else if m.filesystemFocusIndex == 2 {
			m.metadataViewport.LineDown(1)
		}
	case "ctrl+k":
		if m.filesystemFocusIndex == 1 {
			m.selectedFileIndex = 0
			// Reset list cursor to top
			for m.filesList.Index() > 0 {
				m.filesList.CursorUp()
			}
			m.updateMetadataContent()
		}
	case "ctrl+j":
		if m.filesystemFocusIndex == 1 {
			if len(m.currentFiles) > 0 {
				m.selectedFileIndex = len(m.currentFiles) - 1
				// Move list cursor to bottom
				for m.filesList.Index() < len(m.currentFiles)-1 {
					m.filesList.CursorDown()
				}
				m.updateMetadataContent()
			}
		}
	}

	// Update components based on focus
	if m.filesystemFocusIndex == 0 {
		m.filesystemInput, cmd = m.filesystemInput.Update(msg)
		cmds = append(cmds, cmd)

		// Update file results when text changes
		currentQuery := m.filesystemInput.Value()
		if currentQuery != m.lastFilesystemQuery {
			m.updateFilesystemResults()
			m.lastFilesystemQuery = currentQuery
		}
	} else if m.filesystemFocusIndex == 1 {
		// Only let the list handle non-navigation keys
		msgStr := msg.String()
		if msgStr != "up" && msgStr != "down" && msgStr != "k" && msgStr != "j" {
			m.filesList, cmd = m.filesList.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		m.metadataViewport, cmd = m.metadataViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.mode == ModeHistory {
		return m.renderHistoryView()
	} else {
		return m.renderFilesystemView()
	}
}

// renderHistoryView renders the history search view
func (m Model) renderHistoryView() string {
	// Ensure we have minimum dimensions
	if m.width < 20 || m.height < 10 {
		return "Terminal too small. Please resize your terminal."
	}

	// Calculate dimensions
	inputHeight := 3
	helpHeight := m.height - inputHeight - 6 // Leave more room for help text
	suggestionWidth := (m.width / 2) - 1
	helpWidth := m.width - suggestionWidth - 3

	// Style the text input
	var inputStyle lipgloss.Style
	var inputTitle string
	if m.focusIndex == 0 && !m.focusOnHelp {
		inputStyle = m.styles.BorderFocused
		inputTitle = " 🔍 Search Commands (Active)\n"
	} else {
		inputStyle = m.styles.BorderBlurred
		inputTitle = " 🔍 Search Commands\n"
	}

	// Ensure textInput has proper width
	m.textInput.Width = suggestionWidth - 4 // Account for borders and padding

	// Create input content with title
	inputContent := m.textInput.View()

	inputBox := inputStyle.
		Width(suggestionWidth).
		Height(inputHeight).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Title.Width(suggestionWidth-4).Render(inputTitle),
			inputContent,
		))

	// Style the suggestions list
	var listStyle lipgloss.Style
	var listTitle string
	if m.focusIndex == 1 && !m.focusOnHelp {
		listStyle = m.styles.BorderFocused
		listTitle = " 📋 Command History (Active) "
	} else {
		listStyle = m.styles.BorderBlurred
		listTitle = " 📋 Command History "
	}

	// Create suggestions content with title
	suggestionsContent := m.suggestionsList.View()

	suggestionBox := listStyle.
		Width(suggestionWidth).
		Height(helpHeight).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Title.Width(suggestionWidth-4).Render(listTitle),
			suggestionsContent,
		))

	// Style the help viewport
	var helpStyle lipgloss.Style
	var helpTitle string
	if m.focusOnHelp {
		helpStyle = m.styles.BorderFocused
		helpTitle = " 📖 Help Documentation (Active) "
	} else {
		helpStyle = m.styles.BorderBlurred
		helpTitle = " 📖 Help Documentation "
	}

	// Create help content with title
	helpContent := lipgloss.NewStyle().
		Bold(m.focusOnHelp).
		Render(m.helpViewport.View())

	helpBox := helpStyle.
		Width(helpWidth).
		Height(helpHeight + inputHeight + 2).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Title.Width(helpWidth-4).Render(helpTitle),
			helpContent,
		))

	// Combine left column
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		inputBox,
		suggestionBox,
	)

	// Combine everything horizontally
	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		helpBox,
	)

	// Add help footer
	help := m.renderHistoryHelp()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		main,
		help,
	)
}

// renderFilesystemView renders the filesystem search view
func (m Model) renderFilesystemView() string {
	// Ensure we have minimum dimensions
	if m.width < 30 || m.height < 10 {
		return "Terminal too small. Please resize your terminal."
	}

	// Calculate dimensions
	inputHeight := 3
	listHeight := m.height - inputHeight - 6 // Leave more room for help text
	leftWidth := (m.width * 4 / 10) - 1      // 40% for input + files
	rightWidth := m.width - leftWidth - 3    // 60% for metadata

	// Style the filesystem input
	var inputStyle lipgloss.Style
	var fsInputTitle string
	if m.filesystemFocusIndex == 0 {
		inputStyle = m.styles.BorderFocused
		fsInputTitle = " 📁 Search Files & Directories (Active) "
	} else {
		inputStyle = m.styles.BorderBlurred
		fsInputTitle = " 📁 Search Files & Directories "
	}

	// Ensure filesystemInput has proper width
	m.filesystemInput.Width = leftWidth - 4 // Account for borders and padding

	// Create filesystem input content with title
	fsInputContent := m.filesystemInput.View()

	inputBox := inputStyle.
		Width(leftWidth).
		Height(inputHeight).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Title.Width(leftWidth-4).Render(fsInputTitle),
			fsInputContent,
		))

	// Style the files list
	var filesListStyle lipgloss.Style
	var filesTitle string
	if m.filesystemFocusIndex == 1 {
		filesListStyle = m.styles.BorderFocused
		// Get the current filter title with active indicator
		filesTitle = m.getFilesListActiveTitle()
	} else {
		filesListStyle = m.styles.BorderBlurred
		filesTitle = m.getFilesListTitle()
	}

	// Create files list content with title
	filesContent := m.filesList.View()

	filesBox := filesListStyle.
		Width(leftWidth).
		Height(listHeight).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Title.Width(leftWidth-4).Render(filesTitle),
			filesContent,
		))

	// Style the metadata viewport
	var metadataStyle lipgloss.Style
	var metadataTitle string
	if m.filesystemFocusIndex == 2 {
		metadataStyle = m.styles.BorderFocused
		metadataTitle = " 📄 File Information (Active) "
	} else {
		metadataStyle = m.styles.BorderBlurred
		metadataTitle = " 📄 File Information "
	}

	// Create metadata content with title
	metadataContent := m.metadataViewport.View()

	metadataBox := metadataStyle.
		Width(rightWidth).
		Height(inputHeight + listHeight + 2).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Title.Width(rightWidth-4).Render(metadataTitle),
			metadataContent,
		))

	// Combine left column
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		inputBox,
		filesBox,
	)

	// Combine everything horizontally
	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		metadataBox,
	)

	// Add help footer
	help := m.renderFilesystemHelp()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		main,
		help,
	)
}

// updateLayout updates component dimensions
func (m *Model) updateLayout() {
	if m.mode == ModeHistory {
		inputHeight := 3
		helpHeight := m.height - inputHeight - 6 // Leave room for help text
		suggestionWidth := (m.width / 2) - 1
		helpWidth := m.width - suggestionWidth - 3

		// Set text input width
		m.textInput.Width = suggestionWidth - 4

		// Set component sizes
		m.suggestionsList.SetSize(suggestionWidth-2, helpHeight-2)
		m.helpViewport.Width = helpWidth - 2
		m.helpViewport.Height = helpHeight + inputHeight
	} else {
		inputHeight := 3
		listHeight := m.height - inputHeight - 6
		leftWidth := (m.width * 4 / 10) - 1
		rightWidth := m.width - leftWidth - 3

		// Set filesystem input width
		m.filesystemInput.Width = leftWidth - 4

		// Set component sizes
		m.filesList.SetSize(leftWidth-2, listHeight-2)
		m.metadataViewport.Width = rightWidth - 2
		m.metadataViewport.Height = inputHeight + listHeight
	}
}

// updateSuggestions updates the suggestions list based on query
func (m *Model) updateSuggestions(query string) {
	matches := SearchWithRanking(m.tree, query, m.config.History.EnableFuzzing)

	items := make([]list.Item, len(matches))
	m.suggestions = make([]string, len(matches))

	for i, match := range matches {
		items[i] = suggestionItem{command: match.Command}
		m.suggestions[i] = match.Command
	}

	m.suggestionsList.SetItems(items)

	// Update help for first item if available
	if len(matches) > 0 {
		m.updateHelp(matches[0].Command)
	}
}

// updateHelp updates the help viewport with command help
func (m *Model) updateHelp(command string) {
	helpTxt := GetOrfillCache(m.helpCache, command)

	// Try to render as markdown first
	if rendered, err := m.glamourRenderer.Render(helpTxt); err == nil {
		m.helpViewport.SetContent(rendered)
	} else {
		// Fall back to plain text
		m.helpViewport.SetContent(helpTxt)
	}
}

// renderHistoryHelp renders the help footer for history mode
func (m Model) renderHistoryHelp() string {
	var keys []string
	var descs []string

	keys = append(keys, "enter")
	descs = append(descs, "copy command")

	keys = append(keys, "ctrl+e")
	descs = append(descs, "send to terminal")

	keys = append(keys, "tab")
	descs = append(descs, "switch focus")

	keys = append(keys, "f1")
	descs = append(descs, "show help")

	keys = append(keys, "ctrl+z")
	descs = append(descs, "copy help text")

	keys = append(keys, "f2")
	descs = append(descs, "filesystem mode")

	keys = append(keys, "esc")
	descs = append(descs, "quit")

	var helpEntries []string
	for i, key := range keys {
		helpEntries = append(helpEntries,
			fmt.Sprintf("%s %s",
				m.styles.HelpKey.Render(key),
				m.styles.HelpDesc.Render(descs[i])))
	}

	return lipgloss.NewStyle().
		Padding(1, 0, 0, 2).
		Render(strings.Join(helpEntries, " • "))
}

// renderFilesystemHelp renders the help footer for filesystem mode
func (m Model) renderFilesystemHelp() string {
	var keys []string
	var descs []string

	keys = append(keys, "enter")
	descs = append(descs, "open file")

	keys = append(keys, "ctrl+x")
	descs = append(descs, "copy path")

	keys = append(keys, "ctrl+t")
	descs = append(descs, "toggle filter")

	keys = append(keys, "tab")
	descs = append(descs, "switch focus")

	keys = append(keys, "f2")
	descs = append(descs, "history mode")

	keys = append(keys, "esc")
	descs = append(descs, "quit")

	var helpEntries []string
	for i, key := range keys {
		helpEntries = append(helpEntries,
			fmt.Sprintf("%s %s",
				m.styles.HelpKey.Render(key),
				m.styles.HelpDesc.Render(descs[i])))
	}

	return lipgloss.NewStyle().
		Padding(1, 0, 0, 2).
		Render(strings.Join(helpEntries, " • "))
}

// updateFilesystemResults updates the files list based on query and filter
func (m *Model) updateFilesystemResults() {
	query := m.filesystemInput.Value()
	if m.fsIndexer == nil {
		return
	}

	// Search files using the filesystem indexer
	results := m.fsIndexer.SearchFiles(query, m.config.History.EnableFuzzing)

	// Apply filter
	var filteredResults []RankedFile
	for _, result := range results {
		switch m.filterMode {
		case FilterModeAll:
			filteredResults = append(filteredResults, result)
		case FilterModeDirs:
			if result.Metadata.IsDirectory {
				filteredResults = append(filteredResults, result)
			}
		case FilterModeFiles:
			if !result.Metadata.IsDirectory {
				filteredResults = append(filteredResults, result)
			}
		}
	}

	// Update current files and create list items
	m.currentFiles = filteredResults
	items := make([]list.Item, len(filteredResults))
	for i, file := range filteredResults {
		items[i] = fileItem{rankedFile: file}
	}

	m.filesList.SetItems(items)

	// Reset selection
	m.selectedFileIndex = 0

	// Update metadata for first item if available
	if len(filteredResults) > 0 {
		m.updateMetadataContent()
	} else {
		m.metadataViewport.SetContent("No files found matching your search.")
	}
}

// updateFilesListTitle updates the files list title based on filter mode
func (m *Model) updateFilesListTitle() {
	// This method is now replaced by getFilesListTitle() and getFilesListActiveTitle()
	// but kept for compatibility
}

// getFilesListTitle returns the files list title without active indicator
func (m *Model) getFilesListTitle() string {
	var filterIcon, filterName string
	switch m.filterMode {
	case FilterModeAll:
		filterIcon = "📁"
		filterName = "All Files & Directories"
	case FilterModeDirs:
		filterIcon = "📂"
		filterName = "Directories Only"
	case FilterModeFiles:
		filterIcon = "📄"
		filterName = "Files Only"
	}

	return fmt.Sprintf(" %s %s ", filterIcon, filterName)
}

// getFilesListActiveTitle returns the files list title with active indicator
func (m *Model) getFilesListActiveTitle() string {
	var filterIcon, filterName string
	switch m.filterMode {
	case FilterModeAll:
		filterIcon = "📁"
		filterName = "All Files & Directories"
	case FilterModeDirs:
		filterIcon = "📂"
		filterName = "Directories Only"
	case FilterModeFiles:
		filterIcon = "📄"
		filterName = "Files Only"
	}

	return fmt.Sprintf(" %s %s (Active) ", filterIcon, filterName)
}

// updateMetadataContent updates the metadata viewport with file details
func (m *Model) updateMetadataContent() {
	if len(m.currentFiles) == 0 || m.selectedFileIndex >= len(m.currentFiles) {
		m.metadataViewport.SetContent("Select a file to view details...")
		return
	}

	file := m.currentFiles[m.selectedFileIndex]
	metadata := file.Metadata

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s\n\n", filepath.Base(file.Path)))
	content.WriteString(fmt.Sprintf("**Full Path:** %s\n\n", file.Path))

	if metadata.IsDirectory {
		content.WriteString("**Type:** Directory 📁\n\n")
	} else {
		content.WriteString("**Type:** File 📄\n\n")
	}

	if metadata.Size > 0 {
		content.WriteString(fmt.Sprintf("**Size:** %s\n\n", formatFileSize(metadata.Size)))
	}

	if !metadata.LastModified.IsZero() {
		content.WriteString(fmt.Sprintf("**Modified:** %s\n\n", metadata.LastModified.Format("2006-01-02 15:04:05")))
	}

	if metadata.Timestamp != nil {
		content.WriteString(fmt.Sprintf("**Last Accessed:** %s\n\n", metadata.Timestamp.Format("2006-01-02 15:04:05")))
	}

	content.WriteString(fmt.Sprintf("**Access Count:** %d\n\n", metadata.AccessCount))
	content.WriteString(fmt.Sprintf("**Score:** %.2f\n\n", file.Score))

	if metadata.IsHidden {
		content.WriteString("**Hidden:** Yes\n\n")
	}

	if metadata.IsSymlink {
		content.WriteString("**Symlink:** Yes\n\n")
	}

	// Try to render as markdown
	if rendered, err := m.glamourRenderer.Render(content.String()); err == nil {
		m.metadataViewport.SetContent(rendered)
	} else {
		m.metadataViewport.SetContent(content.String())
	}
}

// copyToClipboard copies text to clipboard
func copyToClipboard(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "📋 Copied %s%s%s to clipboard.\n", Green, text, Reset)
	return nil
}

// runBubbleTeaApp starts the Bubble Tea application
func runBubbleTeaApp(tree *AVLTree, hc *cache.Cache, fsIndexer *FilesystemIndexer, mode BubbleTeaMode) error {
	// Initialize colors
	InitializeColors()
	Green, Info, Warning, Error, Reset = GetANSIColors()

	model := InitialModel(tree, hc, fsIndexer, mode)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := program.Run()
	return err
}
