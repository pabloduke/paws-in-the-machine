package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// marduk's channel (docs/draft.md: Resistance-lite): the deck ships
// with the messenger, the mission brief is waiting on first login,
// and later messages arrive as story flags land — the mission-giver
// reacts to play, never to a clock.
func TestDeckMessengerMardukChannel(t *testing.T) {
	w := game.NewWorld()
	d, ok := engine.Part[hacking.Deck](w.FindID("deck"))
	if !ok {
		t.Fatalf("the deck should carry the hacking.Deck component")
	}
	mgr := d.Messenger
	if !mgr.Enabled() || mgr.Contact != "marduk" {
		t.Fatalf("the deck should carry marduk's channel: %+v", mgr)
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

// Mission 1 end to end from the deck's side (docs/draft.md): marduk's
// brief is waiting at boot, the objective chain and notes checklist
// advance flag by flag, the layoff plans read/copy/send hooks fire,
// and marduk's payoff lands the moment the drop takes the file. The
// overworld halves (barista, rack) are pinned by their own tests; here
// their flags are set directly.
func TestMissionOneLayoffPlans(t *testing.T) {
	w := game.NewWorld()
	d, _ := engine.Part[hacking.Deck](w.FindID("deck"))
	mgr := d.Messenger
	s := deckSession(t, w)

	notes := func() string {
		result := s.ExecDetailed("cat ~/notes/notes.md")
		if result.Document == nil {
			t.Fatalf("notes should open as a document")
		}
		return result.Document.Text
	}

	// The brief names the job; the objective and checklist point at
	// the barista, explicitly (mission 1 is the loud end of the
	// hint-fade).
	if thread := mgr.Thread(w); !strings.Contains(thread[0].Text, "layoff plans") {
		t.Fatalf("the brief should assign the layoff plans: %+v", thread)
	}
	if got := d.CurrentObjective(w); got != "find the barista's Microslop intel" {
		t.Fatalf("the first objective should point at the barista: %q", got)
	}
	if n := notes(); !strings.Contains(n, "[ ] talk to the barista") {
		t.Fatalf("the checklist should open all-unchecked: %q", n)
	}

	// The overworld work, as flags (barista dialogue, backroom rack).
	w.Flags["knows_microslop_password"] = true
	if got := d.CurrentObjective(w); got != "get the deck a route into Microslop" {
		t.Fatalf("the objective should advance to the route: %q", got)
	}
	w.Flags["microslop_route_open"] = true
	if got := d.CurrentObjective(w); got != "pull the layoff plans off Microslop" {
		t.Fatalf("the objective should advance to the plans: %q", got)
	}
	if n := notes(); !strings.Contains(n, "[x] talk to the barista") ||
		!strings.Contains(n, "[ ] connect to microslop") {
		t.Fatalf("the checklist should tick as flags land: %q", n)
	}

	// The terminal half for real: in, read, copy home, send.
	s.Exec("ssh microslop")
	s.Exec("apple")
	if out, _ := s.Exec("cat /var/log/access.log"); !strings.Contains(out, "/srv/hr/") {
		t.Fatalf("the access log should breadcrumb HR: %q", out)
	}
	s.ExecDetailed("cat /srv/hr/rif_q3.txt")
	if !w.Flags["read_layoff_plans"] {
		t.Fatalf("reading the plans should set the flag")
	}
	s.Exec("cp /srv/hr/rif_q3.txt ~/")
	if !w.Flags["got_layoff_plans"] {
		t.Fatalf("copying the plans home should set the flag")
	}
	if got := d.CurrentObjective(w); got != "send the plans to marduk" {
		t.Fatalf("the objective should advance to delivery: %q", got)
	}

	if out, _ := s.Exec("send ~/rif_q3.txt"); !strings.Contains(out, "uploading rif_q3.txt") {
		t.Fatalf("send should upload the plans: %q", out)
	}
	if !w.Flags["layoff_plans_delivered"] {
		t.Fatalf("sending the plans should set the delivered flag")
	}
	thread := mgr.Thread(w)
	if last := thread[len(thread)-1]; !strings.Contains(last.Text, "drop confirms") {
		t.Fatalf("marduk's payoff should arrive on delivery: %+v", last)
	}
	if n := notes(); !strings.Contains(n, "Mission complete") {
		t.Fatalf("the checklist should close out: %q", n)
	}
	if got := d.CurrentObjective(w); got != "trace the whisper in the dead code" {
		t.Fatalf("the objectives should fall through to the side thread: %q", got)
	}
}
