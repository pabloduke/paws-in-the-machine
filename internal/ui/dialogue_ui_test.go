package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func kr(r rune) tea.KeyMsg          { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
func spec(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func TestDialogueMenuFlow(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	w := game.NewWorld()
	w.Stats.Charm = 5 // below the barista's [Charm 8] gate
	m := New(engine.New(w), "intro")
	var mod tea.Model = m
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range "north" {
		mod, _ = mod.Update(kr(r))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	for _, r := range "meow barista" {
		mod, _ = mod.Update(kr(r))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))

	// Menu at first contact (charm 5, no shard): charm / tip-jar / mrow /
	// leave. Highlight the last row, then wrap past it.
	mod, _ = mod.Update(kr('4'))
	if mm := mod.(Model); mm.dlgSel != 3 {
		t.Fatalf("number should highlight: dlgSel=%d", mm.dlgSel)
	}
	if !strings.Contains(mod.View(), "\x1b[7m") {
		t.Fatalf("no reverse-video highlight in view")
	}
	mod, _ = mod.Update(spec(tea.KeyDown))
	if mm := mod.(Model); mm.dlgSel != 0 {
		t.Fatalf("down should wrap to 0: dlgSel=%d", mm.dlgSel)
	}
	mod, _ = mod.Update(spec(tea.KeyEnter)) // locked charm pick
	mm := mod.(Model)
	if mm.dialogue == nil {
		t.Fatalf("locked pick should stay in dialogue")
	}
	joined := rawLog(mm.entries)
	if !strings.Contains(joined, "isn't up to that yet") {
		t.Fatalf("expected locked refusal in log")
	}
	mod, _ = mod.Update(kr('4'))
	mod, _ = mod.Update(spec(tea.KeyEnter)) // Leave.
	if mm := mod.(Model); mm.dialogue != nil {
		t.Fatalf("Leave should end dialogue")
	}
}

func TestRewriteReachesDialogueIntercept(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	w := game.NewWorld()
	w.Rewrites["greet barista"] = "meow barista"
	m := New(engine.New(w), "intro")
	var mod tea.Model = m
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range "north" {
		mod, _ = mod.Update(kr(r))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	for _, r := range "greet barista" {
		mod, _ = mod.Update(kr(r))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if mm := mod.(Model); mm.dialogue == nil {
		t.Fatalf("rewrite expanding to a meow command should open the dialogue modal")
	}
}
