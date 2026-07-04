package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// The centered modals: the must-spend level-up picker, the inventory
// list, and the read-only character sheet. Dialogue keeps its own
// Session state; these are the lighter overlays.

// modalKind selects which centered modal owns the keyboard.
type modalKind int

const (
	modalNone      modalKind = iota
	modalLevelUp             // must-spend stat picker; opens itself on level-up
	modalInventory           // scrollable list of what Buddy carries
	modalStats               // read-only character sheet
)

// trainOrder fixes the row order of the level-up picker.
var trainOrder = []string{"stealth", "agility", "charm"}

type modalSurface struct{}

func (modalSurface) Active(m *Model) bool { return m.modal != modalNone }

// Intercept claims the "inventory" and "stats" verbs.
func (modalSurface) Intercept(m *Model, cmd engine.Command) bool {
	switch cmd.Verb {
	case "inventory":
		m.modal = modalInventory
		m.invOff = 0
		m.input.Blur()
		return true
	case "stats":
		m.modal = modalStats
		m.input.Blur()
		return true
	}
	return false
}

// HandleKey routes keys to whichever centered modal is open.
func (modalSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	if msg.Type == tea.KeyCtrlC {
		return tea.Quit
	}
	switch m.modal {
	case modalLevelUp:
		m.updateLevelUp(msg)
	case modalInventory:
		m.updateInventory(msg)
	case modalStats:
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
			m.closeModal()
		}
	}
	return nil
}

// Overlay composites the open modal over the main row.
func (modalSurface) Overlay(m *Model, bg string) string {
	switch m.modal {
	case modalLevelUp:
		return m.centerModal(bg, m.levelUpView(), 40)
	case modalInventory:
		return m.centerModal(bg, m.inventoryView(), 40)
	case modalStats:
		return m.centerModal(bg, m.statsView(), 40)
	}
	return bg
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

// closeModal dismisses the open modal and returns focus to the prompt.
func (m *Model) closeModal() {
	m.modal = modalNone
	m.input.Focus()
}

// updateLevelUp drives the must-spend stat picker: up/down highlight,
// enter trains. There is no escape — points are spent on the spot, so
// they never linger and the modal never needs a reopen path.
func (m *Model) updateLevelUp(msg tea.KeyMsg) {
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
}

// updateInventory scrolls the carried-items modal; esc or enter closes.
func (m *Model) updateInventory(msg tea.KeyMsg) {
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
}

// modalListHeight is how many list rows a scrollable modal shows.
func (m Model) modalListHeight() int {
	h := m.mainRowHeight() - 8 // borders, title, hint, scroll markers
	if h < 3 {
		h = 3
	}
	return h
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
