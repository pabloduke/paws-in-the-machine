package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// The full failure-fork loop against real content (pinned seed):
// fail the hound → alerted (event beat, journal, −2) → shatter the
// mug (+4, consumed) → the same sneak now clears. Failure changes
// the situation; changed circumstances re-roll (the XCOM rule).
func TestHoundFailureFork(t *testing.T) {
	w := game.NewWorld()
	eng := engine.New(w)
	eng.Execute("north") // coffee shop
	w.Pending = nil      // not testing arrival beats here

	// Fail at Stealth 10: the world forks.
	if out := eng.Execute("sneak past hound"); !strings.Contains(out, "growl") {
		t.Fatalf("expected the sneak to fail at Stealth 10: %q", out)
	}
	if !w.Flags["hound_alerted"] {
		t.Fatalf("failure should alert the hound")
	}
	if len(w.Pending) != 1 || !strings.Contains(w.Pending[0], "on its feet") {
		t.Fatalf("the alert should land as an event beat: %v", w.Pending)
	}
	w.Pending = nil
	if j := engine.JournalText(w); !strings.Contains(j, "distraction") {
		t.Fatalf("the alert should surface in the journal: %q", j)
	}

	// Alerted (−2), retry still fails — and identically-numbered
	// attempts stay deterministic.
	if out := eng.Execute("sneak past hound"); !strings.Contains(out, "growl") {
		t.Fatalf("alerted retry should still fail: %q", out)
	}

	// Change the situation: the mug distraction (+4) beats the alert.
	if out := eng.Execute("knock espresso"); !strings.Contains(out, "detonates") {
		t.Fatalf("knocking should shatter the mug: %q", out)
	}
	out := eng.Execute("sneak past hound") // 10 −2 +4 = 12: clears
	if !strings.Contains(out, "skirting board") || w.Room().ID != "backroom" {
		t.Fatalf("distracted sneak should clear the hound: %q (room %s)", out, w.Room().ID)
	}
	if w.Flags["mug_shattered"] {
		t.Fatalf("the distraction should be spent by the roll")
	}
}

// The perception loop against real content: the lured hound stops
// watching the back door (docs/systems/stealth.md, observers), the
// prose and journal telegraph it, and the sneak rolls with the
// unwatched bonus. Perception reads flags — the lure is not spent.
func TestHoundLure(t *testing.T) {
	w := game.NewWorld()
	eng := engine.New(w)
	eng.Execute("north") // coffee shop
	w.Pending = nil

	// The lure lands via the barista's dialogue effects; here the flag
	// stands in for having asked her (dialogue mechanics have their
	// own tests).
	w.Flags["hound_lured"] = true

	if out := eng.Execute("examine hound"); !strings.Contains(out, "jerky") {
		t.Fatalf("the hound's prose should telegraph snack time: %q", out)
	}
	if j := engine.JournalText(w); !strings.Contains(j, "barely watched") {
		t.Fatalf("the lure should surface in the journal: %q", j)
	}

	// Unwatched +10: Stealth 10 rolls as 20 against difficulty 22.
	out := eng.Execute("sneak past hound")
	if !strings.Contains(out, "skirting board") || w.Room().ID != "backroom" {
		t.Fatalf("sneaking past the lured hound should clear: %q (room %s)", out, w.Room().ID)
	}
	if !strings.Contains(out, "+2 XP") {
		t.Fatalf("an unwatched pass should pay difficulty minus effective: %q", out)
	}
	if !w.Flags["hound_lured"] {
		t.Fatalf("perception must not consume the lure flag")
	}
}
