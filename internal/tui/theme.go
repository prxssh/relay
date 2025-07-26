package tui

import "github.com/charmbracelet/lipgloss"

type theme struct {
	Bg, Fg       lipgloss.Color
	Red, Green   lipgloss.Color
	Yellow, Blue lipgloss.Color
	Aqua, Orange lipgloss.Color
	Gray         lipgloss.Color
}

func newTheme() theme {
	// Gruvbox Dark, Medium-Contrast Color Palette
	return theme{
		Bg:     lipgloss.Color("#282828"),
		Fg:     lipgloss.Color("#ebdbb2"),
		Red:    lipgloss.Color("#cc241d"),
		Green:  lipgloss.Color("#98971a"),
		Yellow: lipgloss.Color("#d79921"),
		Blue:   lipgloss.Color("#458588"),
		Aqua:   lipgloss.Color("#689d6a"),
		Orange: lipgloss.Color("#d65d0e"),
		Gray:   lipgloss.Color("#928374"),
	}
}
