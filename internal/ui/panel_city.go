package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// The city panel (docs/systems/hubs.md): the persistent left panel
// listing hubs to travel to, plus any decks in scope to log into.
// citySurface owns the keyboard while the panel is focused (Tab).

// panelKind tags a focusable row in the left panel.
type panelKind int

const (
	panelHub  panelKind = iota // a city district — Enter travels
	panelDeck                  // a terminal in reach — Enter logs in
)

// panelItem is one selectable row in the left panel.
type panelItem struct {
	kind   panelKind
	entity *engine.Entity
	name   string
}

// citySurface is the travel panel's input mode.
type citySurface struct{}

func (citySurface) Active(m *Model) bool { return m.panelFocused }

// focusCityPanel shifts focus to the panel (Tab from the prompt),
// preselecting the hub Buddy is in.
func (m *Model) focusCityPanel() {
	if len(m.panelItems()) == 0 {
		return
	}
	m.panelFocused = true
	m.selected = currentHubIndex(m.eng.World, hubs.List(m.eng.World))
}

// panelItems is the left panel's focusable list: hubs to travel to,
// then decks in scope to log into.
func (m Model) panelItems() []panelItem {
	var items []panelItem
	for _, h := range hubs.List(m.eng.World) {
		items = append(items, panelItem{panelHub, h, h.Name})
	}
	for _, d := range hacking.DecksInScope(m.eng.World) {
		items = append(items, panelItem{panelDeck, d, engine.DisplayName(m.eng.World, d)})
	}
	return items
}

// HandleKey handles keys while the city panel has focus. Enter
// travels to a hub or logs into a deck, depending on the row.
func (citySurface) HandleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	items := m.panelItems()
	if len(items) == 0 {
		m.panelFocused = false
		return nil
	}
	if m.selected >= len(items) {
		m.selected = 0
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyTab, tea.KeyEsc:
		m.panelFocused = false
	case tea.KeyUp:
		m.selected = (m.selected - 1 + len(items)) % len(items)
	case tea.KeyDown:
		m.selected = (m.selected + 1) % len(items)
	case tea.KeyEnter:
		item := items[m.selected]
		m.panelFocused = false
		switch item.kind {
		case panelHub:
			m.entries = append(m.entries, echoStyle.Render("> [travel] "+item.name))
			if out := hubs.Travel(m.eng.World, item.entity.ID); out != "" {
				m.entries = append(m.entries, out)
			}
			m.refreshLog()
		case panelDeck:
			if d, ok := engine.Part[hacking.Deck](item.entity); ok {
				m.openShell(d)
			}
		}
	}
	return nil
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

// cityPanel renders the hub list (docs/systems/hubs.md).
func (m Model) cityPanel() string {
	items := m.panelItems()
	cur := hubs.Current(m.eng.World)

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("THE CITY"))
	prevKind := panelHub
	for i, it := range items {
		// A blank line and a green subhead set the deck(s) apart from
		// the hubs — logging in is not walking somewhere.
		if it.kind == panelDeck && (i == 0 || prevKind == panelHub) {
			b.WriteString("\n\n" + deckTitleStyle.Render("// UPLINK"))
		}
		prevKind = it.kind

		marker := "  "
		if it.kind == panelHub && it.entity == cur {
			marker = "* "
		}
		row := marker + it.name
		switch {
		case m.panelFocused && i == m.selected:
			row = hubSelStyle.Render(row)
		case it.kind == panelDeck:
			row = deckRowStyle.Render(row)
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
