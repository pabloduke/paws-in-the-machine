// Package ui is the Bubble Tea terminal shell around the engine: a
// scrolling story viewport with a command prompt underneath.
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

var (
	storyStyle  = lipgloss.NewStyle().Padding(0, 1)
	echoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
)

// Model is the Bubble Tea model for a session.
type Model struct {
	eng      *engine.Engine
	viewport viewport.Model
	input    textinput.Model
	history  []string
	ready    bool
}

// New builds a session around the engine, seeding the transcript with
// the intro text and the opening room description.
func New(eng *engine.Engine, intro string) Model {
	ti := textinput.New()
	ti.Prompt = promptStyle.Render("> ")
	ti.Placeholder = "what does Buddy do?"
	ti.Focus()

	return Model{
		eng:     eng,
		input:   ti,
		history: []string{intro, engine.Look(eng.World)},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		inputHeight := 1
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-inputHeight)
			// The viewport's default bindings grab letters (j, k, u,
			// d, b, f, space) that belong to the prompt. Scrollback is
			// PgUp/PgDn only; typing must never scroll.
			m.viewport.KeyMap = viewport.KeyMap{
				PageUp:   key.NewBinding(key.WithKeys("pgup")),
				PageDown: key.NewBinding(key.WithKeys("pgdown")),
			}
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - inputHeight
		}
		m.refresh()
		return m, nil

	case tea.KeyMsg:
		// Keystrokes go to exactly one component: PgUp/PgDn scroll the
		// transcript, everything else belongs to the prompt. Sending
		// keys to both is how typing "use deck" used to scroll the view.
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyPgUp, tea.KeyPgDown:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case tea.KeyEnter:
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if line == "" {
				return m, nil
			}
			m.history = append(m.history, echoStyle.Render("> "+line))
			if out := m.eng.Execute(line); out != "" {
				m.history = append(m.history, out)
			}
			m.refresh()
			if m.eng.World.Quitting() {
				return m, tea.Quit
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	// Non-key messages (mouse wheel, blink ticks, ...) go to both.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// refresh re-renders the transcript into the viewport, wrapped to the
// current width, and keeps it scrolled to the newest text.
func (m *Model) refresh() {
	if !m.ready {
		return
	}
	wrapped := storyStyle.Width(m.viewport.Width).Render(strings.Join(m.history, "\n\n"))
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	if !m.ready {
		return "booting the deck..."
	}
	return m.viewport.View() + "\n" + m.input.View()
}
