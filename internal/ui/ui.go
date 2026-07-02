// Package ui is the Bubble Tea terminal shell around the engine.
// Layout: city hub panel (left), story viewport (center, largest),
// reserved panel (right), and a bordered command box at the bottom
// showing recent commands, with up/down recall at the prompt.
// Tab focuses the city panel for hub travel (docs/systems/hubs.md).
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

const (
	leftPanelWidth  = 24
	rightPanelWidth = 24
	historyLines    = 4 // recent commands shown above the prompt
	bottomPadding   = 1 // blank lines between the command box and screen edge
	commandBoxH     = historyLines + 1 + 2 + bottomPadding
)

var (
	storyStyle  = lipgloss.NewStyle().Padding(0, 1)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
	panelFocusStyle = panelStyle.BorderForeground(lipgloss.Color("5"))
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	hubSelStyle     = lipgloss.NewStyle().Reverse(true)
)

// Model is the Bubble Tea model for a session. The viewport is a live
// room view: the current room rendered fresh from state, with recent
// message output below it, cleared whenever Buddy changes rooms.
type Model struct {
	eng      *engine.Engine
	viewport viewport.Model
	input    textinput.Model
	intro    string // shown above the room view until Buddy first moves
	messages []string
	ready    bool
	width    int
	height   int

	panelFocused bool
	selected     int

	commands []string // executed commands, oldest first
	histPos  int      // 0 = live input; n = n commands back
	draft    string   // live input stashed while browsing history
}

// New builds a session around the engine, seeding the transcript with
// the intro text and the opening room description.
func New(eng *engine.Engine, intro string) Model {
	ti := textinput.New()
	ti.Prompt = promptStyle.Render("> ")
	ti.Placeholder = "what does Buddy do?"
	ti.Focus()

	return Model{
		eng:   eng,
		input: ti,
		intro: intro,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		vpWidth := msg.Width - leftPanelWidth - rightPanelWidth
		vpHeight := msg.Height - commandBoxH
		if !m.ready {
			m.viewport = viewport.New(vpWidth, vpHeight)
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
			m.viewport.Height = vpHeight
		}
		m.refresh()
		return m, nil

	case tea.KeyMsg:
		if m.panelFocused {
			return m.updatePanel(msg)
		}
		// Keystrokes go to exactly one component: PgUp/PgDn scroll the
		// transcript, Tab focuses the city panel, Up/Down browse
		// command history, everything else belongs to the prompt.
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
		case tea.KeyUp:
			m.recall(1)
			return m, nil
		case tea.KeyDown:
			m.recall(-1)
			return m, nil
		case tea.KeyEnter:
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			m.histPos, m.draft = 0, ""
			if line == "" {
				return m, nil
			}
			m.commands = append(m.commands, line)
			m.runCommand(line)
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

// recall moves through executed commands: dir=1 older, dir=-1 newer.
// Position 0 restores whatever was being typed.
func (m *Model) recall(dir int) {
	pos := m.histPos + dir
	if pos < 0 || pos > len(m.commands) {
		return
	}
	if m.histPos == 0 {
		m.draft = m.input.Value()
	}
	m.histPos = pos
	if pos == 0 {
		m.input.SetValue(m.draft)
	} else {
		m.input.SetValue(m.commands[len(m.commands)-pos])
	}
	m.input.CursorEnd()
}

// updatePanel handles keys while the city panel has focus.
func (m Model) updatePanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := hubs.List(m.eng.World)
	if len(list) == 0 {
		m.panelFocused = false
		return m, nil
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyTab, tea.KeyEsc:
		m.panelFocused = false
	case tea.KeyUp:
		m.selected = (m.selected - 1 + len(list)) % len(list)
	case tea.KeyDown:
		m.selected = (m.selected + 1) % len(list)
	case tea.KeyEnter:
		hub := list[m.selected]
		m.messages = nil
		m.intro = ""
		if out := hubs.Travel(m.eng.World, hub.ID); out != "" {
			m.messages = append(m.messages, out)
		}
		m.refresh()
		m.panelFocused = false
	}
	return m, nil
}

// runCommand executes one player command and updates the room view:
// messages reset on room change, and output that merely repeats the
// room description (an explicit "look") is not doubled.
func (m *Model) runCommand(line string) {
	before := m.eng.World.Room()
	out := m.eng.Execute(line)
	if m.eng.World.Room() != before {
		m.messages = nil
		m.intro = ""
	}
	if out != "" && out != engine.Look(m.eng.World) {
		m.messages = append(m.messages, out)
	}
	if len(m.messages) > 8 {
		m.messages = m.messages[len(m.messages)-8:]
	}
	m.refresh()
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

// refresh re-renders the room view: the current room from live state,
// then recent messages, scrolled to the newest text.
func (m *Model) refresh() {
	if !m.ready {
		return
	}
	var parts []string
	if m.intro != "" {
		parts = append(parts, m.intro)
	}
	parts = append(parts, engine.Look(m.eng.World))
	parts = append(parts, m.messages...)
	wrapped := storyStyle.Width(m.viewport.Width).Render(strings.Join(parts, "\n\n"))
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	if !m.ready {
		return "booting the deck..."
	}
	main := lipgloss.JoinHorizontal(lipgloss.Top,
		m.cityPanel(), m.viewport.View(), m.rightPanel())
	return main + "\n" + m.commandBox() + strings.Repeat("\n", bottomPadding)
}

// cityPanel renders the hub list (docs/systems/hubs.md).
func (m Model) cityPanel() string {
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
		}
		b.WriteString("\n" + row)
	}
	hint := "tab: focus"
	if m.panelFocused {
		hint = "↑↓ enter, esc"
	}
	b.WriteString("\n\n" + dimStyle.Render(hint))

	style := panelStyle
	if m.panelFocused {
		style = panelFocusStyle
	}
	return style.Width(leftPanelWidth - 2).Height(m.viewport.Height - 2).Render(b.String())
}

// rightPanel is reserved space for a future system (inventory, stats,
// suspicion — undecided). Kept empty on purpose.
func (m Model) rightPanel() string {
	return panelStyle.Width(rightPanelWidth - 2).Height(m.viewport.Height - 2).Render("")
}

// commandBox renders recent commands above the prompt, bordered,
// lifted off the bottom edge.
func (m Model) commandBox() string {
	rows := make([]string, 0, historyLines+1)
	start := len(m.commands) - historyLines
	if start < 0 {
		start = 0
	}
	for i := 0; i < historyLines-len(m.commands[start:]); i++ {
		rows = append(rows, "")
	}
	for _, c := range m.commands[start:] {
		rows = append(rows, dimStyle.Render("> "+c))
	}
	rows = append(rows, m.input.View())
	return panelStyle.Width(m.width - 2).Render(strings.Join(rows, "\n"))
}
