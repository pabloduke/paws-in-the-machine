package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
		m.shellEntries = append(m.shellEntries, m.shell.End())
		m.closeShell()
		return nil
	case tea.KeyPgUp, tea.KeyPgDown:
		var cmd tea.Cmd
		m.shellVP, cmd = m.shellVP.Update(msg)
		return cmd
	case tea.KeyEnter:
		line := strings.TrimSpace(m.shellInput.Value())
		m.shellInput.Reset()
		if line == "" {
			return nil
		}
		m.shellEntries = append(m.shellEntries,
			crtEchoStyle.Render(m.shell.Prompt()+line))
		out, done := m.shell.Exec(line)
		if out != "" {
			m.shellEntries = append(m.shellEntries, crtOutputStyle.Render(out))
		}
		if done {
			m.closeShell()
			return nil
		}
		m.shellInput.Prompt = crtPromptStyle.Render(m.shell.Prompt())
		m.refreshShell()
		return nil
	}
	var cmd tea.Cmd
	m.shellInput, cmd = m.shellInput.Update(msg)
	return cmd
}

// Screen is the full-screen terminal layout: dominant terminal with
// the prompt attached beneath it, read-only quest panel right.
func (shellSurface) Screen(m *Model) string {
	termW, questW, panelH := m.shellDims()

	title := crtTitleStyle.Render("CYBERDECK // " + strings.ToUpper(m.shell.HostName()))
	term := crtPanelFocusStyle.Width(termW - 2).Height(panelH - 2).
		Render(title + "\n" + m.shellVP.View())
	quest := crtPanelStyle.Width(questW - 2).Height(panelH - 2).
		Render(m.questPanel())
	row := lipgloss.JoinHorizontal(lipgloss.Top, term, quest)

	prompt := lipgloss.NewStyle().MaxWidth(termW).Background(crtDark).Render(m.shellInput.View())
	return row + "\n" + prompt
}

// Resize refits the terminal to the window.
func (shellSurface) Resize(m *Model) {
	m.resizeShell()
	m.refreshShell()
}

// shellDims returns terminal width, quest-panel width, and panel
// height. The terminal is preserved first; the quest panel shrinks.
func (m Model) shellDims() (termW, questW, panelH int) {
	questW = 26
	if m.width-questW < 46 {
		questW = m.width - 46
	}
	if questW < 12 {
		questW = 12
	}
	termW = m.width - questW
	if termW < 20 {
		termW = 20
	}
	panelH = m.height - promptHeight
	if panelH < 5 {
		panelH = 5
	}
	return termW, questW, panelH
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
	m.shellEntries = []string{crtDimStyle.Render(
		"PAWS/OS — 'help' lists commands · 'exit' (or esc) leaves the terminal")}

	ti := textinput.New()
	ti.Prompt = crtPromptStyle.Render(s.Prompt())
	ti.Focus()
	m.shellInput = ti
	m.input.Blur()

	m.resizeShell()
	m.refreshShell()
}

// resizeShell fits the scrollback viewport to the current window.
func (m *Model) resizeShell() {
	termW, _, panelH := m.shellDims()
	w, h := termW-4, panelH-3 // borders + title row
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
}

// refreshShell re-renders the terminal scrollback, pinned to newest.
func (m *Model) refreshShell() {
	wrapped := crtOutputStyle.Width(m.shellVP.Width).
		Render(strings.Join(m.shellEntries, "\n"))
	m.shellVP.SetContent(wrapped)
	m.shellVP.GotoBottom()
}

// closeShell tears the session down and restores the room UI. Rules
// don't evaluate while the terminal is open (docs/systems/events.md),
// so logging out is when the world reacts to flags set in-session.
func (m *Model) closeShell() {
	m.shell = nil
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
	b.WriteString(crtPanelTitleStyle.Render("OBJECTIVE"))
	b.WriteString("\n" + m.deckCfg.CurrentObjective(w))

	b.WriteString("\n\n" + crtPanelTitleStyle.Render("LOCATION"))
	b.WriteString("\n" + m.shell.HostName() + ":" + m.shell.Path())

	b.WriteString("\n\n" + crtPanelTitleStyle.Render("DISCOVERIES"))
	if found := m.shell.Discoveries(); len(found) == 0 {
		b.WriteString("\n" + crtDimStyle.Render("none yet"))
	} else {
		for _, name := range found {
			b.WriteString("\n" + name)
		}
	}

	b.WriteString("\n\n" + crtPanelTitleStyle.Render("STATUS"))
	b.WriteString("\nlink: stable")
	b.WriteString("\nICE: " + crtDimStyle.Render("none detected"))
	b.WriteString("\ntrace: " + crtDimStyle.Render("cold"))
	return b.String()
}
