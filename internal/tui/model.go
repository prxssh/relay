package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/prxssh/relay/internal/relay"
)

const logo = `
______ _____ _       _____   __
| ___ \  ___| |     / _ \ \ / /
| |_/ / |__ | |    / /_\ \ V / 
|    /|  __|| |    |  _  |\ /  
| |\ \| |___| |____| | | || |  
\_| \_\____/\_____/\_| |_/\_/  
`

func Start() error {
	client, err := relay.NewClient()
	if err != nil {
		return err
	}

	p := tea.NewProgram(newModel(client), tea.WithAltScreen())
	_, err = p.Run()

	return err
}

/////////////// Private ///////////////

type model struct {
	client        *relay.Client
	screens       map[viewState]screen
	activeState   viewState
	theme         theme
	width, height int
}

func newModel(client *relay.Client) model {
	theme := newTheme()

	screens := map[viewState]screen{
		initialState: newInitialView(theme),
	}

	return model{
		client:      client,
		theme:       theme,
		screens:     screens,
		activeState: initialState,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var currScreen screen

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for i := range m.screens {
			m.screens[viewState(i)].SetSize(m.width, m.height)
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "a":
			return m, tea.Quit
		}
	}

	currScreen, cmd = m.screens[m.activeState].Update(msg)
	m.screens[m.activeState] = currScreen

	return m, cmd
}

func (m model) View() string {
	screenContent := m.screens[m.activeState].View()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, screenContent)
}
