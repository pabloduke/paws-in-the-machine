package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	mainMenu screen = iota
	createMenu
	createItemMenu
	worldItemForm
	terminalForm
	createNPCScreen
	createQuestScreen
	createRoomScreen
	createHubScreen
	editScreen
	placeScreen
)

type menuEntry struct {
	label  string
	target screen
}

var screenMenus = map[screen][]menuEntry{
	mainMenu: {
		{label: "Create", target: createMenu},
		{label: "Edit", target: editScreen},
		{label: "Place", target: placeScreen},
	},
	createMenu: {
		{label: "Create Item", target: createItemMenu},
		{label: "Create NPC", target: createNPCScreen},
		{label: "Create Quest", target: createQuestScreen},
		{label: "Create Room", target: createRoomScreen},
		{label: "Create Hub", target: createHubScreen},
	},
	createItemMenu: {
		{label: "Create World Item", target: worldItemForm},
		{label: "Create Terminal", target: terminalForm},
	},
	createNPCScreen:   {{label: "Back", target: createMenu}},
	createQuestScreen: {{label: "Back", target: createMenu}},
	createRoomScreen:  {{label: "Back", target: createMenu}},
	createHubScreen:   {{label: "Back", target: createMenu}},
	editScreen:        {{label: "Back", target: mainMenu}},
	placeScreen:       {{label: "Back", target: mainMenu}},
}

var screenTitles = map[screen]string{
	mainMenu:          "GAME EDITOR",
	createMenu:        "CREATE",
	createItemMenu:    "CREATE ITEM",
	worldItemForm:     "CREATE WORLD ITEM",
	terminalForm:      "CREATE TERMINAL",
	createNPCScreen:   "CREATE NPC",
	createQuestScreen: "CREATE QUEST",
	createRoomScreen:  "CREATE ROOM",
	createHubScreen:   "CREATE HUB",
	editScreen:        "EDIT",
	placeScreen:       "PLACE",
}

var itemKinds = []string{"Takeable", "Fixed", "Scenery"}

const (
	worldNameRow = iota
	worldKindRow
	worldShortRow
	worldFullRow
	worldSaveRow
	worldCancelRow
	worldRowCount
)

const (
	terminalUsernameRow = iota
	terminalHostRow
	terminalSaveRow
	terminalCancelRow
	terminalRowCount
)

type model struct {
	screen screen
	cursor int
	w      int
	h      int

	itemName  textinput.Model
	itemShort textinput.Model
	itemFull  textinput.Model
	itemKind  int

	username textinput.Model
	hostname textinput.Model
}

func newModel() *model {
	m := &model{
		screen:    mainMenu,
		itemName:  newInput(48),
		itemShort: newInput(96),
		itemFull:  newInput(256),
		username:  newInput(48),
		hostname:  newInput(96),
	}
	return m
}

func newInput(limit int) textinput.Model {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = limit
	in.Width = min(limit, 48)
	in.TextStyle = inputTextStyle
	in.Cursor.Style = inputCursorStyle
	return in
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		inputWidth := min(48, max(1, msg.Width-36))
		m.itemName.Width = inputWidth
		m.itemShort.Width = inputWidth
		m.itemFull.Width = inputWidth
		m.username.Width = inputWidth
		m.hostname.Width = inputWidth
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.screen == worldItemForm {
			return m.updateWorldItemForm(msg)
		}
		if m.screen == terminalForm {
			return m.updateTerminalForm(msg)
		}
		return m.updateMenu(msg)
	}
	return m, nil
}

func (m *model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := screenMenus[m.screen]
	if len(entries) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "up":
		m.cursor = (m.cursor - 1 + len(entries)) % len(entries)
	case "down":
		m.cursor = (m.cursor + 1) % len(entries)
	case "enter":
		return m.open(entries[m.cursor].target)
	case "esc":
		if m.screen != mainMenu {
			return m.open(parentScreen(m.screen))
		}
	case "q":
		if m.screen == mainMenu {
			return m, tea.Quit
		}
	default:
		if len(msg.Runes) == 1 {
			n := int(msg.Runes[0] - '1')
			if n >= 0 && n < len(entries) {
				return m.open(entries[n].target)
			}
		}
	}
	return m, nil
}

func (m *model) open(next screen) (tea.Model, tea.Cmd) {
	m.screen = next
	m.cursor = 0
	m.blurInputs()
	switch next {
	case worldItemForm:
		m.resetWorldItemForm()
		m.itemName.Focus()
		return m, textinput.Blink
	case terminalForm:
		m.resetTerminalForm()
		m.username.Focus()
		return m, textinput.Blink
	default:
		return m, nil
	}
}

func parentScreen(s screen) screen {
	switch s {
	case createMenu:
		return mainMenu
	case createItemMenu, createNPCScreen, createQuestScreen, createRoomScreen, createHubScreen:
		return createMenu
	case worldItemForm, terminalForm:
		return createItemMenu
	default:
		return mainMenu
	}
}

func (m *model) updateWorldItemForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m.open(createItemMenu)
	}
	if moved := m.moveFormCursor(msg, worldRowCount); moved {
		m.focusWorldItemRow()
		return m, textinput.Blink
	}
	if m.cursor == worldKindRow {
		switch msg.String() {
		case "left":
			m.itemKind = (m.itemKind - 1 + len(itemKinds)) % len(itemKinds)
		case "right":
			m.itemKind = (m.itemKind + 1) % len(itemKinds)
		}
	}
	if msg.String() == "enter" {
		switch m.cursor {
		case worldSaveRow, worldCancelRow:
			return m.open(createItemMenu)
		default:
			m.cursor++
			m.focusWorldItemRow()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	switch m.cursor {
	case worldNameRow:
		m.itemName, cmd = m.itemName.Update(msg)
	case worldShortRow:
		m.itemShort, cmd = m.itemShort.Update(msg)
	case worldFullRow:
		m.itemFull, cmd = m.itemFull.Update(msg)
	}
	return m, cmd
}

func (m *model) updateTerminalForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m.open(createItemMenu)
	}
	if moved := m.moveFormCursor(msg, terminalRowCount); moved {
		m.focusTerminalRow()
		return m, textinput.Blink
	}
	if msg.String() == "enter" {
		switch m.cursor {
		case terminalSaveRow, terminalCancelRow:
			return m.open(createItemMenu)
		default:
			m.cursor++
			m.focusTerminalRow()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	switch m.cursor {
	case terminalUsernameRow:
		m.username, cmd = m.username.Update(msg)
	case terminalHostRow:
		m.hostname, cmd = m.hostname.Update(msg)
	}
	return m, cmd
}

func (m *model) moveFormCursor(msg tea.KeyMsg, count int) bool {
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		m.cursor = (m.cursor - 1 + count) % count
		return true
	case tea.KeyDown, tea.KeyTab:
		m.cursor = (m.cursor + 1) % count
		return true
	}
	return false
}

func (m *model) focusWorldItemRow() {
	m.blurInputs()
	switch m.cursor {
	case worldNameRow:
		m.itemName.Focus()
	case worldShortRow:
		m.itemShort.Focus()
	case worldFullRow:
		m.itemFull.Focus()
	}
}

func (m *model) focusTerminalRow() {
	m.blurInputs()
	switch m.cursor {
	case terminalUsernameRow:
		m.username.Focus()
	case terminalHostRow:
		m.hostname.Focus()
	}
}

func (m *model) blurInputs() {
	m.itemName.Blur()
	m.itemShort.Blur()
	m.itemFull.Blur()
	m.username.Blur()
	m.hostname.Blur()
}

func (m *model) resetWorldItemForm() {
	m.itemName.SetValue("")
	m.itemShort.SetValue("")
	m.itemFull.SetValue("")
	m.itemKind = 0
}

func (m *model) resetTerminalForm() {
	m.username.SetValue("")
	m.hostname.SetValue("")
}

func (m *model) View() string {
	var body []string
	if m.screen == worldItemForm {
		body = m.worldItemRows()
	} else if m.screen == terminalForm {
		body = m.terminalRows()
	} else {
		body = m.menuRows()
	}

	// The initial roughs used a 34x16 interior. The prototype deliberately
	// scales that footprint by 50% while remaining responsive on smaller
	// terminals.
	innerWidth := 51
	for _, line := range body {
		innerWidth = max(innerWidth, lipgloss.Width(line))
	}
	if m.w > 0 {
		innerWidth = min(innerWidth, max(20, m.w-6))
	}
	lines := []string{
		center("PAWS_IN_THE_SHELL", innerWidth, brandStyle),
		center(screenTitles[m.screen], innerWidth, titleStyle),
		"",
		"",
	}
	lines = append(lines, body...)
	innerHeight := max(24, len(lines)+4)
	if m.h > 0 {
		innerHeight = min(innerHeight, max(len(lines), m.h-4))
	}

	panel := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(cyberCyan).
		Background(cyberPanel).
		Padding(1, 2).
		Width(innerWidth + 4).
		Height(innerHeight + 2).
		Render(strings.Join(lines, "\n"))
	if m.w > 0 && m.h > 0 {
		placed := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, panel)
		return lipgloss.NewStyle().
			Background(cyberBackground).
			Width(m.w).
			Height(m.h).
			Render(placed)
	}
	return panel
}

func (m *model) menuRows() []string {
	entries := screenMenus[m.screen]
	rows := make([]string, 0, len(entries))
	for i, entry := range entries {
		label := fmt.Sprintf("%d. %s", i+1, entry.label)
		if i == m.cursor {
			label = selectedMenuStyle.Render(label)
		} else {
			label = menuStyle.Render(label)
		}
		rows = append(rows, selector(i == m.cursor)+" "+label)
	}
	return rows
}

func (m *model) worldItemRows() []string {
	return []string{
		formRow(m.cursor == worldNameRow, "Enter Item Name:", m.itemName.View()),
		formRow(m.cursor == worldKindRow, "Select Item Type:", choiceStyle.Render("< "+itemKinds[m.itemKind]+" >")),
		formRow(m.cursor == worldShortRow, "Enter Short Description:", m.itemShort.View()),
		formRow(m.cursor == worldFullRow, "Enter Full Description:", m.itemFull.View()),
		actionRow(m.cursor == worldSaveRow, "Save"),
		actionRow(m.cursor == worldCancelRow, "Cancel"),
	}
}

func (m *model) terminalRows() []string {
	return []string{
		formRow(m.cursor == terminalUsernameRow, "Enter Username:", m.username.View()),
		formRow(m.cursor == terminalHostRow, "Enter Host Name:", m.hostname.View()),
		actionRow(m.cursor == terminalSaveRow, "Save"),
		actionRow(m.cursor == terminalCancelRow, "Cancel"),
	}
}

func formRow(selected bool, label, value string) string {
	labelText := fmt.Sprintf("%-25s", label)
	if selected {
		labelText = selectedFieldLabelStyle.Render(labelText)
	} else {
		labelText = fieldLabelStyle.Render(labelText)
	}
	return fmt.Sprintf("%s %s %s", selector(selected), labelText, value)
}

func actionRow(selected bool, label string) string {
	if selected {
		return selector(true) + " " + selectedMenuStyle.Render(label)
	}
	return selector(false) + " " + menuStyle.Render(label)
}

func selector(selected bool) string {
	if selected {
		return selectorStyle.Render(">")
	}
	return " "
}

func center(s string, width int, style lipgloss.Style) string {
	return style.Width(width).Align(lipgloss.Center).Render(s)
}
