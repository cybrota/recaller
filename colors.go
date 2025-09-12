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
	"os"
	"strings"

	ui "github.com/gizak/termui/v3"
)

type ColorScheme struct {
	Primary     ui.Color
	Secondary   ui.Color
	Accent      ui.Color
	Success     ui.Color
	Warning     ui.Color
	Error       ui.Color
	Info        ui.Color
	Background  ui.Color
	Surface     ui.Color
	OnPrimary   ui.Color
	OnSecondary ui.Color
	OnSurface   ui.Color
	Border      ui.Color
	BorderFocus ui.Color
	Text        ui.Color
	TextMuted   ui.Color
}

type TerminalMode int

const (
	TerminalModeUnknown TerminalMode = iota
	TerminalModeLight
	TerminalModeDark
)

var (
	currentColorScheme *ColorScheme
	detectedMode       TerminalMode
)

// detectTerminalMode attempts to detect whether the terminal is in light or dark mode
func detectTerminalMode() TerminalMode {
	// Check environment variables that might indicate the theme
	if colorScheme := os.Getenv("COLORFGBG"); colorScheme != "" {
		// COLORFGBG format is typically "foreground;background"
		// Higher background numbers usually indicate dark mode
		parts := strings.Split(colorScheme, ";")
		if len(parts) >= 2 {
			bg := parts[len(parts)-1]
			// Dark background colors are typically 0-8, light are 15, 7, etc.
			if bg == "0" || bg == "8" || bg == "16" {
				return TerminalModeDark
			} else if bg == "15" || bg == "7" || bg == "255" {
				return TerminalModeLight
			}
		}
	}

	// Check TERM_THEME environment variable (some terminals set this)
	if theme := os.Getenv("TERM_THEME"); theme != "" {
		theme = strings.ToLower(theme)
		if strings.Contains(theme, "dark") {
			return TerminalModeDark
		} else if strings.Contains(theme, "light") {
			return TerminalModeLight
		}
	}

	// Check other common environment variables
	if theme := os.Getenv("THEME"); theme != "" {
		theme = strings.ToLower(theme)
		if strings.Contains(theme, "dark") {
			return TerminalModeDark
		} else if strings.Contains(theme, "light") {
			return TerminalModeLight
		}
	}

	// Default to dark mode as it's more common in terminals
	return TerminalModeDark
}

// createLightColorScheme returns a color scheme optimized for light terminals
func createLightColorScheme() *ColorScheme {
	return &ColorScheme{
		Primary:     ui.Color(4), // Dark Blue for better contrast with white text
		Secondary:   ui.Color(6), // Dark Cyan
		Accent:      ui.ColorMagenta,
		Success:     ui.Color(2), // Dark Green for better contrast
		Warning:     ui.Color(3), // Dark Yellow/Orange for better contrast
		Error:       ui.ColorRed,
		Info:        ui.Color(4), // Dark Blue
		Background:  ui.ColorWhite,
		Surface:     ui.ColorWhite,
		OnPrimary:   ui.ColorWhite, // White text on dark blue - excellent contrast
		OnSecondary: ui.ColorWhite, // White text on dark cyan - good contrast
		OnSurface:   ui.ColorBlack,
		Border:      ui.Color(8), // Medium gray for softer borders
		BorderFocus: ui.Color(4), // Dark Blue
		Text:        ui.ColorBlack,
		TextMuted:   ui.Color(240), // Medium gray - better visibility than 8
	}
}

// createDarkColorScheme returns a color scheme optimized for dark terminals
func createDarkColorScheme() *ColorScheme {
	return &ColorScheme{
		Primary:     ui.Color(6), // Cyan/Dark Cyan
		Secondary:   ui.Color(4), // Blue
		Accent:      ui.ColorMagenta,
		Success:     ui.Color(2),  // Green
		Warning:     ui.Color(11), // Bright Yellow for better contrast
		Error:       ui.Color(9),  // Bright Red
		Info:        ui.Color(14), // Bright Cyan
		Background:  ui.ColorBlack,
		Surface:     ui.ColorBlack,
		OnPrimary:   ui.ColorBlack, // Black text on cyan - good contrast
		OnSecondary: ui.ColorWhite, // White text on blue - excellent contrast
		OnSurface:   ui.ColorWhite,
		Border:      ui.Color(240), // Medium gray for softer borders
		BorderFocus: ui.Color(14),  // Bright Cyan
		Text:        ui.ColorWhite,
		TextMuted:   ui.Color(245), // Lighter gray - better visibility than 244
	}
}

// InitializeColors detects terminal mode and sets up the appropriate color scheme
func InitializeColors() {
	detectedMode = detectTerminalMode()

	switch detectedMode {
	case TerminalModeLight:
		currentColorScheme = createLightColorScheme()
	case TerminalModeDark:
		currentColorScheme = createDarkColorScheme()
	default:
		// Default to dark mode
		currentColorScheme = createDarkColorScheme()
	}
}

// GetColorScheme returns the current color scheme
func GetColorScheme() *ColorScheme {
	if currentColorScheme == nil {
		InitializeColors()
	}
	return currentColorScheme
}

// GetTerminalMode returns the detected terminal mode
func GetTerminalMode() TerminalMode {
	return detectedMode
}

// ANSI color codes for terminal output (adaptive to mode)
func GetANSIColors() (success, info, warning, error, reset string) {
	// For light mode terminals, use darker colors for better contrast
	// For dark mode terminals, use brighter colors
	if detectedMode == TerminalModeLight {
		success = "\033[32m" // Green
		info = "\033[34m"    // Blue
		warning = "\033[33m" // Yellow
		error = "\033[31m"   // Red
	} else {
		success = "\033[92m" // Bright Green
		info = "\033[96m"    // Bright Cyan
		warning = "\033[93m" // Bright Yellow
		error = "\033[91m"   // Bright Red
	}

	reset = "\033[0m"
	return
}

// Helper functions for consistent styling
func StyleBorder(focused bool) ui.Style {
	scheme := GetColorScheme()
	if focused {
		return ui.NewStyle(scheme.BorderFocus)
	}
	return ui.NewStyle(scheme.Border)
}

func StyleText() ui.Style {
	scheme := GetColorScheme()
	return ui.NewStyle(scheme.Text)
}

func StyleTextMuted() ui.Style {
	scheme := GetColorScheme()
	return ui.NewStyle(scheme.TextMuted)
}

func StylePrimary() ui.Style {
	scheme := GetColorScheme()
	return ui.NewStyle(scheme.OnPrimary, scheme.Primary)
}

func StyleSuccess() ui.Style {
	scheme := GetColorScheme()
	// For success backgrounds, ensure good contrast
	if detectedMode == TerminalModeLight {
		return ui.NewStyle(ui.ColorWhite, scheme.Success) // White text on dark green
	} else {
		return ui.NewStyle(ui.ColorBlack, scheme.Success) // Black text on green
	}
}

func StyleWarning() ui.Style {
	scheme := GetColorScheme()
	// For warning backgrounds, use contrasting text
	if detectedMode == TerminalModeLight {
		return ui.NewStyle(ui.ColorBlack, scheme.Warning) // Black text on dark yellow
	} else {
		return ui.NewStyle(ui.ColorBlack, scheme.Warning) // Black text on bright yellow
	}
}

func StyleError() ui.Style {
	scheme := GetColorScheme()
	// For error backgrounds, ensure good contrast
	if detectedMode == TerminalModeLight {
		return ui.NewStyle(ui.ColorWhite, scheme.Error) // White text on red
	} else {
		return ui.NewStyle(ui.ColorBlack, scheme.Error) // Black text on bright red
	}
}

func StyleInfo() ui.Style {
	scheme := GetColorScheme()
	// For info backgrounds, ensure good contrast
	if detectedMode == TerminalModeLight {
		return ui.NewStyle(ui.ColorWhite, scheme.Info) // White text on dark blue
	} else {
		return ui.NewStyle(ui.ColorBlack, scheme.Info) // Black text on bright cyan
	}
}
