package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Event beats and the journal (docs/systems/events.md). Fired rules
// queue their output on World.Pending; eventsSurface presents one
// beat per centered modal and drains each into the LOG on dismissal.
// journalSurface shows the derived journal ("journal" verb).

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
	b.WriteString(dimStyle.Render("░▒▓") + roomTitleStyle.Render(" · ") + dimStyle.Render("▓▒░"))
	b.WriteString("\n\n" + m.eng.World.Pending[0])
	b.WriteString("\n\n" + dimStyle.Render("enter to continue"))
	return b.String()
}

type journalSurface struct{}

func (journalSurface) Active(m *Model) bool { return m.journalOpen }

// Intercept claims the "journal" verb.
func (journalSurface) Intercept(m *Model, cmd engine.Command) bool {
	if cmd.Verb != "journal" {
		return false
	}
	m.journalOpen = true
	m.input.Blur()
	return true
}

// HandleKey: esc or enter closes.
func (journalSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc, tea.KeyEnter:
		m.journalOpen = false
		m.input.Focus()
	}
	return nil
}

// Overlay composites the journal over the main row.
func (journalSurface) Overlay(m *Model, bg string) string {
	return m.centerModal(bg, m.journalView(), 60)
}

// journalView renders the derived journal: every entry whose flag is
// true, recomputed from world state on every frame.
func (m Model) journalView() string {
	var b strings.Builder
	b.WriteString(roomTitleStyle.Render("JOURNAL"))
	entries := m.eng.World.JournalEntries()
	if len(entries) == 0 {
		b.WriteString("\n\n" + dimStyle.Render("nothing yet - the city keeps its secrets"))
	} else {
		for _, e := range entries {
			b.WriteString("\n\n- " + e)
		}
	}
	b.WriteString("\n\n" + dimStyle.Render("esc to close"))
	return b.String()
}
