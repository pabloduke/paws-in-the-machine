package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

// baristaTalk opens a fresh dialogue session with the coffee-shop
// barista, the way the UI's talk intercept does.
func baristaTalk(t *testing.T, w *engine.World) *dialogue.Session {
	t.Helper()
	npc := w.FindID("barista")
	if npc == nil {
		t.Fatal("the coffee shop should have a barista")
	}
	s, err := dialogue.Start(w, npc)
	if err != nil {
		t.Fatalf("opening dialogue with the barista: %v", err)
	}
	return s
}

// choose picks the first visible option whose text contains want, so the
// test reads by intent instead of pinning brittle choice indices.
func choose(t *testing.T, s *dialogue.Session, want string) string {
	t.Helper()
	for i, o := range s.Options() {
		if strings.Contains(o.Choice.Text, want) {
			return s.Choose(i + 1)
		}
	}
	t.Fatalf("no visible choice matching %q; options were %v", want, optionTexts(s))
	return ""
}

// hasOption reports whether a visible choice contains want.
func hasOption(s *dialogue.Session, want string) bool {
	for _, o := range s.Options() {
		if strings.Contains(o.Choice.Text, want) {
			return true
		}
	}
	return false
}

func optionTexts(s *dialogue.Session) []string {
	var out []string
	for _, o := range s.Options() {
		out = append(out, o.Choice.Text)
	}
	return out
}

// The greeting is NPC memory (#14): it runs neutral at first contact and
// cold once Buddy has burned her, driven purely by a per-NPC flag.
func TestBaristaGreetingRemembers(t *testing.T) {
	w := game.NewWorld()

	if got := baristaTalk(t, w).Text(); !strings.Contains(got, "You lost, orange") {
		t.Fatalf("first-contact greeting should be neutral, got %q", got)
	}

	// Burn her: the petty cat move sets the cold memory and ends the beat.
	s := baristaTalk(t, w)
	choose(t, s, "tip jar")
	if !w.Flags["barista_burned"] {
		t.Fatal("knocking the tip jar over should set barista_burned")
	}

	if got := baristaTalk(t, w).Text(); !strings.Contains(got, "broom with your name") {
		t.Fatalf("a burned barista should greet cold, got %q", got)
	}
}

// Burning her closes the hound-lure favor but never the apple reveal:
// cruelty costs goodwill, not the critical path (failure is a fork, not
// a wall).
func TestBaristaBurnedIsAForkNotAWall(t *testing.T) {
	w := game.NewWorld()
	w.Stats.Charm = 8

	// Even a burned barista can be won over, and the softened line still
	// hands out the Microslop password — the route survives the grudge.
	w.Flags["barista_burned"] = true
	choose(t, baristaTalk(t, w), "Purr")
	if !w.Flags["barista_softened"] {
		t.Fatal("a burned barista should still be softenable by charm")
	}
	choose(t, baristaTalk(t, w), "what Microslop did")
	if !w.Flags["knows_microslop_password"] {
		t.Fatal("apple must stay reachable through a burned barista (no soft-lock)")
	}

	// But the favor is gone: softened + burned hides the hound-lure line.
	if hasOption(baristaTalk(t, w), "too much") {
		t.Fatal("a burned barista should not offer to lure the hound")
	}
}

// The warm path is untouched: an unburned, softened barista still offers
// to call the hound off.
func TestBaristaLureSurvivesWhenNotBurned(t *testing.T) {
	w := game.NewWorld()
	w.Stats.Charm = 8

	choose(t, baristaTalk(t, w), "Purr")
	if !hasOption(baristaTalk(t, w), "too much") {
		t.Fatal("a softened, unburned barista should offer the hound lure")
	}
}
