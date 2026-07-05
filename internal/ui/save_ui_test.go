package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// newSaving is newSized with a temp save slot.
func newSaving(t *testing.T, w *engine.World) tea.Model {
	t.Helper()
	mm := newSized(w).(Model)
	mm.SavePath = filepath.Join(t.TempDir(), "save.json")
	return mm
}

// lastEntry returns the newest LOG line.
func lastEntry(mod tea.Model) string {
	mm := mod.(Model)
	return mm.entries[len(mm.entries)-1]
}

// Save mid-run, keep playing, load — the world snaps back.
func TestSaveLoadCycle(t *testing.T) {
	w := game.NewWorld()
	mod := newSaving(t, w)

	mod = typeLine(mod, "take shard")
	mod = typeLine(mod, "north") // coffee shop, shard in paw
	mod = typeLine(mod, "save")
	if log := lastEntry(mod); !strings.Contains(log, "Saved") {
		t.Fatalf("save should confirm: %q", log)
	}

	// Keep playing: back south, drop the shard.
	mod = typeLine(mod, "south")
	mod = typeLine(mod, "drop shard")
	if w.Room().ID != "lair" || w.Carried(w.FindID("shard")) {
		t.Fatalf("test setup: expected to be in the lair without the shard")
	}

	mod = typeLine(mod, "load")
	if log := lastEntry(mod); !strings.Contains(log, "Loaded") {
		t.Fatalf("load should confirm: %q", log)
	}
	if w.Room().ID != "coffeeshop" {
		t.Fatalf("load should restore position: %s", w.Room().ID)
	}
	if !w.Carried(w.FindID("shard")) {
		t.Fatalf("load should restore inventory")
	}
}

func TestLoadWithoutSave(t *testing.T) {
	mod := newSaving(t, game.NewWorld())
	mod = typeLine(mod, "load")
	if log := lastEntry(mod); !strings.Contains(log, "nothing kept yet") {
		t.Fatalf("missing-save message expected: %q", log)
	}
}
