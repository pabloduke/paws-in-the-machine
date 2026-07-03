// Package ui is the Bubble Tea terminal shell around the engine.
// Layout ("Option E"): city hub panel (left), live room view (center),
// reserved panel (right), a full-width LOG panel carrying the rolling
// command/output transcript, and a slim prompt at the bottom.
// Tab focuses the city panel for hub travel (docs/systems/hubs.md);
// Up/Down recall previous commands; PgUp/PgDn scroll the LOG.
package ui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

const (
	leftPanelWidth  = 24
	rightPanelWidth = 24
	logHeight       = 8 // LOG panel total height, border included
	promptHeight    = 1
)

var (
	bodyStyle   = lipgloss.NewStyle().Padding(0, 1)
	echoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
	panelFocusStyle = panelStyle.BorderForeground(lipgloss.Color("5"))
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	roomTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	hubSelStyle     = lipgloss.NewStyle().Reverse(true)
)

// Model is the Bubble Tea model for a session. The room view renders
// the current room fresh from world state; the LOG keeps the
// persistent transcript.
type Model struct {
	eng    *engine.Engine
	log    viewport.Model
	input  textinput.Model
	ready  bool
	width  int
	height int

	panelFocused bool
	selected     int
	dialogue     *dialogue.Session
	dlgSel       int // highlighted choice in the dialogue menu

	entries  []string // transcript lines shown in the LOG
	commands []string // executed commands, oldest first
	histPos  int      // 0 = live input; n = n commands back
	draft    string   // live input stashed while browsing history
}

// New builds a session around the engine, seeding the LOG with the
// intro text.
func New(eng *engine.Engine, intro string) Model {
	ti := textinput.New()
	ti.Prompt = promptStyle.Render("> ")
	ti.Placeholder = "what does Buddy do?"
	ti.Focus()

	return Model{
		eng:     eng,
		input:   ti,
		entries: []string{intro},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) mainRowHeight() int { return m.height - logHeight - promptHeight }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// LOG panel: 2 border rows + 1 title row + viewport.
		logW, logH := msg.Width-4, logHeight-3
		if !m.ready {
			m.log = viewport.New(logW, logH)
			// The viewport's default bindings grab letters (j, k, u,
			// d, b, f, space) that belong to the prompt. Scrollback is
			// PgUp/PgDn only; typing must never scroll.
			m.log.KeyMap = viewport.KeyMap{
				PageUp:   key.NewBinding(key.WithKeys("pgup")),
				PageDown: key.NewBinding(key.WithKeys("pgdown")),
			}
			m.ready = true
		} else {
			m.log.Width = logW
			m.log.Height = logH
		}
		m.refreshLog()
		return m, nil

	case tea.KeyMsg:
		if m.dialogue != nil {
			return m.updateDialogue(msg)
		}
		if m.panelFocused {
			return m.updatePanel(msg)
		}
		// Keystrokes go to exactly one component: PgUp/PgDn scroll the
		// LOG, Tab focuses the city panel, Up/Down browse command
		// history, everything else belongs to the prompt.
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
			m.log, cmd = m.log.Update(msg)
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
			m.entries = append(m.entries, echoStyle.Render("> "+line))
			m.executeLine(line)
			m.refreshLog()
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
	m.log, cmd = m.log.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// executeLine runs one prompt command, intercepting "talk <npc>" to
// enter dialogue mode before falling back to the engine.
func (m *Model) executeLine(line string) {
	cmd, ok := engine.Parse(line)
	if ok && cmd.Verb == "talk" {
		if cmd.Object == "" {
			m.entries = append(m.entries, "Talk to whom?")
			return
		}
		target := m.eng.World.InScope(cmd.Object)
		if target == nil {
			m.entries = append(m.entries, "You don't see any "+strconv.Quote(cmd.Object)+" here.")
			return
		}
		session, err := dialogue.Start(m.eng.World, target)
		if err != nil {
			m.entries = append(m.entries, err.Error())
			return
		}
		m.dialogue = session
		m.dlgSel = 0
		m.input.Blur()
		m.entries = append(m.entries, session.Speaker()+": "+session.Text())
		return
	}
	if out := m.eng.Execute(line); out != "" {
		m.entries = append(m.entries, out)
	}
}

// updateDialogue drives the conversation menu in the room panel:
// up/down move the highlight, enter picks, 1-9 pick directly, esc
// leaves. The LOG keeps the transcript; the panel is the live view.
func (m Model) updateDialogue(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	options := m.dialogue.Options()
	if m.dlgSel >= len(options) {
		m.dlgSel = 0
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.dialogue = nil
		m.input.Focus()
		m.entries = append(m.entries, dimStyle.Render("[conversation ended]"))
		m.refreshLog()
		return m, nil
	case tea.KeyUp:
		if len(options) > 0 {
			m.dlgSel = (m.dlgSel - 1 + len(options)) % len(options)
		}
		return m, nil
	case tea.KeyDown:
		if len(options) > 0 {
			m.dlgSel = (m.dlgSel + 1) % len(options)
		}
		return m, nil
	case tea.KeyEnter:
		return m.pickDialogue(m.dlgSel + 1)
	case tea.KeyRunes:
		if len(msg.Runes) != 1 || msg.Runes[0] < '1' || msg.Runes[0] > '9' {
			return m, nil
		}
		return m.pickDialogue(int(msg.Runes[0] - '0'))
	}
	return m, nil
}

// pickDialogue applies a 1-based choice and records the exchange in
// the LOG. Locked picks log the refusal and stay on the node.
func (m Model) pickDialogue(n int) (tea.Model, tea.Cmd) {
	options := m.dialogue.Options()
	if n < 1 || n > len(options) {
		return m, nil
	}
	locked := options[n-1].Locked
	m.entries = append(m.entries, echoStyle.Render("> "+options[n-1].Choice.Text))
	if out := m.dialogue.Pick(n); out != "" {
		m.entries = append(m.entries, out)
	}
	if m.dialogue.Done() {
		m.dialogue = nil
		m.input.Focus()
	} else if !locked {
		m.dlgSel = 0
		m.entries = append(m.entries, m.dialogue.Speaker()+": "+m.dialogue.Text())
	}
	m.refreshLog()
	return m, nil
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
		m.entries = append(m.entries, echoStyle.Render("> [travel] "+hub.Name))
		if out := hubs.Travel(m.eng.World, hub.ID); out != "" {
			m.entries = append(m.entries, out)
		}
		m.refreshLog()
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

// refreshLog re-renders the transcript, keeping it scrolled to the
// newest text.
func (m *Model) refreshLog() {
	if !m.ready {
		return
	}
	wrapped := lipgloss.NewStyle().Width(m.log.Width).
		Render(strings.Join(m.entries, "\n"))
	m.log.SetContent(wrapped)
	m.log.GotoBottom()
}

func (m Model) View() string {
	if !m.ready {
		return "booting the deck..."
	}
	mainRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.cityPanel(), m.roomPanel(), m.rightPanel())
	return mainRow + "\n" + m.logPanel() + "\n" + m.input.View()
}

// roomPanel is the live room view: title from the room name, body from
// world state — always current, never a transcript. While a
// conversation is active it becomes the dialogue menu instead.
func (m Model) roomPanel() string {
	width := m.width - leftPanelWidth - rightPanelWidth
	var content string
	if m.dialogue != nil {
		content = m.dialogueView()
	} else {
		name, body, _ := strings.Cut(engine.Look(m.eng.World), "\n\n")
		content = roomTitleStyle.Render(strings.ToUpper(name))
		if body != "" {
			content += "\n\n" + body
		}
	}
	return panelStyle.Width(width - 2).Height(m.mainRowHeight() - 2).
		Render(bodyStyle.Width(width - 4).Render(content))
}

// dialogueView renders the active conversation: the NPC line on top,
// then the choice menu. The highlighted row is selected with enter;
// locked stat-gated rows render dim with their tag and a ✗.
func (m Model) dialogueView() string {
	var b strings.Builder
	b.WriteString(roomTitleStyle.Render(strings.ToUpper(m.dialogue.Speaker())))
	b.WriteString("\n\n" + m.dialogue.Text() + "\n")
	options := m.dialogue.Options()
	sel := m.dlgSel
	if sel >= len(options) {
		sel = 0
	}
	for i, o := range options {
		row := strconv.Itoa(i+1) + ". "
		if o.Tag != "" {
			row += o.Tag + " "
		}
		row += o.Choice.Text
		if o.Locked {
			row += " ✗"
		}
		switch {
		case i == sel:
			row = hubSelStyle.Render(row)
		case o.Locked:
			row = dimStyle.Render(row)
		}
		b.WriteString("\n" + row)
	}
	b.WriteString("\n\n" + dimStyle.Render("up/down enter · 1-9 · esc to walk away"))
	return b.String()
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
			marker = "* "
		}
		row := marker + h.Name
		if m.panelFocused && i == m.selected {
			row = hubSelStyle.Render(row)
		}
		b.WriteString("\n" + row)
	}
	hint := "tab: focus"
	if m.panelFocused {
		hint = "up/down enter, esc"
	} else if m.dialogue != nil {
		hint = "dialogue active"
	}
	b.WriteString("\n\n" + dimStyle.Render(hint))

	style := panelStyle
	if m.panelFocused {
		style = panelFocusStyle
	}
	return style.Width(leftPanelWidth - 2).Height(m.mainRowHeight() - 2).Render(b.String())
}

// rightPanel: INVENTORY (Buddy always knows what he carries), YOU SEE
// (only obvious entities — scenery is discovered through prose), and
// EXITS. See docs/systems/visibility.md.
func (m Model) rightPanel() string {
	w := m.eng.World

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("INVENTORY"))
	if len(w.Player.Contents) == 0 {
		b.WriteString("\n" + dimStyle.Render("  nothing"))
	}
	for _, e := range w.Player.Contents {
		b.WriteString("\n  " + engine.DisplayName(w, e))
	}

	b.WriteString("\n\n" + panelTitleStyle.Render("YOU SEE"))
	for _, e := range w.Obvious() {
		b.WriteString("\n  " + engine.DisplayName(w, e))
	}
	if x, ok := engine.Part[engine.Exits](w.Room()); ok && len(x.Dirs) > 0 {
		b.WriteString("\n\n" + panelTitleStyle.Render("EXITS"))
		dirs := make([]string, 0, len(x.Dirs))
		for dir := range x.Dirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		for _, dir := range dirs {
			b.WriteString("\n  " + dir)
		}
	}
	return panelStyle.Width(rightPanelWidth - 2).Height(m.mainRowHeight() - 2).Render(b.String())
}

// logPanel renders the transcript viewport.
func (m Model) logPanel() string {
	content := panelTitleStyle.Render("LOG") + "\n" + m.log.View()
	return panelStyle.Width(m.width - 2).Render(content)
}
