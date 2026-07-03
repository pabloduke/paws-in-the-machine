// Package ui is the Bubble Tea terminal shell around the engine.
// Layout ("Option E"): city hub panel (left), live room view (center),
// reserved panel (right), a full-width LOG panel carrying the rolling
// command/output transcript, and a slim prompt at the bottom.
// Tab focuses the city panel for hub travel (docs/systems/hubs.md);
// Up/Down recall previous commands; PgUp/PgDn scroll the LOG.
package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

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

// modalKind selects which centered modal owns the keyboard. Dialogue
// keeps its own Session state; these are the lighter overlays.
type modalKind int

const (
	modalNone modalKind = iota
	modalLevelUp   // must-spend stat picker; opens itself on level-up
	modalInventory // scrollable list of what Buddy carries
	modalStats     // read-only character sheet
)

// trainOrder fixes the row order of the level-up picker.
var trainOrder = []string{"stealth", "agility", "charm"}

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

	modal  modalKind
	lvlSel int // highlighted stat in the level-up modal
	invOff int // scroll offset into the inventory modal

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
		if m.modal != modalNone {
			return m.updateModal(msg)
		}
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

// executeLine runs one prompt command, intercepting the interactive
// verbs — "talk" enters dialogue mode, "inventory" and "stats" open
// modals — before falling back to the engine. Rewrites apply first so
// idioms that expand to intercepted verbs still hit the intercepts.
func (m *Model) executeLine(line string) {
	line = m.eng.World.Rewrite(line)
	cmd, ok := engine.Parse(line)
	if ok && cmd.Verb == "inventory" {
		m.modal = modalInventory
		m.invOff = 0
		m.input.Blur()
		return
	}
	if ok && cmd.Verb == "stats" {
		m.modal = modalStats
		m.input.Blur()
		return
	}
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
	m.maybeLevelUp()
}

// maybeLevelUp opens the must-spend stat picker whenever Buddy has
// points. Called after every path that can award XP; while a
// conversation is open it waits, and the dialogue close paths re-check.
func (m *Model) maybeLevelUp() {
	if m.modal == modalNone && m.dialogue == nil && m.eng.World.StatPoints > 0 {
		m.modal = modalLevelUp
		m.lvlSel = 0
		m.input.Blur()
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
		m.maybeLevelUp()
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
		// Numbers move the highlight; only enter commits.
		if len(msg.Runes) != 1 || msg.Runes[0] < '1' || msg.Runes[0] > '9' {
			return m, nil
		}
		if n := int(msg.Runes[0] - '0'); n <= len(options) {
			m.dlgSel = n - 1
		}
		return m, nil
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
		m.maybeLevelUp()
	} else if !locked {
		m.dlgSel = 0
		m.entries = append(m.entries, m.dialogue.Speaker()+": "+m.dialogue.Text())
	}
	m.refreshLog()
	return m, nil
}

// updateModal routes keys to whichever centered modal is open.
func (m Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch m.modal {
	case modalLevelUp:
		return m.updateLevelUp(msg)
	case modalInventory:
		return m.updateInventory(msg)
	case modalStats:
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
			m.closeModal()
		}
	}
	return m, nil
}

// closeModal dismisses the open modal and returns focus to the prompt.
func (m *Model) closeModal() {
	m.modal = modalNone
	m.input.Focus()
}

// updateLevelUp drives the must-spend stat picker: up/down highlight,
// enter trains. There is no escape — points are spent on the spot, so
// they never linger and the modal never needs a reopen path.
func (m Model) updateLevelUp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		m.lvlSel = (m.lvlSel - 1 + len(trainOrder)) % len(trainOrder)
	case tea.KeyDown:
		m.lvlSel = (m.lvlSel + 1) % len(trainOrder)
	case tea.KeyEnter:
		if out := engine.Train(m.eng.World, trainOrder[m.lvlSel]); out != "" {
			m.entries = append(m.entries, out)
		}
		m.refreshLog()
		if m.eng.World.StatPoints == 0 {
			m.closeModal()
		}
	}
	return m, nil
}

// updateInventory scrolls the carried-items modal; esc or enter closes.
func (m Model) updateInventory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.modalListHeight()
	max := len(m.inventoryLines()) - rows
	if max < 0 {
		max = 0
	}
	switch msg.Type {
	case tea.KeyEsc, tea.KeyEnter:
		m.closeModal()
	case tea.KeyUp:
		m.invOff--
	case tea.KeyDown:
		m.invOff++
	case tea.KeyPgUp:
		m.invOff -= rows
	case tea.KeyPgDown:
		m.invOff += rows
	}
	if m.invOff > max {
		m.invOff = max
	}
	if m.invOff < 0 {
		m.invOff = 0
	}
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
	if m.dialogue != nil {
		mainRow = m.centerModal(mainRow, m.dialogueView(), 60)
	}
	switch m.modal {
	case modalLevelUp:
		mainRow = m.centerModal(mainRow, m.levelUpView(), 40)
	case modalInventory:
		mainRow = m.centerModal(mainRow, m.inventoryView(), 40)
	case modalStats:
		mainRow = m.centerModal(mainRow, m.statsView(), 40)
	}
	return mainRow + "\n" + m.logPanel() + "\n" + m.input.View()
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

// roomPanel is the live room view: title from the room name, body from
// world state — always current, never a transcript.
func (m Model) roomPanel() string {
	width := m.width - leftPanelWidth - rightPanelWidth
	name, body, _ := strings.Cut(engine.Look(m.eng.World), "\n\n")
	content := roomTitleStyle.Render(strings.ToUpper(name))
	if body != "" {
		content += "\n\n" + body
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
	b.WriteString("\n\n" + dimStyle.Render("up/down or 1-9 to highlight · enter to say it · esc to walk away"))
	return b.String()
}

// levelUpView is the must-spend stat picker: each row shows the stat
// and what training it would make it.
func (m Model) levelUpView() string {
	w := m.eng.World
	var b strings.Builder
	b.WriteString(roomTitleStyle.Render("LEVEL UP!"))
	plural := "point"
	if w.StatPoints != 1 {
		plural = "points"
	}
	b.WriteString(fmt.Sprintf("\n\nLevel %d. %d stat %s to spend.\n", w.Level, w.StatPoints, plural))
	for i, name := range trainOrder {
		v := *w.Stats.ByName(name)
		row := fmt.Sprintf("%-8s %d → %d", engine.Capitalize(name), v, v+1)
		if i == m.lvlSel {
			row = hubSelStyle.Render(row)
		}
		b.WriteString("\n" + row)
	}
	b.WriteString("\n\n" + dimStyle.Render("up/down to highlight · enter to train"))
	return b.String()
}

// modalListHeight is how many list rows a scrollable modal shows.
func (m Model) modalListHeight() int {
	h := m.mainRowHeight() - 8 // borders, title, hint, scroll markers
	if h < 3 {
		h = 3
	}
	return h
}

// inventoryLines is the inventory modal's full list, pre-scroll.
func (m Model) inventoryLines() []string {
	w := m.eng.World
	if len(w.Player.Contents) == 0 {
		return []string{dimStyle.Render("nothing — traveling light")}
	}
	var lines []string
	for _, e := range w.Player.Contents {
		lines = append(lines, engine.DisplayName(w, e))
	}
	return lines
}

// inventoryView renders the scrollable carried-items modal.
func (m Model) inventoryView() string {
	lines := m.inventoryLines()
	rows := m.modalListHeight()

	var b strings.Builder
	b.WriteString(roomTitleStyle.Render("INVENTORY") + "\n")
	if m.invOff > 0 {
		b.WriteString("\n" + dimStyle.Render("▲ more"))
	}
	end := m.invOff + rows
	if end > len(lines) {
		end = len(lines)
	}
	for _, line := range lines[m.invOff:end] {
		b.WriteString("\n" + line)
	}
	if end < len(lines) {
		b.WriteString("\n" + dimStyle.Render("▼ more"))
	}
	b.WriteString("\n\n" + dimStyle.Render("up/down to scroll · esc to close"))
	return b.String()
}

// statsView is the read-only character sheet modal.
func (m Model) statsView() string {
	var b strings.Builder
	b.WriteString(roomTitleStyle.Render("BUDDY") + "\n\n")
	b.WriteString(engine.StatSheet(m.eng.World))
	b.WriteString("\n\n" + dimStyle.Render("esc to close"))
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

// rightPanel: BUDDY (level and XP — the full sheet and inventory live
// in their modals), YOU SEE (only obvious entities — scenery is
// discovered through prose), and EXITS. See docs/systems/visibility.md.
func (m Model) rightPanel() string {
	w := m.eng.World

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("BUDDY"))
	b.WriteString(fmt.Sprintf("\n  Lv %d · XP %d/%d", w.Level, w.XP, w.NextLevelCost()))
	if w.StatPoints > 0 {
		b.WriteString("\n  " + dimStyle.Render(fmt.Sprintf("● %d to spend", w.StatPoints)))
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
