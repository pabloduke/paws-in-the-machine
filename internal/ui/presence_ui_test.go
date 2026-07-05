package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// The barista beat end-to-end: show her the shard mid-conversation
// (flag flips live, but she stays put — the scene holds the stage),
// end the conversation (she slips to the back room), leave and
// return (the empty counter gets its event beat).
func TestBaristaPresenceBeat(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	mod = typeLine(mod, "take shard")
	mod = typeLine(mod, "north")
	mod = typeLine(mod, "talk barista")
	if mod.(Model).dialogue == nil {
		t.Fatalf("talk should open the conversation")
	}

	// Pick "Nudge the data-shard into view" (row 2: charm, shard, mrow, leave).
	mod, _ = mod.Update(kr('2'))
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if !w.Flags["barista_saw_shard"] {
		t.Fatalf("flag should be set live by the pick")
	}
	if mod.(Model).dialogue != nil {
		t.Fatalf("the shard line ends the conversation")
	}

	// Scene over: placement caught up — she's in the back room.
	barista := w.FindID("barista")
	if barista.Parent.ID != "backroom" {
		t.Fatalf("barista should be re-placed when the scene ends: %s", barista.Parent.ID)
	}
	if w.InScope("barista") != nil {
		t.Fatalf("barista should no longer be in scope in the coffee shop")
	}

	// Re-entering the coffee shop narrates the change.
	mod = typeLine(mod, "south")
	mod = typeLine(mod, "north")
	if len(w.Pending) != 1 || !strings.Contains(w.Pending[0], "unmanned") {
		t.Fatalf("empty-counter beat should fire on re-entry: %v", w.Pending)
	}
	if v := mod.View(); !strings.Contains(v, "unmanned") {
		t.Fatalf("event modal should present the beat:\n%s", v)
	}
	mod, _ = mod.Update(spec(tea.KeyEnter)) // dismiss

	// Once only: leave and return again, no repeat.
	mod = typeLine(mod, "south")
	mod = typeLine(mod, "north")
	if len(w.Pending) != 0 {
		t.Fatalf("empty-counter beat must not repeat: %v", w.Pending)
	}
}

// Mid-conversation, the stage must hold: the flag is true but the
// barista hasn't moved while her dialogue is still open.
func TestPlacementHeldDuringDialogue(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	mod = typeLine(mod, "take shard")
	mod = typeLine(mod, "north")
	mod = typeLine(mod, "talk barista")
	if !w.HoldPlacements {
		t.Fatalf("an open conversation should hold placements")
	}

	// Force her placement condition true without ending the scene.
	w.Flags["barista_saw_shard"] = true
	w.CheckEvents()
	if w.FindID("barista").Parent.ID != "coffeeshop" {
		t.Fatalf("interlocutor must not vanish mid-sentence")
	}

	// Esc ends the scene; the stage catches up.
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if w.HoldPlacements {
		t.Fatalf("closing the conversation should release the hold")
	}
	if w.FindID("barista").Parent.ID != "backroom" {
		t.Fatalf("placement should catch up when the scene ends")
	}
}
