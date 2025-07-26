package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type initialViewModel struct {
	theme         theme
	width, height int
}

func newInitialView(theme theme) screen {
	return &initialViewModel{theme: theme}
}

func (m *initialViewModel) SetSize(width, height int) {
	m.width, m.height = width, height
}

func (m *initialViewModel) Update(msg tea.Msg) (screen, tea.Cmd) {
	return m, nil
}

func (m *initialViewModel) View() string {
	if m.width == 0 {
		return ""
	}

	logoStyle := lipgloss.NewStyle().Foreground(m.theme.Blue)
	helpStyle := lipgloss.NewStyle().Foreground(m.theme.Gray)

	styledLogo := logoStyle.Render(logo)
	statusText := helpStyle.Render("No torrents added.")
	helpText := helpStyle.Render(
		"Press 'a' to add a torrent or 'q' to quite.",
	)

	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center, styledLogo, statusText, helpText))
}
