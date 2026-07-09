package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// The Resistance channel (docs/draft.md: Resistance-lite): the deck
// ships with the messenger, the welcome brief is waiting on first
// login, and later messages arrive as story flags land — the
// mission-giver reacts to play, never to a clock.
func TestDeckMessengerResistanceChannel(t *testing.T) {
	w := game.NewWorld()
	d, ok := engine.Part[hacking.Deck](w.FindID("deck"))
	if !ok {
		t.Fatalf("the deck should carry the hacking.Deck component")
	}
	mgr := d.Messenger
	if !mgr.Enabled() || mgr.Contact != "resistance" {
		t.Fatalf("the deck should carry the resistance channel: %+v", mgr)
	}

	thread := mgr.Thread(w)
	if len(thread) != 1 || !strings.Contains(thread[0].Text, "barista") {
		t.Fatalf("the welcome brief should be waiting and point at the barista: %+v", thread)	}
	if mgr.Unread(w) != 1 {
		t.Fatalf("the welcome brief should start unread")
	}

	w.Flags["microslop_route_open"] = true // the backroom rack (backroom.go)
	thread = mgr.Thread(w)
	if len(thread) != 2 || !strings.Contains(thread[1].Text, "bridge") {
		t.Fatalf("the route flag should deliver the bridge message: %+v", thread)
	}
}

// `talk` is the messenger's friendly alias in ~/.aliases — the same
// verb that opens dialogue in the overworld.
func TestTalkAliasOpensMessenger(t *testing.T) {
	w := game.NewWorld()
	s := deckSession(t, w)
	if result := s.ExecDetailed("talk"); !result.Messenger {
		t.Fatalf("talk should expand to messenger and signal the panel: %+v", result)
	}
}
