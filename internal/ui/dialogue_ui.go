package ui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

// The dialogue surface (docs/systems/dialogue.md): "talk <npc>" opens
// the conversation menu as a centered overlay; the LOG keeps the
// transcript, the overlay is the live view.

type dialogueSurface struct{}

func (dialogueSurface) Active(m *Model) bool { return m.dialogue != nil }

// Intercept claims the "talk" verb and starts a session.
func (dialogueSurface) Intercept(m *Model, cmd engine.Command) bool {
	if cmd.Verb != "talk" {
		return false
	}
	if cmd.Object == "" {
		m.entries = append(m.entries, "Talk to whom?")
		return true
	}
	target := m.eng.World.InScope(cmd.Object)
	if target == nil {
		m.entries = append(m.entries, "You don't see any "+strconv.Quote(cmd.Object)+" here.")
		return true
	}
	session, err := dialogue.Start(m.eng.World, target)
	if err != nil {
		m.entries = append(m.entries, err.Error())
		return true
	}
	m.dialogue = session
	m.dlgSel = 0
	m.input.Blur()
	// The stage holds still while the scene plays (presence.md).
	m.eng.World.HoldPlacements = true
	m.entries = append(m.entries, session.Speaker()+": "+session.Text())
	return true
}

// HandleKey drives the conversation menu: up/down move the highlight,
// enter picks, 1-9 highlight directly, esc leaves.
func (dialogueSurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	options := m.dialogue.Options()
	if m.dlgSel >= len(options) {
		m.dlgSel = 0
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc:
		m.dialogue = nil
		m.input.Focus()
		m.entries = append(m.entries, dimStyle.Render("[conversation ended]"))
		m.endScene()
		m.refreshLog()
		m.maybeLevelUp()
	case tea.KeyUp:
		if len(options) > 0 {
			m.dlgSel = (m.dlgSel - 1 + len(options)) % len(options)
		}
	case tea.KeyDown:
		if len(options) > 0 {
			m.dlgSel = (m.dlgSel + 1) % len(options)
		}
	case tea.KeyEnter:
		m.pickDialogue(m.dlgSel + 1)
	case tea.KeyRunes:
		// Numbers move the highlight; only enter commits.
		if len(msg.Runes) != 1 || msg.Runes[0] < '1' || msg.Runes[0] > '9' {
			return nil
		}
		if n := int(msg.Runes[0] - '0'); n <= len(options) {
			m.dlgSel = n - 1
		}
	}
	return nil
}

// Overlay composites the conversation over the main row.
func (dialogueSurface) Overlay(m *Model, bg string) string {
	return m.centerModal(bg, m.dialogueView(), 60)
}

// pickDialogue applies a 1-based choice and records the exchange in
// the LOG. Locked picks log the refusal and stay on the node.
func (m *Model) pickDialogue(n int) {
	options := m.dialogue.Options()
	if n < 1 || n > len(options) {
		return
	}
	locked := options[n-1].Locked
	m.entries = append(m.entries, echoStyle.Render("> "+options[n-1].Choice.Text))
	if out := m.dialogue.Pick(n); out != "" {
		m.entries = append(m.entries, out)
	}
	if m.dialogue.Done() {
		m.dialogue = nil
		m.input.Focus()
		m.endScene()
		m.maybeLevelUp()
	} else if !locked {
		m.dlgSel = 0
		m.entries = append(m.entries, m.dialogue.Speaker()+": "+m.dialogue.Text())
	}
	m.refreshLog()
}

// endScene releases the placement hold and runs one checkpoint so
// the stage catches up the moment the conversation is over.
func (m *Model) endScene() {
	m.eng.World.HoldPlacements = false
	m.eng.World.CheckEvents()
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
