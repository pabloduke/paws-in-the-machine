package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// The hacking terminal (docs/systems/hacking.md). The screen swaps
// wholesale while a session is live; the room UI underneath is
// untouched and restored when the terminal closes.

type shellSurface struct{}

type shellEntryKind uint8

const (
	shellOutput shellEntryKind = iota
	shellEcho
	shellDim
)

func (shellSurface) Active(m *Model) bool { return m.shell != nil }

// Intercept claims "use <deck>" and logs in.
func (shellSurface) Intercept(m *Model, cmd engine.Command) bool {
	if cmd.Verb != "use" || cmd.Object == "" {
		return false
	}
	target := m.eng.World.InScope(cmd.Object)
	if target == nil {
		return false
	}
	d, isDeck := engine.Part[hacking.Deck](target)
	if !isDeck {
		return false
	}
	m.openShell(d)
	return true
}

// HandleKey drives the terminal: Enter executes, Tab completes,
// Shift+Tab moves panel focus, and Up/Down recall command history.
func (shellSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	if m.editorPath != "" && m.editorVim && msg.Type != tea.KeyCtrlC && msg.Type != tea.KeyShiftTab {
		// A vim buffer owns every key until :q or :wq — including esc,
		// which switches modes instead of closing the terminal.
		return m.vimEditorKey(msg)
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc:
		// Close the whole terminal from any depth — like shutting the
		// window; the shell narrates the disconnect first.
		m.saveShellEditor()
		m.appendShell(shellOutput, m.shell.End())
		m.closeShell()
		return nil
	case tea.KeyShiftTab:
		if m.editorPath != "" {
			m.saveShellEditor()
			m.readerFocus = false
			m.shellInput.Focus()
			m.refreshShell()
			return nil
		}
		m.readerFocus = !m.readerFocus
		if m.readerFocus {
			m.shellInput.Blur()
		} else {
			m.shellInput.Focus()
		}
		return nil
	case tea.KeyTab:
		if m.editorPath != "" {
			var cmd tea.Cmd
			m.shellEditor, cmd = m.shellEditor.Update(msg)
			return cmd
		}
		if m.readerFocus {
			return nil
		}
		before := m.shellInput.Value()
		completed, matches := m.shell.Complete(before)
		m.shellInput.SetValue(completed)
		m.shellInput.CursorEnd()
		if len(matches) > 1 && completed == before {
			m.appendShell(shellDim, strings.Join(matches, "  "))
			m.refreshShell()
		}
		return nil
	case tea.KeyPgUp, tea.KeyPgDown:
		var cmd tea.Cmd
		m.shellVP, cmd = m.shellVP.Update(msg)
		return cmd
	case tea.KeyUp, tea.KeyDown:
		if m.editorPath != "" {
			var cmd tea.Cmd
			m.shellEditor, cmd = m.shellEditor.Update(msg)
			return cmd
		}
		if m.readerFocus {
			var cmd tea.Cmd
			if m.msgOpen {
				m.msgVP, cmd = m.msgVP.Update(msg)
			} else {
				m.shellReader, cmd = m.shellReader.Update(msg)
			}
			return cmd
		}
		if !m.shell.AwaitingPassword() {
			if msg.Type == tea.KeyUp {
				m.recallShell(1)
			} else {
				m.recallShell(-1)
			}
		}
		return nil
	case tea.KeyEnter:
		if m.editorPath != "" {
			var cmd tea.Cmd
			m.shellEditor, cmd = m.shellEditor.Update(msg)
			return cmd
		}
		if m.readerFocus {
			return nil
		}
		line := strings.TrimSpace(m.shellInput.Value())
		secret := m.shell.AwaitingPassword()
		m.shellInput.Reset()
		m.shellHistPos, m.shellDraft = 0, ""
		if line == "" {
			return nil
		}
		if !secret {
			m.rememberShell(line)
		}
		echoed := line
		if secret {
			echoed = strings.Repeat("*", len([]rune(line)))
		}
		m.appendShell(shellEcho, m.shell.Prompt()+echoed)
		result := m.shell.ExecDetailed(line)
		if result.Edit != nil {
			m.openShellEditor(result.Edit)
			if result.Output != "" {
				m.appendShell(shellDim, result.Output)
			}
		} else if result.Document != nil {
			m.openShellDocument(result.Document)
			m.resizeShell()
			m.refreshShellReader()
			m.appendShell(shellDim, "opened "+result.Document.Path+" in reader")
		} else if result.Messenger {
			m.toggleMessenger()
		} else if result.Output != "" {
			m.appendShell(shellOutput, result.Output)
		}
		if result.Done {
			m.closeShell()
			return nil
		}
		m.syncMessenger()
		m.syncShellInput()
		m.resizeShellInput()
		m.refreshShell()
		return nil
	}
	if m.editorPath != "" {
		var cmd tea.Cmd
		m.shellEditor, cmd = m.shellEditor.Update(msg)
		return cmd
	}
	if m.readerFocus {
		return nil
	}
	var cmd tea.Cmd
	m.shellInput, cmd = m.shellInput.Update(msg)
	return cmd
}

func (m *Model) rememberShell(line string) {
	if len(m.shellHistory) == 0 || m.shellHistory[len(m.shellHistory)-1] != line {
		m.shellHistory = append(m.shellHistory, line)
	}
}

func (m *Model) recallShell(dir int) {
	pos := m.shellHistPos + dir
	if pos < 0 || pos > len(m.shellHistory) {
		return
	}
	if m.shellHistPos == 0 {
		m.shellDraft = m.shellInput.Value()
	}
	m.shellHistPos = pos
	if pos == 0 {
		m.shellInput.SetValue(m.shellDraft)
	} else {
		m.shellInput.SetValue(m.shellHistory[len(m.shellHistory)-pos])
	}
	m.shellInput.CursorEnd()
}

func (m *Model) syncShellInput() {
	theme := m.mainTerminalTheme()
	style := theme.promptStyle()
	m.shellInput.Prompt = style.Render(m.shell.Prompt())
	m.shellInput.TextStyle = style
	m.shellInput.Cursor.Style = style
	m.shellInput.EchoMode = textinput.EchoNormal
	if m.shell.AwaitingPassword() {
		m.shellInput.EchoMode = textinput.EchoPassword
		m.shellInput.EchoCharacter = '*'
	}
}

// Screen is the full-screen terminal layout: dominant terminal with an
// in-panel prompt, modal right panel (idle art / reader / editor /
// messenger — never game state; user ruling 2026-07-10).
func (shellSurface) Screen(m *Model) string {
	termW, statusW, panelH := m.shellDims()
	theme := m.mainTerminalTheme()

	titleText := ansi.Truncate("CYBERDECK // "+strings.ToUpper(m.shell.HostName()), m.shellVP.Width, "")
	title := theme.titleStyle().
		Width(m.shellVP.Width).
		MaxWidth(m.shellVP.Width).
		Render(titleText)
	prompt := lipgloss.NewStyle().
		Width(m.shellVP.Width).
		MaxWidth(m.shellVP.Width).
		Background(theme.dark).
		Render(m.shellInput.View())
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.shellVP.View(), prompt)
	termStyle := theme.panelStyle(!m.readerFocus && m.editorPath == "")
	term := termStyle.Width(termW - 2).Height(panelH - 2).
		Render(content)
	statusStyle := termPanelStyle
	if m.readerFocus || m.editorPath != "" {
		statusStyle = termPanelFocusStyle
	}
	panel := m.idlePanel()
	if m.msgOpen {
		panel = m.messengerPanel()
	}
	if m.hasReaderContent() || m.editorPath != "" {
		panel = m.readerPanel()
	}
	status := statusStyle.Width(statusW - 2).Height(panelH - 2).Render(panel)
	return lipgloss.JoinHorizontal(lipgloss.Top, term, status)
}

// Resize refits the terminal to the window.
func (shellSurface) Resize(m *Model) {
	m.resizeShell()
	m.refreshShell()
	m.refreshShellReader()
	m.refreshMessenger()
	m.resizeShellEditor()
}

// shellDims returns terminal width, status-panel width, and panel
// height. The terminal is preserved first; the status panel shrinks.
func (m Model) shellDims() (termW, statusW, panelH int) {
	statusW = 26
	if m.hasReaderContent() || m.editorPath != "" || m.msgOpen {
		statusW = m.width * 40 / 100
	}
	if m.width-statusW < 46 {
		statusW = m.width - 46
	}
	if statusW < 12 {
		statusW = 12
	}
	termW = m.width - statusW
	if termW < 20 {
		termW = 20
	}
	panelH = m.height
	if panelH < 5 {
		panelH = 5
	}
	return termW, statusW, panelH
}

// BootIntoDeck opens the terminal on the first deck in scope — the
// game starts at the terminal with marduk's brief waiting (user ruling
// 2026-07-09, docs/draft.md). Called by main after New, not inside it,
// so the overworld remains the default surface everywhere else (and in
// tests). No-op when no deck is reachable.
func (m *Model) BootIntoDeck() {
	decks := hacking.DecksInScope(m.eng.World)
	if len(decks) == 0 {
		return
	}
	if d, ok := engine.Part[hacking.Deck](decks[0]); ok {
		m.openShell(d)
	}
}

// openShell logs into the deck and swaps the screen to the terminal.
func (m *Model) openShell(d hacking.Deck) {
	s, err := hacking.NewSession(m.eng.World, d.Net, d.Host)
	if err != nil {
		m.entries = append(m.entries, err.Error())
		return
	}
	m.shell = s
	m.deckCfg = d
	m.shellHistory = nil
	m.shellHistPos = 0
	m.shellDraft = ""
	m.shellEntries = []string{"CantOS — 'help' lists commands · 'exit' (or esc) leaves the terminal"}
	m.shellEntryKinds = []shellEntryKind{shellDim}
	m.readerTitle = ""
	m.readerMD = ""
	m.readerText = ""
	m.readerFocus = false
	m.editorPath = ""
	m.msgOpen = false
	m.msgSeen = 0

	ti := textinput.New()
	theme := m.mainTerminalTheme()
	ti.Prompt = theme.promptStyle().Render(s.Prompt())
	ti.TextStyle = theme.promptStyle()
	ti.Cursor.Style = theme.promptStyle()
	ti.Focus()
	m.shellInput = ti
	m.syncShellInput()
	m.input.Blur()

	m.resizeShell()
	m.syncMessenger() // announce anything waiting on login
	m.refreshShell()
}

// resizeShell fits the scrollback viewport to the current window.
func (m *Model) resizeShell() {
	termW, statusW, panelH := m.shellDims()
	w, h := termW-4, panelH-5 // borders + title row + prompt row; viewport renders one trailing row
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if m.shellVP.Width == 0 {
		m.shellVP = viewport.New(w, h)
		m.shellVP.KeyMap = viewport.KeyMap{
			PageUp:   key.NewBinding(key.WithKeys("pgup")),
			PageDown: key.NewBinding(key.WithKeys("pgdown")),
		}
	} else {
		m.shellVP.Width = w
		m.shellVP.Height = h
	}
	m.resizeShellInput()

	rw, rh := statusW-4, panelH-5 // borders + reader title + hint row
	if rw < 1 {
		rw = 1
	}
	if rh < 1 {
		rh = 1
	}
	if m.shellReader.Width == 0 {
		m.shellReader = viewport.New(rw, rh)
	} else {
		m.shellReader.Width = rw
		m.shellReader.Height = rh
	}
	if m.msgVP.Width == 0 {
		m.msgVP = viewport.New(rw, rh)
	} else {
		m.msgVP.Width = rw
		m.msgVP.Height = rh
	}
	m.resizeShellEditor()
}

func (m *Model) resizeShellInput() {
	// textinput.Width covers the visible value, but View also draws the
	// cursor in one more cell. Reserve that cell so typing does not push
	// the prompt one column past the panel and soft-wrap it.
	m.shellInput.Width = m.shellVP.Width - lipgloss.Width(m.shell.Prompt()) - 1
	if m.shellInput.Width < 1 {
		m.shellInput.Width = 1
	}
}

// refreshShell re-renders the terminal scrollback, pinned to newest.
func (m *Model) refreshShell() {
	theme := m.mainTerminalTheme()
	entries := make([]string, 0, len(m.shellEntries))
	for i, entry := range m.shellEntries {
		kind := shellOutput
		if i < len(m.shellEntryKinds) {
			kind = m.shellEntryKinds[i]
		}
		style := theme.outputStyle()
		switch kind {
		case shellEcho:
			style = theme.echoStyle()
		case shellDim:
			style = theme.dimStyle()
		}
		entries = append(entries, style.Width(m.shellVP.Width).Render(entry))
	}
	wrapped := strings.Join(entries, "\n")
	if missing := m.shellVP.Height - lipgloss.Height(wrapped); missing > 0 {
		wrapped = strings.Repeat("\n", missing) + wrapped
	}
	m.shellVP.SetContent(wrapped)
	m.shellVP.GotoBottom()
}

func (m Model) mainTerminalTheme() terminalTheme {
	if m.shell != nil && m.shell.IsRemote() {
		return remoteTerminalTheme
	}
	return localTerminalTheme
}

func (m *Model) appendShell(kind shellEntryKind, text string) {
	m.shellEntries = append(m.shellEntries, text)
	m.shellEntryKinds = append(m.shellEntryKinds, kind)
}

func (m *Model) refreshShellReader() {
	if !m.hasReaderContent() || m.shellReader.Width == 0 {
		return
	}
	rendered := m.readerText
	if m.readerMD != "" {
		rendered = hacking.RenderMarkdown(m.readerMD, m.shellReader.Width)
	} else {
		rendered = termOutputStyle.Width(m.shellReader.Width).Render(m.readerText)
	}
	m.shellReader.SetContent(rendered)
}

func (m Model) readerPanel() string {
	if m.editorPath != "" {
		return m.editorPanel()
	}
	if !m.hasReaderContent() {
		return ""
	}
	titleText := ansi.Truncate("READER // "+m.readerTitle, m.shellReader.Width, "")
	title := termTitleStyle.
		Width(m.shellReader.Width).
		MaxWidth(m.shellReader.Width).
		Render(titleText)
	hintText := "shift+tab: focus · tab: complete"
	if m.readerFocus {
		hintText = "shift+tab: terminal · up/down: scroll"
	}
	hintText = ansi.Truncate(hintText, m.shellReader.Width, "")
	hint := termDimStyle.
		Width(m.shellReader.Width).
		MaxWidth(m.shellReader.Width).
		Render(hintText)
	return lipgloss.JoinVertical(lipgloss.Left, title, m.shellReader.View(), hint)
}

func (m Model) hasReaderContent() bool {
	return m.readerMD != "" || m.readerText != ""
}

func (m *Model) openShellDocument(doc *hacking.Document) {
	m.editorPath = ""
	m.msgOpen = false // the right panel is modal; the reader takes it
	m.readerTitle = doc.Path
	m.readerMD = ""
	m.readerText = ""
	switch doc.Kind {
	case hacking.DocumentMarkdown:
		m.readerMD = doc.Text
	case hacking.DocumentText:
		m.readerText = doc.Text
	}
}

func (m *Model) openShellEditor(edit *hacking.EditBuffer) {
	m.msgOpen = false // the right panel is modal; the editor takes it
	m.readerTitle = edit.Path
	m.readerMD = ""
	m.readerText = ""
	m.readerFocus = true
	m.editorPath = edit.Path
	m.editorVim = edit.Vim // vim buffers start in normal mode
	m.vimInsert = false
	m.vimCmd = ""
	m.vimMsg = ""
	if m.shellEditor.Width() == 0 {
		m.shellEditor = textarea.New()
		m.shellEditor.Prompt = ""
		m.shellEditor.ShowLineNumbers = false
		m.shellEditor.FocusedStyle.Base = termPromptStyle
		m.shellEditor.FocusedStyle.Text = termPromptStyle
		m.shellEditor.BlurredStyle.Base = termPromptStyle
		m.shellEditor.BlurredStyle.Text = termPromptStyle
		m.shellEditor.Cursor.Style = termPromptStyle
	}
	m.resizeShell()
	m.shellEditor.SetValue(edit.Text)
	m.shellEditor.Focus()
}

func (m *Model) resizeShellEditor() {
	if m.shellEditor.Width() == 0 || m.shellReader.Width == 0 {
		return
	}
	m.shellEditor.SetWidth(m.shellReader.Width)
	m.shellEditor.SetHeight(m.shellReader.Height)
}

func (m *Model) saveShellEditor() {
	if m.editorPath == "" {
		return
	}
	out := m.shell.SaveEdit(m.editorPath, m.shellEditor.Value())
	if out != "" {
		m.appendShell(shellDim, out)
	}
	m.closeShellEditor("saved " + m.editorPath)
}

// closeShellEditor puts the buffer away without touching the file —
// the tail shared by every way out of the editor (:q discards; save
// paths write first, then land here).
func (m *Model) closeShellEditor(note string) {
	m.readerTitle = m.editorPath
	m.readerMD = ""
	m.readerText = note
	m.editorPath = ""
	m.editorVim = false
	m.vimInsert = false
	m.vimCmd = ""
	m.vimMsg = ""
	m.readerFocus = false
	m.refreshShellReader()
}

// vimEditorKey drives a vim buffer: normal mode moves and takes
// :-commands, i enters insert, esc leaves it. The dialect is tiny on
// purpose — enough that vim hands work on autopilot, nothing a newb
// can get trapped by without the hint line showing the way out.
func (m *Model) vimEditorKey(msg tea.KeyMsg) tea.Cmd {
	m.vimMsg = ""
	if m.vimCmd != "" { // typing a :-command on the hint line
		switch msg.Type {
		case tea.KeyEsc:
			m.vimCmd = ""
		case tea.KeyEnter:
			m.vimExec()
		case tea.KeyBackspace:
			m.vimCmd = m.vimCmd[:len(m.vimCmd)-1]
		case tea.KeyRunes:
			m.vimCmd += string(msg.Runes)
		}
		return nil
	}
	if m.vimInsert {
		if msg.Type == tea.KeyEsc {
			m.vimInsert = false
			return nil
		}
		var cmd tea.Cmd
		m.shellEditor, cmd = m.shellEditor.Update(msg)
		return cmd
	}
	// normal mode
	switch msg.Type {
	case tea.KeyEsc:
		m.vimMsg = "type :q to quit" // the classic
		return nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "i":
			m.vimInsert = true
		case ":":
			m.vimCmd = ":"
		case "h", "j", "k", "l":
			arrows := map[string]tea.KeyType{
				"h": tea.KeyLeft, "j": tea.KeyDown,
				"k": tea.KeyUp, "l": tea.KeyRight,
			}
			var cmd tea.Cmd
			m.shellEditor, cmd = m.shellEditor.Update(
				tea.KeyMsg{Type: arrows[string(msg.Runes)]})
			return cmd
		}
		return nil
	case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight,
		tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		var cmd tea.Cmd
		m.shellEditor, cmd = m.shellEditor.Update(msg)
		return cmd
	}
	return nil
}

// vimExec runs a finished :-command: the small dialect (:w, :q, :wq/:x).
func (m *Model) vimExec() {
	cmd := m.vimCmd
	m.vimCmd = ""
	switch cmd {
	case ":w":
		out := m.shell.SaveEdit(m.editorPath, m.shellEditor.Value())
		m.vimMsg = out
	case ":q":
		m.closeShellEditor("closed " + m.editorPath + " without saving")
	case ":wq", ":x":
		m.saveShellEditor()
	default:
		m.vimMsg = "not an editor command: " + strings.TrimPrefix(cmd, ":")
	}
}

func (m Model) editorPanel() string {
	name, hint := "EDITOR", "shift+tab: save · esc: save+close"
	if m.editorVim {
		// The hint line doubles as vim's bottom row: pending :-command,
		// status flash, or the mode indicator.
		name = "VIM"
		switch {
		case m.vimCmd != "":
			hint = m.vimCmd
		case m.vimMsg != "":
			hint = m.vimMsg
		case m.vimInsert:
			hint = "-- INSERT -- · esc: normal mode"
		default:
			hint = "i: insert · :wq save+quit · :q quit"
		}
	}
	titleText := ansi.Truncate(name+" // "+m.editorPath, m.shellReader.Width, "")
	title := termTitleStyle.
		Width(m.shellReader.Width).
		MaxWidth(m.shellReader.Width).
		Render(titleText)
	hintText := ansi.Truncate(hint, m.shellReader.Width, "")
	hintRow := termDimStyle.
		Width(m.shellReader.Width).
		MaxWidth(m.shellReader.Width).
		Render(hintText)
	return lipgloss.JoinVertical(lipgloss.Left, title, m.shellEditor.View(), hintRow)
}

// closeShell tears the session down and restores the room UI. Rules
// don't evaluate while the terminal is open (docs/systems/events.md),
// so logging out is when the world reacts to flags set in-session.
func (m *Model) closeShell() {
	m.shell = nil
	m.readerFocus = false
	m.editorPath = ""
	m.msgOpen = false
	m.input.Focus()
	m.entries = append(m.entries, dimStyle.Render("[left the terminal]"))
	m.eng.World.CheckEvents()
	m.refreshLog()
	m.maybeLevelUp()
}

// idleArt is the right panel's screen-saver: pure chrome, no game
// state (user ruling 2026-07-10 — the panel stays blank unless a
// file, the editor, or the messenger claims it).
const idleArt = ` /\_/\
( -.- )  zzz
 |> <|`

// idlePanel is the right panel when nothing claims it: dim art,
// nothing else — no objectives, no status, no state.
func (m Model) idlePanel() string {
	_, statusW, panelH := m.shellDims()
	hint := "Tab: complete\n Shift+Tab: panel"
	if m.readerFocus {
		hint = "Shift+Tab: terminal"
	}
	art := termDimStyle.Render(idleArt + "\n\n CantOS\n\n " + hint)
	return lipgloss.Place(statusW-4, panelH-4, lipgloss.Center, lipgloss.Center, art)
}

// toggleMessenger is the `messenger` command landing in the UI: the
// right panel is modal, and this claims or releases it. Opening reads
// the whole thread.
func (m *Model) toggleMessenger() {
	mgr := m.deckCfg.Messenger
	if !mgr.Enabled() {
		m.appendShell(shellDim, "messenger: no service on this deck")
		return
	}
	if m.msgOpen {
		m.msgOpen = false
		m.msgSeen = 0
		m.resizeShell() // the panel narrows back to the quest column
		return
	}
	m.msgOpen = true
	m.readerTitle = "" // take the panel from the reader
	m.readerMD = ""
	m.readerText = ""
	m.readerFocus = false
	mgr.MarkRead(m.eng.World)
	m.resizeShell() // the panel widens like the reader
	m.refreshMessenger()
}

// syncMessenger runs after every command: with the panel open, new
// arrivals render (and read) immediately; closed, they get one dim
// scrollback notice — flags fire mid-session, so the Resistance can
// react to a hack while it happens, still without a single timer.
func (m *Model) syncMessenger() {
	mgr := m.deckCfg.Messenger
	if !mgr.Enabled() {
		return
	}
	if m.msgOpen {
		if mgr.Unread(m.eng.World) > 0 {
			mgr.MarkRead(m.eng.World)
			m.refreshMessenger()
		}
		return
	}
	if n := mgr.Unread(m.eng.World); n > m.msgSeen {
		m.appendShell(shellDim,
			fmt.Sprintf("[messenger] %d unread — 'messenger' opens it", n))
		m.msgSeen = n
	}
}

// refreshMessenger re-renders the thread, pinned to the newest message.
func (m *Model) refreshMessenger() {
	if !m.msgOpen || m.msgVP.Width == 0 {
		return
	}
	mgr := m.deckCfg.Messenger
	from := termPanelTitleStyle.Render(strings.ToUpper(mgr.Contact))
	var lines []string
	for _, msg := range mgr.Thread(m.eng.World) {
		lines = append(lines, from+"\n"+msg.Text)
	}
	content := termDimStyle.Render("no messages")
	if len(lines) > 0 {
		content = strings.Join(lines, "\n\n")
	}
	m.msgVP.SetContent(termOutputStyle.Width(m.msgVP.Width).Render(content))
	m.msgVP.GotoBottom()
}

// messengerPanel is the messenger's turn holding the modal right panel.
func (m Model) messengerPanel() string {
	titleText := ansi.Truncate("MESSENGER // "+
		strings.ToUpper(m.deckCfg.Messenger.Contact), m.msgVP.Width, "")
	title := termTitleStyle.
		Width(m.msgVP.Width).
		MaxWidth(m.msgVP.Width).
		Render(titleText)
	hintText := "shift+tab: focus · tab: complete"
	if m.readerFocus {
		hintText = "shift+tab: terminal · up/down: scroll"
	}
	hintText = ansi.Truncate(hintText, m.msgVP.Width, "")
	hint := termDimStyle.
		Width(m.msgVP.Width).
		MaxWidth(m.msgVP.Width).
		Render(hintText)
	return lipgloss.JoinVertical(lipgloss.Left, title, m.msgVP.View(), hint)
}
