package game_test

import (
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// The deck lives in the lair and nowhere else (user ruling
// 2026-07-07): hacking starts at home. The lair is the base you scan
// from and return to — that's the point of having one.
func TestDeckStaysHome(t *testing.T) {
	w := game.NewWorld()

	if decks := hacking.DecksInScope(w); len(decks) != 1 || decks[0].ID != "deck" {
		t.Fatalf("the deck should be loggable from the lair: %v", decks)
	}

	hubs.Travel(w, "okuda")
	if decks := hacking.DecksInScope(w); len(decks) != 0 {
		t.Fatalf("no deck should be reachable outside the lair: %v", decks)
	}

	hubs.Travel(w, "neighborhood")
	if decks := hacking.DecksInScope(w); len(decks) != 1 {
		t.Fatalf("coming home should put the deck back in reach: %v", decks)
	}
}
