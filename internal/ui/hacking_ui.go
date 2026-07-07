package ui

import (
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

// HandleKey drives the terminal: enter executes a command, esc closes
// the terminal outright, PgUp/PgDn scroll, everything else types.
func (shellSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc:
		// Close the whole terminal from any depth — like shutting the
		// window; the shell narrates the disconnect first.
		m.saveShellEditor()
		m.shellEntries = append(m.shellEntries, m.shell.End())
		m.closeShell()
		return nil
	case tea.KeyTab:
		if m.editorPath != "" {
			m.saveShellEditor()
			m.readerFocus = false
			m.refreshShell()
			return nil
		}
		if m.hasReaderContent() {
			m.readerFocus = !m.readerFocus
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
			m.shellReader, cmd = m.shellReader.Update(msg)
			return cmd
		}
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
		m.shellInput.Reset()
		if line == "" {
			return nil
		}
		m.shellEntries = append(m.shellEntries,
			termEchoStyle.Render(m.shell.Prompt()+line))
		result := m.shell.ExecDetailed(line)
		if result.Edit != nil {
			m.openShellEditor(result.Edit)
			if result.Output != "" {
				m.shellEntries = append(m.shellEntries, termDimStyle.Render(result.Output))
			}
		} else if result.Document != nil {
			m.openShellDocument(result.Document)
			m.resizeShell()
			m.refreshShellReader()
			m.shellEntries = append(m.shellEntries,
				termDimStyle.Render("opened "+result.Document.Path+" in reader"))
		} else if result.Output != "" {
			m.shellEntries = append(m.shellEntries, result.Output)
		}
		if result.Done {
			m.closeShell()
			return nil
		}
		m.shellInput.Prompt = termPromptStyle.Render(m.shell.Prompt())
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

// Screen is the full-screen terminal layout: dominant terminal with an
// in-panel prompt, read-only status panel right.
func (shellSurface) Screen(m *Model) string {
	termW, statusW, panelH := m.shellDims()

	titleText := ansi.Truncate("CYBERDECK // "+strings.ToUpper(m.shell.HostName()), m.shellVP.Width, "")
	title := termTitleStyle.
		Width(m.shellVP.Width).
		MaxWidth(m.shellVP.Width).
		Render(titleText)
	prompt := lipgloss.NewStyle().
		Width(m.shellVP.Width).
		MaxWidth(m.shellVP.Width).
		Background(termDark).
		Render(m.shellInput.View())
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.shellVP.View(), prompt)
	termStyle := termPanelFocusStyle
	if m.readerFocus || m.editorPath != "" {
		termStyle = termPanelStyle
	}
	term := termStyle.Width(termW - 2).Height(panelH - 2).
		Render(content)
	statusStyle := termPanelStyle
	if m.readerFocus || m.editorPath != "" {
		statusStyle = termPanelFocusStyle
	}
	panel := m.questPanel()
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
	m.resizeShellEditor()
}

// shellDims returns terminal width, status-panel width, and panel
// height. The terminal is preserved first; the status panel shrinks.
func (m Model) shellDims() (termW, statusW, panelH int) {
	statusW = 26
	if m.hasReaderContent() || m.editorPath != "" {
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

// openShell logs into the deck and swaps the screen to the terminal.
func (m *Model) openShell(d hacking.Deck) {
	s, err := hacking.NewSession(m.eng.World, d.Net, d.Host)
	if err != nil {
		m.entries = append(m.entries, err.Error())
		return
	}
	m.shell = s
	m.deckCfg = d
	m.shellEntries = []string{termDimStyle.Render(
		"PAWS/OS — 'help' lists commands · 'exit' (or esc) leaves the terminal")}
	m.readerTitle = ""
	m.readerMD = ""
	m.readerText = ""
	m.readerFocus = false
	m.editorPath = ""

	ti := textinput.New()
	ti.Prompt = termPromptStyle.Render(s.Prompt())
	ti.TextStyle = termPromptStyle
	ti.Cursor.Style = termPromptStyle
	ti.Focus()
	m.shellInput = ti
	m.input.Blur()

	m.resizeShell()
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
	m.resizeShellEditor()
}

func (m *Model) resizeShellInput() {
	m.shellInput.Width = m.shellVP.Width - lipgloss.Width(m.shell.Prompt())
	if m.shellInput.Width < 1 {
		m.shellInput.Width = 1
	}
}

// refreshShell re-renders the terminal scrollback, pinned to newest.
func (m *Model) refreshShell() {
	wrapped := termOutputStyle.Width(m.shellVP.Width).
		Render(strings.Join(m.shellEntries, "\n"))
	if missing := m.shellVP.Height - lipgloss.Height(wrapped); missing > 0 {
		wrapped = strings.Repeat("\n", missing) + wrapped
	}
	m.shellVP.SetContent(wrapped)
	m.shellVP.GotoBottom()
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
	hintText := ansi.Truncate("tab: terminal · up/down: scroll", m.shellReader.Width, "")
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
	m.readerTitle = edit.Path
	m.readerMD = ""
	m.readerText = ""
	m.readerFocus = true
	m.editorPath = edit.Path
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
		m.shellEntries = append(m.shellEntries, termDimStyle.Render(out))
	}
	m.readerTitle = m.editorPath
	m.readerMD = ""
	m.readerText = "saved " + m.editorPath
	m.editorPath = ""
	m.readerFocus = false
	m.refreshShellReader()
}

func (m Model) editorPanel() string {
	titleText := ansi.Truncate("EDITOR // "+m.editorPath, m.shellReader.Width, "")
	title := termTitleStyle.
		Width(m.shellReader.Width).
		MaxWidth(m.shellReader.Width).
		Render(titleText)
	hintText := ansi.Truncate("tab: save · esc: save+close", m.shellReader.Width, "")
	hint := termDimStyle.
		Width(m.shellReader.Width).
		MaxWidth(m.shellReader.Width).
		Render(hintText)
	return lipgloss.JoinVertical(lipgloss.Left, title, m.shellEditor.View(), hint)
}

// closeShell tears the session down and restores the room UI. Rules
// don't evaluate while the terminal is open (docs/systems/events.md),
// so logging out is when the world reacts to flags set in-session.
func (m *Model) closeShell() {
	m.shell = nil
	m.readerFocus = false
	m.editorPath = ""
	m.input.Focus()
	m.entries = append(m.entries, dimStyle.Render("[left the terminal]"))
	m.eng.World.CheckEvents()
	m.refreshLog()
	m.maybeLevelUp()
}

// questPanel is the read-only orientation panel beside the terminal.
func (m Model) questPanel() string {
	w := m.eng.World
	var b strings.Builder
	b.WriteString(termPanelTitleStyle.Render("OBJECTIVE"))
	b.WriteString("\n" + m.deckCfg.CurrentObjective(w))

	b.WriteString("\n\n" + termPanelTitleStyle.Render("LOCATION"))
	b.WriteString("\n" + m.shell.HostName() + ":" + m.shell.Path())

	b.WriteString("\n\n" + termPanelTitleStyle.Render("DISCOVERIES"))
	if found := m.shell.Discoveries(); len(found) == 0 {
		b.WriteString("\n" + termDimStyle.Render("none yet"))
	} else {
		for _, name := range found {
			b.WriteString("\n" + name)
		}
	}

	b.WriteString("\n\n" + termPanelTitleStyle.Render("STATUS"))
	b.WriteString("\nlink: stable")
	b.WriteString("\nICE: " + termDimStyle.Render("none detected"))
	b.WriteString("\ntrace: " + termDimStyle.Render("cold"))
	return b.String()
}
