package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// Hacking pays XP (issue #24): terminal beats set flags in-session,
// and the payout lands at the checkpoint after logout — content-valued
// amounts, once each, through the same AwardXP funnel as everything.
func TestHackingBeatsPayXPOnceAtLogout(t *testing.T) {
	w := game.NewWorld()
	s := deckSession(t, w)

	s.Exec("ssh sunfarm.arc")
	if out, _ := s.Exec("cat /var/log/burial.log"); !strings.Contains(out, "buried the sun") {
		t.Fatalf("reading burial.log should work: %q", out)
	}
	if !w.Flags["heard_whisper"] {
		t.Fatalf("cat should fire the OnRead hook")
	}
	if w.XP != 0 {
		t.Fatalf("no XP while the session is open (rules don't run in-terminal), got %d", w.XP)
	}

	// Logout is the checkpoint (closeShell calls CheckEvents).
	w.CheckEvents()
	if w.XP != 3 {
		t.Fatalf("the whisper beat should pay 3 XP at logout, got %d", w.XP)
	}
	if joined := strings.Join(w.Pending, "\n"); !strings.Contains(joined, "+3 XP") {
		t.Fatalf("the payout should land as an event beat: %v", w.Pending)
	}
	w.Pending = nil

	// Once means once.
	w.CheckEvents()
	if w.XP != 3 || len(w.Pending) != 0 {
		t.Fatalf("the beat must not pay twice: XP=%d pending=%v", w.XP, w.Pending)
	}
}

// A bigger haul in one session pays each beat exactly once, and the
// awards flow through leveling like any other XP.
func TestHackingHaulLevelsUp(t *testing.T) {
	w := game.NewWorld()
	s := deckSession(t, w)

	s.Exec("ssh sunfarm.arc")
	s.Exec("cat /var/log/burial.log")           // heard_whisper: 3
	s.Exec("cp /srv/archive/sun.frag ~/notes/") // got_sun_fragment: 5
	s.Exec("run /srv/archive/dig.bin")          // ran_dig: 2
	w.CheckEvents()

	// 10 XP total: level 1→2 costs 5, leaving 5 toward level 3.
	if w.Level != 2 || w.XP != 5 {
		t.Fatalf("the haul should level Buddy up: level=%d xp=%d", w.Level, w.XP)
	}
	payouts := 0
	for _, p := range w.Pending {
		if strings.Contains(p, " XP") {
			payouts++
		}
	}
	if payouts != 3 {
		t.Fatalf("each beat should pay exactly once: %v", w.Pending)
	}
}
