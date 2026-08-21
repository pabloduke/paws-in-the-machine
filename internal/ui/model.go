// Package ui is the Bubble Tea terminal shell around the engine.
// Layout ("Option E"): city hub panel (left), live room view (center),
// reserved panel (right), a full-width LOG panel carrying the rolling
// command/output transcript, and a slim prompt at the bottom.
// Shift+Tab focuses the city panel for hub travel (docs/systems/hubs.md);
// Up/Down recall previous commands; PgUp/PgDn scroll the LOG.
//
// Layout of this package (docs/BOUNDARIES.md): this file holds the
// core Model and the surface registry; each system's UI lives in its
// own <name>_ui.go file and plugs in as a surface. Adding a system's
// UI is one line in the surfaces list plus its own file.
package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
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
	eventsSurface{},
	modalSurface{},
	dialogueSurface{},
	pdaSurface{},
	journalSurface{},
	saveSurface{},
	citySurface{},
}

// Model is the Bubble Tea model for a session. The room view renders
// the current room fresh from world state; the LOG keeps the
// persistent transcript. Per-system fields are appended by the system
// that owns them; the surface registry above dispatches to them.
type Model struct {
	eng *engine.Engine
	// themeID selects immutable presentation data. It is UI preference,
	// never engine or world state.
	themeID string
	log     viewport.Model
	input   textinput.Model
	ready   bool
	width   int
	height  int

	panelFocused bool
	selected     int
	dialogue     *dialogue.Session
	dlgSel       int // highlighted choice in the dialogue menu

	modal  modalKind
	lvlSel int // highlighted stat in the level-up modal
	invOff int // scroll offset into the inventory modal

	journalOpen bool // the journal modal (docs/systems/events.md)

	// PDA menus (pda_ui.go): the carried read-only field device.
	pdaMode     pdaMode
	pdaCfg      hacking.PDA
	pdaSel      int
	pdaFiles    []hacking.TextFile
	pdaHosts    []string
	pdaDocTitle string
	pdaDocText  string

	// SavePath is the one save slot (docs/systems/saveload.md);
	// defaults to the user config dir, overridable (tests, flags).
	SavePath string

	// Hacking terminal (docs/systems/hacking.md). While shell is
	// non-nil the whole screen swaps to the terminal layout; the
	// normal panels and LOG are untouched underneath.
	shell           *hacking.Session
	deckCfg         hacking.Deck
	shellEntries    []string // raw terminal scrollback, separate from the LOG
	shellEntryKinds []shellEntryKind
	shellVP         viewport.Model
	shellInput      textinput.Model
	shellHistory    []string
	shellHistPos    int
	shellDraft      string
	shellReader     viewport.Model
	shellEditor     textarea.Model
	readerTitle     string
	readerMD        string
	readerText      string
	readerFocus     bool
	editorPath      string
	// vim-mode editor state (vi/vim/nvim): modal instead of autosave.
	editorVim bool   // the open buffer came from vim, not edit
	vimInsert bool   // insert mode (i); false = normal mode (esc)
	vimCmd    string // pending :-command line, ":" prefix included
	vimMsg    string // one-key status flash (bad :cmd, esc hint)
	// Messenger panel (hacking_ui.go): another modal state of the
	// right panel, toggled by the `messenger` command.
	msgOpen bool
	msgVP   viewport.Model
	msgSeen int // unread count already announced in the scrollback

	entries  []textElement // raw semantic transcript lines shown in the LOG
	commands []string      // executed commands, oldest first
	histPos  int           // 0 = live input; n = n commands back
	draft    string        // live input stashed while browsing history

	// phase is the render phase for the ambient weather (rain.go) —
	// presentation state only, advanced by the rain ticker. The game
	// itself never ticks (docs/systems/turns.md).
	phase int
	// rainLevel is the rain's opacity rung ('[' dims, ']' brightens;
	// 0 turns it off entirely). Player preference, not game state.
	rainLevel int
}

// rainFrame paces the ambient animation — one const to dial if it
// ever matters (battery, slow ssh links). Fast frames + staggered
// per-drop speeds (rain.go) is what makes the rain read as smooth.
const rainFrame = 100 * time.Millisecond

type rainTick time.Time

func rainTicker() tea.Cmd {
	return tea.Tick(rainFrame, func(t time.Time) tea.Msg { return rainTick(t) })
}

// New builds a session around the engine, seeding the LOG with the
// intro text.
func New(eng *engine.Engine, intro string, options ...Option) Model {
	opts := modelOptions{themeID: defaultThemeID}
	for _, option := range options {
		option(&opts)
	}
	theme, ok := themeByID(opts.themeID)
	if !ok {
		opts.themeID = defaultThemeID
		theme, _ = themeByID(defaultThemeID)
	}
	ti := textinput.New()
	ti.Prompt = theme.Overworld.prompt.Render("> ")
	ti.Placeholder = "what does Buddy do?"
	ti.Focus()

	return Model{
		eng:       eng,
		themeID:   opts.themeID,
		input:     ti,
		entries:   []textElement{{Role: textBody, Text: intro}},
		SavePath:  defaultSavePath(),
		rainLevel: defaultRainLevel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, rainTicker())
}

func (m Model) mainRowHeight() int { return m.height - logHeight - promptHeight }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rainTick:
		// Pure re-render fuel: advance the weather, reschedule, touch
		// nothing else.
		m.phase++
		return m, rainTicker()

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
		// '[' / ']' dial the rain's opacity anywhere in the overworld —
		// a view preference, so the prompt never sees the keys.
		switch msg.String() {
		case "[":
			m.setRain(-1)
			return m, nil
		case "]":
			m.setRain(1)
			return m, nil
		}
		// Keystrokes go to exactly one component: PgUp/PgDn scroll the
		// LOG, Shift+Tab focuses the city panel, Up/Down browse command
		// history, everything else belongs to the prompt.
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyShiftTab:
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
			m.appendLog(textEcho, "> "+line)
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
// parsed commands (interactive verbs like "meow" or "use deck");
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
		m.appendLog(textBody, out)
	}
	m.maybeLevelUp()
}

func (m Model) View() string {
	styles := m.presentation().Overworld
	if !m.ready {
		return styles.panelTitle.Render("PAWS IN THE MACHINE") + "\n" +
			styles.rain.Render("rain on the window.")
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

// setRain nudges the rain opacity one rung, clamped to [0, max];
// 0 makes it invisible.
func (m *Model) setRain(d int) {
	m.rainLevel += d
	if m.rainLevel < 0 {
		m.rainLevel = 0
	}
	if m.rainLevel > maxRainLevel {
		m.rainLevel = maxRainLevel
	}
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
		Render(renderLog(m.entries, m.presentation().Overworld))
	m.log.SetContent(wrapped)
	m.log.GotoBottom()
}

// centerModal composites content in a focused box centered over the
// main row, leaving the panels visible around it.
func (m Model) centerModal(bg, content string, modalW int) string {
	styles := m.presentation().Overworld
	if max := m.width - 8; modalW > max {
		modalW = max
	}
	if modalW < 20 {
		modalW = 20 // absurdly narrow terminals get a clipped box, not negative widths
	}
	box := styles.panelFocus.Width(modalW).
		Render(styles.body.Width(modalW - 4).Render(content))
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
