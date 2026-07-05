package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Event beats (docs/systems/events.md). Fired rules queue their output
// on World.Pending; eventsSurface presents one beat per centered modal
// and drains each into the LOG on dismissal.

type eventsSurface struct{}

func (eventsSurface) Active(m *Model) bool { return len(m.eng.World.Pending) > 0 }

// HandleKey dismisses the current beat; the next queued beat (if any)
// keeps the surface active.
func (eventsSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEnter, tea.KeyEsc:
		m.entries = append(m.entries, m.eng.World.Pending[0])
		m.eng.World.Pending = m.eng.World.Pending[1:]
		m.refreshLog()
		if len(m.eng.World.Pending) == 0 {
			m.maybeLevelUp()
		}
	}
	return nil
}

// Overlay composites the current beat over the main row.
func (eventsSurface) Overlay(m *Model, bg string) string {
	return m.centerModal(bg, m.eventView(), 50)
}

// eventView renders one story beat, scene-break style.
func (m Model) eventView() string {
	var b strings.Builder
	b.WriteString(roomTitleStyle.Render("* * *"))
	b.WriteString("\n\n" + m.eng.World.Pending[0])
	b.WriteString("\n\n" + dimStyle.Render("enter to continue"))
	return b.String()
}
