package tui

import tea "github.com/charmbracelet/bubbletea"

// Screen is a contract that all our views must fulfill.
type screen interface {
	SetSize(width, height int)
	Update(msg tea.Msg) (screen, tea.Cmd)
	View() string
}

type viewState int

const (
	initialState = iota
)
