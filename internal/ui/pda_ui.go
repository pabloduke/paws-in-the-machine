package ui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// The PDA surface (docs/systems/hacking.md): "use pda" opens Buddy's
// carried read-only slab as a centered overlay — menus only, never a
// shell. Check notes reads the deck's .md/.txt mirror; Scan ports
// runs the passive sniffer. Nothing can be hacked or written from it.

type pdaMode int

const (
	pdaClosed pdaMode = iota
	pdaMenu
	pdaNotes
	pdaReading
	pdaHosts
	pdaReport
)

type pdaSurface struct{}

func (pdaSurface) Active(m *Model) bool { return m.pdaMode != pdaClosed }

// Intercept claims "use <pda>" the way the shell claims "use <deck>".
func (pdaSurface) Intercept(m *Model, cmd engine.Command) bool {
	if cmd.Verb != "use" || cmd.Object == "" {
		return false
	}
	target := m.eng.World.InScope(cmd.Object)
	if target == nil {
		return false
	}
	p, isPDA := engine.Part[hacking.PDA](target)
	if !isPDA {
		return false
	}
	m.openPDA(p)
	return true
}

// openPDA wakes the slab — shared by "use pda" and the left panel's
// UPLINK row.
func (m *Model) openPDA(p hacking.PDA) {
	m.pdaCfg = p
	m.pdaMode = pdaMenu
	m.pdaSel = 0
	m.input.Blur()
	m.entries = append(m.entries, dimStyle.Render("[you thumb the PDA awake]"))
	m.refreshLog()
}

// pdaMenuItems is the main menu, in order.
var pdaMenuItems = []string{"Check notes", "Scan ports", "Put it away"}

// HandleKey drives whichever PDA screen is up: up/down move, enter
// selects, esc backs out one screen (menu → closed).
func (pdaSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc:
		m.pdaBack()
	case tea.KeyUp:
		if n := m.pdaListLen(); n > 0 {
			m.pdaSel = (m.pdaSel - 1 + n) % n
		}
	case tea.KeyDown:
		if n := m.pdaListLen(); n > 0 {
			m.pdaSel = (m.pdaSel + 1) % n
		}
	case tea.KeyEnter:
		m.pdaPick()
	case tea.KeyRunes:
		if len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9' {
			if n := int(msg.Runes[0] - '0'); n <= m.pdaListLen() {
				m.pdaSel = n - 1
			}
		}
	}
	return nil
}

// Overlay composites the PDA over the main row.
func (pdaSurface) Overlay(m *Model, bg string) string {
	return m.centerModal(bg, m.pdaView(), 64)
}

// pdaListLen is the row count of the current screen's list.
func (m *Model) pdaListLen() int {
	switch m.pdaMode {
	case pdaMenu:
		return len(pdaMenuItems)
	case pdaNotes:
		return len(m.pdaFiles) + 1 // + Back
	case pdaHosts:
		return len(m.pdaHosts) + 1 // + Back
	}
	return 0
}

// pdaBack steps out one screen; from the main menu it pockets the PDA.
func (m *Model) pdaBack() {
	switch m.pdaMode {
	case pdaReading:
		m.pdaMode = pdaNotes
	case pdaReport:
		m.pdaMode = pdaHosts
	case pdaNotes, pdaHosts:
		m.pdaMode = pdaMenu
		m.pdaSel = 0
	default:
		m.pdaClose()
	}
}

// pdaClose pockets the PDA and gives the prompt back. Reading can set
// story flags (OnRead), so run one checkpoint like a scene ending.
func (m *Model) pdaClose() {
	m.pdaMode = pdaClosed
	m.pdaCfg = hacking.PDA{}
	m.pdaFiles, m.pdaHosts = nil, nil
	m.input.Focus()
	m.entries = append(m.entries, dimStyle.Render("[you pocket the PDA]"))
	m.eng.World.CheckEvents()
	m.refreshLog()
	m.maybeLevelUp()
}

// pdaPick applies enter on the current screen.
func (m *Model) pdaPick() {
	switch m.pdaMode {
	case pdaMenu:
		switch m.pdaSel {
		case 0: // Check notes
			m.pdaFiles = hacking.TextFiles(m.eng.World, m.pdaCfg.Net[m.pdaCfg.Host])
			m.pdaMode = pdaNotes
			m.pdaSel = 0
		case 1: // Scan ports
			m.pdaHosts = hacking.KnownHosts(m.pdaCfg.Net, m.pdaCfg.Host)
			m.pdaMode = pdaHosts
			m.pdaSel = 0
		default: // Put it away
			m.pdaClose()
		}
	case pdaNotes:
		if m.pdaSel >= len(m.pdaFiles) { // Back
			m.pdaBack()
			return
		}
		doc := m.pdaFiles[m.pdaSel].Read(m.eng.World)
		m.pdaDocTitle = doc.Path
		if doc.Kind == hacking.DocumentMarkdown {
			m.pdaDocText = hacking.RenderMarkdown(doc.Text, 58)
		} else {
			m.pdaDocText = doc.Text
		}
		m.pdaMode = pdaReading
	case pdaHosts:
		if m.pdaSel >= len(m.pdaHosts) { // Back
			m.pdaBack()
			return
		}
		host := m.pdaHosts[m.pdaSel]
		m.pdaDocTitle = host
		m.pdaDocText = hacking.PortReport(m.eng.World, m.pdaCfg.Net, host)
		m.pdaMode = pdaReport
	case pdaReading, pdaReport:
		m.pdaBack()
	}
}

// pdaView renders the current PDA screen.
func (m Model) pdaView() string {
	var b strings.Builder
	b.WriteString(roomTitleStyle.Render("◧ PDA"))
	switch m.pdaMode {
	case pdaMenu:
		b.WriteString("\n")
		m.pdaList(&b, pdaMenuItems)
	case pdaNotes:
		b.WriteString("  " + dimStyle.Render("· notes on the deck") + "\n")
		rows := make([]string, 0, len(m.pdaFiles)+1)
		for _, f := range m.pdaFiles {
			rows = append(rows, f.Path)
		}
		rows = append(rows, "Back")
		m.pdaList(&b, rows)
	case pdaHosts:
		b.WriteString("  " + dimStyle.Render("· port sniffer") + "\n")
		rows := append(append([]string{}, m.pdaHosts...), "Back")
		m.pdaList(&b, rows)
	case pdaReading, pdaReport:
		b.WriteString("  " + dimStyle.Render("· "+m.pdaDocTitle) + "\n\n")
		b.WriteString(m.pdaDocText)
	}
	b.WriteString("\n\n" + dimStyle.Render(m.pdaHint()))
	return b.String()
}

// pdaList writes a highlighted menu list.
func (m Model) pdaList(b *strings.Builder, rows []string) {
	sel := m.pdaSel
	if sel >= len(rows) {
		sel = 0
	}
	for i, row := range rows {
		line := strconv.Itoa(i+1) + ". " + row
		if i == sel {
			line = hubSelStyle.Render(line)
		}
		b.WriteString("\n" + line)
	}
}

// pdaHint is the key legend for the current screen.
func (m Model) pdaHint() string {
	switch m.pdaMode {
	case pdaReading, pdaReport:
		return "esc/enter to go back"
	case pdaMenu:
		return "up/down to highlight · enter to select · esc to put it away"
	}
	return "up/down to highlight · enter to select · esc to go back"
}
