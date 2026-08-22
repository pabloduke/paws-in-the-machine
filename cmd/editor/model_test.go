package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func testModel(t *testing.T) *model {
	t.Helper()
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	m := newModel()
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 28})
	return m
}

func key(k string) tea.KeyMsg {
	switch k {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
}

func press(m *model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

func typeText(m *model, text string) {
	for _, r := range text {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestInitialMenuUsesSharedFrameAndDeclaredChoices(t *testing.T) {
	m := testModel(t)
	view := m.View()
	for _, want := range []string{
		"PAWS_IN_THE_SHELL",
		"GAME EDITOR",
		"> 1. Create",
		"2. Edit",
		"3. Place",
		"╔",
		"╝",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("initial menu missing %q:\n%s", want, view)
		}
	}
}

func TestCyberpunkFrameIsLargerAndTitlesAreCentered(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })

	m := newModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	view := m.View()
	for _, styledText := range []string{
		brandStyle.Render("PAWS_IN_THE_SHELL"),
		titleStyle.Render("GAME EDITOR"),
		selectedMenuStyle.Render("1. Create"),
	} {
		if !strings.Contains(view, styledText) {
			t.Fatalf("cyberpunk render missing themed element %q", styledText)
		}
	}
	if !strings.Contains(view, "\x1b[38;2;") || !strings.Contains(view, "\x1b[48;2;") {
		t.Fatal("cyberpunk render did not emit truecolor foreground and background styling")
	}

	lines := strings.Split(ansi.Strip(view), "\n")
	top, bottom := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "╔") {
			top = i
		}
		if strings.Contains(line, "╚") {
			bottom = i
		}
	}
	if top < 0 || bottom < 0 {
		t.Fatal("rendered panel did not contain both borders")
	}
	topLine := lines[top]
	leftCorner := strings.Index(topLine, "╔")
	rightCorner := strings.LastIndex(topLine, "╗")
	if inside := rightCorner - leftCorner - 1; inside < 51 {
		t.Fatalf("panel did not retain the 50%%-larger baseline width: got %d", inside)
	}
	if insideRows := bottom - top - 1; insideRows < 24 {
		t.Fatalf("panel did not retain the 50%%-larger baseline height: got %d", insideRows)
	}

	for _, title := range []string{"PAWS_IN_THE_SHELL", "GAME EDITOR"} {
		assertCenteredInFrame(t, lines, title)
	}
}

func assertCenteredInFrame(t *testing.T, lines []string, title string) {
	t.Helper()
	for _, line := range lines {
		if !strings.Contains(line, title) {
			continue
		}
		leftBorder := strings.Index(line, "║")
		rightBorder := strings.LastIndex(line, "║")
		if leftBorder < 0 || rightBorder <= leftBorder {
			t.Fatalf("title %q was not inside the frame", title)
		}
		inside := line[leftBorder+len("║") : rightBorder]
		left := strings.Index(inside, title)
		right := len(inside) - left - len(title)
		if delta := left - right; delta < -1 || delta > 1 {
			t.Fatalf("title %q is not centered: left=%d right=%d", title, left, right)
		}
		return
	}
	t.Fatalf("title %q was not rendered", title)
}

func TestNumbersNavigateDeclaredHierarchy(t *testing.T) {
	m := testModel(t)
	press(m, "1")
	if m.screen != createMenu || !strings.Contains(m.View(), "1. Create Item") {
		t.Fatalf("1 should open Create menu:\n%s", m.View())
	}
	press(m, "1")
	if m.screen != createItemMenu || !strings.Contains(m.View(), "1. Create World Item") {
		t.Fatalf("1 should open Create Item menu:\n%s", m.View())
	}
	press(m, "2")
	if m.screen != terminalForm || !strings.Contains(m.View(), "Enter Username:") {
		t.Fatalf("2 should open terminal form:\n%s", m.View())
	}
}

func TestSelectorMovesAndEnterActivates(t *testing.T) {
	m := testModel(t)
	press(m, "down", "down")
	if !strings.Contains(m.View(), "> 3. Place") {
		t.Fatalf("selector should move to Place:\n%s", m.View())
	}
	press(m, "enter")
	if m.screen != placeScreen || !strings.Contains(m.View(), "PLACE") {
		t.Fatalf("Enter should activate selected menu item:\n%s", m.View())
	}
	press(m, "1")
	if m.screen != mainMenu {
		t.Fatal("Back-only placeholder should return to the main menu")
	}
}

func TestEscapeReturnsOneMenuLevel(t *testing.T) {
	m := testModel(t)
	press(m, "1", "1", "esc")
	if m.screen != createMenu {
		t.Fatalf("Esc from Create Item should return to Create, got %v", m.screen)
	}
	press(m, "esc")
	if m.screen != mainMenu {
		t.Fatalf("Esc from Create should return to main, got %v", m.screen)
	}
}

func TestWorldItemFormNavigatesFieldsKindsAndActions(t *testing.T) {
	m := testModel(t)
	press(m, "1", "1", "1")
	if m.screen != worldItemForm {
		t.Fatal("world item form did not open")
	}
	typeText(m, "mug")
	press(m, "down", "right")
	if got := itemKinds[m.itemKind]; got != "Fixed" {
		t.Fatalf("Right should select Fixed, got %q", got)
	}
	press(m, "down")
	typeText(m, "a mug")
	press(m, "down")
	typeText(m, "a full description")
	press(m, "down")

	view := m.View()
	for _, want := range []string{"mug", "< Fixed >", "a mug", "a full description", "> Save", "Cancel"} {
		if !strings.Contains(view, want) {
			t.Fatalf("world item form missing %q:\n%s", want, view)
		}
	}
	press(m, "enter")
	if m.screen != createItemMenu {
		t.Fatal("Save should return to Create Item without persistence")
	}
	press(m, "1")
	if m.itemName.Value() != "" || m.itemShort.Value() != "" || m.itemFull.Value() != "" || m.itemKind != 0 {
		t.Fatal("reopened form retained values even though persistence is not implemented")
	}
}

func TestTerminalFormSaveAndCancelDoNotPersist(t *testing.T) {
	m := testModel(t)
	press(m, "1", "1", "2")
	typeText(m, "user")
	press(m, "down")
	typeText(m, "host")
	press(m, "down", "down")
	if !strings.Contains(m.View(), "> Cancel") {
		t.Fatalf("Cancel should be selector-controlled:\n%s", m.View())
	}
	press(m, "enter")
	if m.screen != createItemMenu {
		t.Fatal("Cancel should return to Create Item")
	}
	press(m, "2")
	if m.username.Value() != "" || m.hostname.Value() != "" {
		t.Fatal("terminal form retained values even though persistence is not implemented")
	}
}

func TestUndefinedChoicesAreFramedBackOnlyScreens(t *testing.T) {
	m := testModel(t)
	press(m, "1", "2")
	view := m.View()
	if m.screen != createNPCScreen || !strings.Contains(view, "CREATE NPC") || !strings.Contains(view, "> 1. Back") {
		t.Fatalf("undefined Create NPC destination should be a Back-only screen:\n%s", view)
	}
}
