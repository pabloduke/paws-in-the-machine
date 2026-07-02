// Package ui is the Bubble Tea terminal shell around the engine: a
// scrolling story viewport with a command prompt underneath, and a
// city side panel for hub travel (Tab to focus, arrows to select,
// Enter to travel — see docs/systems/hubs.md).
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

const panelWidth = 24

var (
	storyStyle  = lipgloss.NewStyle().Padding(0, 1)
	echoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
	panelFocusStyle = panelStyle.BorderForeground(lipgloss.Color("5"))
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	hubStyle        = lipgloss.NewStyle()
	hubSelStyle     = lipgloss.NewStyle().Reverse(true)
	panelHintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Model is the Bubble Tea model for a session.
type Model struct {
	eng      *engine.Engine
	viewport viewport.Model
	input    textinput.Model
	history  []string
	ready    bool
	width    int
	height   int

	panelFocused bool
	selected     int
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
		m.width, m.height = msg.Width, msg.Height
		inputHeight := 1
		vpWidth := msg.Width - panelWidth
		if !m.ready {
			m.viewport = viewport.New(vpWidth, msg.Height-inputHeight)
			// The viewport's default bindings grab letters (j, k, u,
			// d, b, f, space) that belong to the prompt. Scrollback is
			// PgUp/PgDn only; typing must never scroll.
			m.viewport.KeyMap = viewport.KeyMap{
				PageUp:   key.NewBinding(key.WithKeys("pgup")),
				PageDown: key.NewBinding(key.WithKeys("pgdown")),
			}
			m.ready = true
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = msg.Height - inputHeight
		}
		m.refresh()
		return m, nil

	case tea.KeyMsg:
		if m.panelFocused {
			return m.updatePanel(msg)
		}
		// Keystrokes go to exactly one component: PgUp/PgDn scroll the
		// transcript, Tab focuses the panel, everything else belongs
		// to the prompt.
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			if list := hubs.List(m.eng.World); len(list) > 0 {
				m.panelFocused = true
				m.selected = currentHubIndex(m.eng.World, list)
			}
			return m, nil
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

// updatePanel handles keys while the city panel has focus.
func (m Model) updatePanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := hubs.List(m.eng.World)
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyTab, tea.KeyEsc:
		m.panelFocused = false
	case tea.KeyUp:
		if m.selected > 0 {
			m.selected--
		}
	case tea.KeyDown:
		if m.selected < len(list)-1 {
			m.selected++
		}
	case tea.KeyEnter:
		if m.selected < len(list) {
			hub := list[m.selected]
			m.history = append(m.history, echoStyle.Render("> [travel] "+hub.Name))
			m.history = append(m.history, hubs.Travel(m.eng.World, hub.ID))
			m.refresh()
		}
		m.panelFocused = false
	}
	return m, nil
}

// currentHubIndex preselects the hub Buddy is in.
func currentHubIndex(w *engine.World, list []*engine.Entity) int {
	cur := hubs.Current(w)
	for i, h := range list {
		if h == cur {
			return i
		}
	}
	return 0
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
	left := m.viewport.View() + "\n" + m.input.View()
	return lipgloss.JoinHorizontal(lipgloss.Top, left, m.panel())
}

// panel renders the city hub list.
func (m Model) panel() string {
	list := hubs.List(m.eng.World)
	cur := hubs.Current(m.eng.World)

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("THE CITY"))
	for i, h := range list {
		marker := "  "
		if h == cur {
			marker = "• "
		}
		row := marker + h.Name
		if m.panelFocused && i == m.selected {
			row = hubSelStyle.Render(row)
		} else {
			row = hubStyle.Render(row)
		}
		b.WriteString("\n" + row)
	}
	hint := "tab: focus"
	if m.panelFocused {
		hint = "↑↓ enter, esc"
	}
	b.WriteString("\n\n" + panelHintStyle.Render(hint))

	style := panelStyle
	if m.panelFocused {
		style = panelFocusStyle
	}
	return style.Width(panelWidth - 2).Height(m.height - 2).Render(b.String())
}
