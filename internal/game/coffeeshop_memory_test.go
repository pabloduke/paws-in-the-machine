package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

// baristaTalk opens a fresh dialogue session with the coffee-shop
// barista, the way the UI's meow command does.
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

// Burning her closes the invited post-it look, but stealth remains open:
// failure is a fork, not a wall.
func TestBaristaBurnedIsAForkNotAWall(t *testing.T) {
	w := game.NewWorld()
	w.Stats.Charm = 8
	w.Stats.Stealth = 100

	// Softening her after the burn does not restore the invitation.
	w.Flags["barista_burned"] = true
	choose(t, baristaTalk(t, w), "Purr")
	if !w.Flags["barista_softened"] {
		t.Fatal("a burned barista should still be softenable by charm")
	}
	eng := engine.New(w)
	eng.Execute("north")
	eng.Execute("look post-it")
	if w.Flags["read_microslop_badge"] {
		t.Fatal("a burned barista must not grant the invited post-it look")
	}

	// The hard route remains available through a guaranteed stealth pass.
	eng.Execute("sneak post-it")
	if !w.Flags["read_microslop_badge"] {
		t.Fatal("the badge must stay reachable through stealth after charm closes")
	}

}

// The warm path grants the invited post-it look.
func TestBaristaInvitationReachesBadge(t *testing.T) {
	w := game.NewWorld()
	w.Stats.Charm = 8

	choose(t, baristaTalk(t, w), "Purr")
	if hasOption(baristaTalk(t, w), "Microslop") {
		t.Fatal("the barista must never speak the Microslop credential")
	}
	eng := engine.New(w)
	eng.Execute("north")
	if out := eng.Execute("look post-it"); !strings.Contains(out, "password `apple`") {
		t.Fatalf("looking at the post-it should expose its credential: %q", out)
	}
	if !w.Flags["read_microslop_badge"] {
		t.Fatal("looking at the post-it should record the Microslop credential")
	}
	if out := eng.Execute("examine post-it"); !strings.Contains(out, "1008476") {
		t.Fatalf("examine post-it should remain directly addressable: %q", out)
	}
	if out := eng.Execute("read post-it"); !strings.Contains(out, "apple") {
		t.Fatalf("read post-it should remain directly addressable: %q", out)
	}
}

// Getting caught at the backpack burns the invitation, applies the cold
// memory, and leaves a harder stealth retry available.
func TestBadgeSneakFailureFork(t *testing.T) {
	w := game.NewWorld()
	w.Seed = 3
	w.Stats.Stealth = 0
	eng := engine.New(w)
	eng.Execute("north")

	if out := eng.Execute("sneak post-it"); !strings.Contains(out, "catches") {
		t.Fatalf("an impossible stealth attempt should be caught: %q", out)
	}
	if !w.Flags["barista_burned"] || w.Flags["read_microslop_badge"] {
		t.Fatalf("failure should burn charm without granting the credential: %v", w.Flags)
	}

	w.Stats.Stealth = 100
	eng.Execute("sneak post-it")
	if !w.Flags["read_microslop_badge"] {
		t.Fatal("stealth must remain open after the failure fork")
	}
}
