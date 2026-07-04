// Package ui is the Bubble Tea terminal shell around the engine.
// Layout ("Option E"): city hub panel (left), live room view (center),
// reserved panel (right), a full-width LOG panel carrying the rolling
// command/output transcript, and a slim prompt at the bottom.
// Tab focuses the city panel for hub travel (docs/systems/hubs.md);
// Up/Down recall previous commands; PgUp/PgDn scroll the LOG.
//
// Layout of this package (docs/BOUNDARIES.md): this file holds the
// core Model and the surface registry; each system's UI lives in its
// own <name>_ui.go file and plugs in as a surface. Adding a system's
// UI is one line in the surfaces list plus its own file.
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// surface is a UI mode a system contributes. While Active it owns the
// keyboard. Optional capabilities (fullscreenSurface, overlaySurface,
// resizableSurface, interceptorSurface) extend it.
type surface interface {
	// Active reports whether this surface currently owns input.
	Active(m *Model) bool
	// HandleKey handles a key msg while active.
	HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd
}

// fullscreenSurface replaces the whole screen while active.
type fullscreenSurface interface {
	Screen(m *Model) string
}

// overlaySurface composites over the main row while active.
type overlaySurface interface {
	Overlay(m *Model, bg string) string
}

// resizableSurface reacts to window size changes while active.
type resizableSurface interface {
	Resize(m *Model)
}

// interceptorSurface claims prompt commands (parsed verbs) before the
// engine sees them. Returns true if the command was handled.
type interceptorSurface interface {
	Intercept(m *Model, cmd engine.Command) bool
}

// surfaces is the registry, in input-priority order: the first active
// surface gets the keyboard. Overlays composite in reverse order, so
// earlier surfaces render on top. Registering a new system's UI is
// one line here (docs/BOUNDARIES.md).
var surfaces = []surface{
	shellSurface{},
	modalSurface{},
	dialogueSurface{},
	citySurface{},
}

// Model is the Bubble Tea model for a session. The room view renders
// the current room fresh from world state; the LOG keeps the
// persistent transcript. Per-system fields are appended by the system
// that owns them; the surface registry above dispatches to them.
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

	modal  modalKind
	lvlSel int // highlighted stat in the level-up modal
	invOff int // scroll offset into the inventory modal

	// Hacking terminal (docs/systems/hacking.md). While shell is
	// non-nil the whole screen swaps to the terminal layout; the
	// normal panels and LOG are untouched underneath.
	shell        *hacking.Session
	deckCfg      hacking.Deck
	shellEntries []string // terminal scrollback, separate from the LOG
	shellVP      viewport.Model
	shellInput   textinput.Model

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
		for _, s := range surfaces {
			if r, ok := s.(resizableSurface); ok && s.Active(&m) {
				r.Resize(&m)
			}
		}
		return m, nil

	case tea.KeyMsg:
		for _, s := range surfaces {
			if s.Active(&m) {
				return m, s.HandleKey(&m, msg)
			}
		}
		// Keystrokes go to exactly one component: PgUp/PgDn scroll the
		// LOG, Tab focuses the city panel, Up/Down browse command
		// history, everything else belongs to the prompt.
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			m.focusCityPanel()
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

// executeLine runs one prompt command. Surfaces get first claim on
// parsed commands (interactive verbs like "talk" or "use deck");
// whatever nobody intercepts falls through to the engine. Rewrites
// apply first so idioms that expand to intercepted verbs still hit
// the intercepts.
func (m *Model) executeLine(line string) {
	line = m.eng.World.Rewrite(line)
	if cmd, ok := engine.Parse(line); ok {
		for _, s := range surfaces {
			if ic, can := s.(interceptorSurface); can && ic.Intercept(m, cmd) {
				return
			}
		}
	}
	if out := m.eng.Execute(line); out != "" {
		m.entries = append(m.entries, out)
	}
	m.maybeLevelUp()
}

func (m Model) View() string {
	if !m.ready {
		return "booting the deck..."
	}
	for _, s := range surfaces {
		if fs, ok := s.(fullscreenSurface); ok && s.Active(&m) {
			return fs.Screen(&m)
		}
	}
	mainRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.cityPanel(), m.roomPanel(), m.rightPanel())
	for i := len(surfaces) - 1; i >= 0; i-- {
		if ov, ok := surfaces[i].(overlaySurface); ok && surfaces[i].Active(&m) {
			mainRow = ov.Overlay(&m, mainRow)
		}
	}
	return mainRow + "\n" + m.logPanel() + "\n" + m.input.View()
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

// centerModal composites content in a focused box centered over the
// main row, leaving the panels visible around it.
func (m Model) centerModal(bg, content string, modalW int) string {
	if max := m.width - 8; modalW > max {
		modalW = max
	}
	if modalW < 20 {
		modalW = 20 // absurdly narrow terminals get a clipped box, not negative widths
	}
	box := panelFocusStyle.Width(modalW).
		Render(bodyStyle.Width(modalW - 4).Render(content))
	x := (m.width - lipgloss.Width(box)) / 2
	y := (m.mainRowHeight() - lipgloss.Height(box)) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return overlay(bg, box, x, y)
}

// overlay splices fg over bg at column x, row y, ANSI-aware.
func overlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	for i, fl := range fgLines {
		j := y + i
		if j < 0 || j >= len(bgLines) {
			continue
		}
		bl := bgLines[j]
		left := ansi.Truncate(bl, x, "")
		if pad := x - ansi.StringWidth(left); pad > 0 {
			left += strings.Repeat(" ", pad)
		}
		right := ansi.TruncateLeft(bl, x+ansi.StringWidth(fl), "")
		bgLines[j] = left + fl + right
	}
	return strings.Join(bgLines, "\n")
}
