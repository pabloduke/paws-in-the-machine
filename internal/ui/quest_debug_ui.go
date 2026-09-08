package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type questDebugSurface struct{}

func (questDebugSurface) Active(m *Model) bool { return m.questConsole }
func (m *Model) toggleQuestConsole() {
	m.questConsole = !m.questConsole
	if m.questConsole {
		m.questInput = textinput.New()
		m.questInput.Prompt = "> "
		m.questInput.CharLimit = 200
		m.questInput.Focus()
		m.questViewport = viewport.New(max(20, m.width-2), max(4, m.height-5))
		m.questViewport.SetContent("Developer console. Esc or F12 closes.\n" + m.eng.World.QuestText(m.eng.World.DevMode))
		m.questConsoleText = "Developer console. Esc or F12 closes."
	}
}
func (questDebugSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.questConsole = false
		return nil
	case "ctrl+c":
		return tea.Quit
	case "pgup", "pgdown", "up", "down":
		var cmd tea.Cmd
		m.questViewport, cmd = m.questViewport.Update(msg)
		return cmd
	case "enter":
		line := strings.TrimSpace(m.questInput.Value())
		m.questInput.SetValue("")
		out, handled := m.eng.World.DevCommand(line)
		if !handled {
			out = "This console accepts developer commands only."
		}
		m.questConsoleText += "\n> " + line + "\n" + out
		if len(m.questConsoleText) > 32000 {
			m.questConsoleText = m.questConsoleText[len(m.questConsoleText)-32000:]
		}
		m.questViewport.SetContent(m.questConsoleText)
		m.questViewport.GotoBottom()
		return nil
	}
	var cmd tea.Cmd
	m.questInput, cmd = m.questInput.Update(msg)
	return cmd
}
func (questDebugSurface) Screen(m *Model) string {
	hint := "Developer console · Esc closes · PgUp/PgDown scroll"
	if m.eng.World.DevMode {
		hint += "\nquest-debug | quest-debug trace | quest-debug reset/complete <id>"
	}
	return hint + "\n" + m.questViewport.View() + "\n" + m.questInput.View()
}
func (questDebugSurface) Resize(m *Model) {
	m.questViewport.Width = max(20, m.width-2)
	m.questViewport.Height = max(4, m.height-5)
	m.questInput.Width = max(16, m.width-4)
}
