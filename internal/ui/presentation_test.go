package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestSemanticLogStoresNoANSI(t *testing.T) {
	m := New(engine.New(game.NewWorld()), "intro")
	m.appendLog(textEcho, "> look")
	m.appendLog(textDim, "[quiet]")

	if raw := rawLog(m.entries); strings.Contains(raw, "\x1b[") {
		t.Fatalf("semantic log stored ANSI: %q", raw)
	}
	if got := m.entries[1]; got.Role != textEcho || got.Text != "> look" {
		t.Fatalf("echo semantics were not preserved: %#v", got)
	}
}

func TestInjectedThemeRestylesExistingLogWithoutTouchingWorld(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })

	const alternateID = "presentation-test"
	alternate := themeRegistry[defaultThemeID]
	alternate.ID = alternateID
	alternate.Overworld.echo = lipgloss.NewStyle().Foreground(lipgloss.Color("#123456"))
	alternate.Terminal.local.bright = lipgloss.Color("#654321")
	themeRegistry[alternateID] = alternate
	t.Cleanup(func() { delete(themeRegistry, alternateID) })

	w := game.NewWorld()
	m := New(engine.New(w), "intro")
	m.appendLog(textEcho, "> look")
	before := renderLog(m.entries, m.presentation().Overworld)
	roomBefore, flagsBefore := w.Room().ID, len(w.Flags)

	if !m.setTheme(alternateID) {
		t.Fatal("registered theme was rejected")
	}
	after := renderLog(m.entries, m.presentation().Overworld)
	if before == after {
		t.Fatal("switching themes did not restyle existing semantic log entries")
	}
	if raw := rawLog(m.entries); raw != "intro\n> look" {
		t.Fatalf("theme switch changed raw log content: %q", raw)
	}
	if w.Room().ID != roomBefore || len(w.Flags) != flagsBefore {
		t.Fatal("theme switch touched world state")
	}

	injected := New(engine.New(game.NewWorld()), "intro", WithTheme(alternateID))
	if injected.themeID != alternateID {
		t.Fatalf("WithTheme selected %q, want %q", injected.themeID, alternateID)
	}
}

func TestUnknownInjectedThemeFallsBack(t *testing.T) {
	m := New(engine.New(game.NewWorld()), "intro", WithTheme("missing"))
	if m.themeID != defaultThemeID {
		t.Fatalf("unknown theme selected %q, want fallback %q", m.themeID, defaultThemeID)
	}
}
