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

// Mission 1 end to end from the deck's side (docs/draft.md): the
// mission file downloads into ~/notes when marduk's brief is read,
// its checklist advances flag by flag, the layoff plans read/copy/send
// hooks fire, and marduk's payoff lands the moment the drop takes the
// file. The overworld halves (barista, rack) are pinned by their own
// tests; here their flags are set directly.
func TestMissionOneLayoffPlans(t *testing.T) {
	w := game.NewWorld()
	d, _ := engine.Part[hacking.Deck](w.FindID("deck"))
	mgr := d.Messenger
	s := deckSession(t, w)

	const missionPath = "~/notes/Microslop_Find_The_Layoff_List.md"
	mission := func() string {
		result := s.ExecDetailed("cat " + missionPath)
		if result.Document == nil {
			t.Fatalf("the mission file should open as a document; got %q", result.Output)
		}
		return result.Document.Text
	}

	// The brief names the job and grants the mission.
	if thread := mgr.Thread(w); !strings.Contains(thread[0].Text, "layoff plans") {
		t.Fatalf("the brief should assign the layoff plans: %+v", thread)
	}
	// Before the brief is read, the mission hasn't downloaded — the
	// file isn't in ~/notes yet.
	if out, _ := s.Exec("ls ~/notes"); strings.Contains(out, "Microslop_Find") {
		t.Fatalf("the mission file should not exist before the brief is read: %q", out)
	}
	if r := s.ExecDetailed("cat " + missionPath); r.Document != nil {
		t.Fatalf("the mission file should not be readable before it's granted")
	}

	// Reading marduk's brief downloads the mission (Msg.Grants).
	mgr.MarkRead(w)
	if out, _ := s.Exec("ls ~/notes"); !strings.Contains(out, "Microslop_Find_The_Layoff_List.md") {
		t.Fatalf("reading the brief should drop the mission file in notes: %q", out)
	}
	if m := mission(); !strings.Contains(m, "[ ] talk to the barista") {
		t.Fatalf("the checklist should open all-unchecked: %q", m)
	}

	// The overworld work, as flags (barista dialogue, backroom rack).
	w.Flags["knows_microslop_password"] = true
	w.Flags["microslop_route_open"] = true
	if m := mission(); !strings.Contains(m, "[x] talk to the barista") ||
		!strings.Contains(m, "[ ] connect to microslop") {
		t.Fatalf("the checklist should tick as flags land: %q", m)
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
	if m := mission(); !strings.Contains(m, "Mission complete") {
		t.Fatalf("the mission file should close out: %q", m)
	}
}

// Mission 0 bootstraps the tutorial: the "open the messenger" note is
// present from the first boot and vanishes the moment the messenger is
// read, replaced by mission 1 (user ruling 2026-07-10).
func TestMissionZeroBootstrap(t *testing.T) {
	w := game.NewWorld()
	d, _ := engine.Part[hacking.Deck](w.FindID("deck"))
	s := deckSession(t, w)

	// At boot: mission 0 present, mission 1 not yet.
	if out, _ := s.Exec("ls ~/notes"); !strings.Contains(out, "use_the_messenger.md") ||
		strings.Contains(out, "Microslop_Find") {
		t.Fatalf("at boot only mission 0 should be in notes: %q", out)
	}
	if r := s.ExecDetailed("cat ~/notes/use_the_messenger.md"); r.Document == nil ||
		!strings.Contains(r.Document.Text, "open the messenger") {
		t.Fatalf("mission 0 should teach opening the messenger: %+v", r)
	}

	// Reading the brief: mission 0 gone, mission 1 arrived.
	d.Messenger.MarkRead(w)
	out, _ := s.Exec("ls ~/notes")
	if strings.Contains(out, "use_the_messenger.md") {
		t.Fatalf("mission 0 should vanish once the messenger is read: %q", out)
	}
	if !strings.Contains(out, "Microslop_Find_The_Layoff_List.md") {
		t.Fatalf("mission 1 should have downloaded: %q", out)
	}
	if r := s.ExecDetailed("cat ~/notes/use_the_messenger.md"); r.Document != nil {
		t.Fatalf("mission 0 should no longer be readable")
	}
}
