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

func newSized(w *engine.World) tea.Model {
	lipgloss.SetColorProfile(termenv.ANSI)
	var mod tea.Model = New(engine.New(w), "intro")
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return mod
}

func typeLine(mod tea.Model, line string) tea.Model {
	for _, r := range line {
		mod, _ = mod.Update(kr(r))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	return mod
}

func TestLevelUpModalMustSpend(t *testing.T) {
	w := game.NewWorld()
	engine.AwardXP(w, engine.LevelCostBase) // level 2, one stat point
	before := w.Stats.Agility

	mod := newSized(w)
	mod = typeLine(mod, "look") // any command surfaces the pending point
	if mm := mod.(Model); mm.modal != modalLevelUp {
		t.Fatalf("pending stat point should open the level-up modal, got modal=%d", mm.modal)
	}
	if !strings.Contains(mod.View(), "LEVEL UP!") {
		t.Fatalf("level-up modal not rendered")
	}

	mod, _ = mod.Update(spec(tea.KeyEsc)) // must spend: esc is a no-op
	if mm := mod.(Model); mm.modal != modalLevelUp {
		t.Fatalf("esc must not close the must-spend modal")
	}

	mod, _ = mod.Update(spec(tea.KeyDown)) // highlight agility
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mm := mod.(Model)
	if w.Stats.Agility != before+1 {
		t.Fatalf("enter should train the highlighted stat: agility %d -> %d", before, w.Stats.Agility)
	}
	if w.StatPoints != 0 || mm.modal != modalNone {
		t.Fatalf("spending the last point should close the modal; points=%d modal=%d",
			w.StatPoints, mm.modal)
	}
}

func TestInventoryModalOpensAndCloses(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "i")
	if mm := mod.(Model); mm.modal != modalInventory {
		t.Fatalf("'i' should open the inventory modal, got modal=%d", mm.modal)
	}
	if view := mod.View(); !strings.Contains(view, "INVENTORY") {
		t.Fatalf("inventory modal not rendered")
	}
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if mm := mod.(Model); mm.modal != modalNone {
		t.Fatalf("esc should close the inventory modal")
	}
}

func TestStatsModalOpensAndCloses(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "stats")
	if mm := mod.(Model); mm.modal != modalStats {
		t.Fatalf("'stats' should open the stats modal, got modal=%d", mm.modal)
	}
	if view := mod.View(); !strings.Contains(view, "Stealth") {
		t.Fatalf("stats modal should render the stat sheet")
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if mm := mod.(Model); mm.modal != modalNone {
		t.Fatalf("enter should close the stats modal")
	}
}
