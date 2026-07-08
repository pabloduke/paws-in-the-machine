package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// deckSession opens a shell on Buddy's deck (home in the lair), the
// way the terminal UI does.
func deckSession(t *testing.T, w *engine.World) *hacking.Session {
	t.Helper()
	d, ok := engine.Part[hacking.Deck](w.FindID("deck"))
	if !ok {
		t.Fatalf("Buddy's deck should carry the hacking.Deck component")
	}
	s, err := hacking.NewSession(w, d.Net, d.Host)
	if err != nil {
		t.Fatalf("opening the deck session: %v", err)
	}
	return s
}

// The closed-ports loop (issue #16), charm route, end to end: the scan
// shows okuda.grid filtered → the front-desk charm fails and forks
// (suspicious, −2) → the flyer trick changes the circumstances (+6) →
// the retry clears → the records-office console opens the port → the
// same scan shows it open and ssh connects. The first overworld↔
// terminal handshake, driven from both sides.
func TestOkudaClosedPortsLoop(t *testing.T) {
	w := game.NewWorld()
	eng := engine.New(w)

	// The terminal side first: okuda.grid answers scans but its one
	// port is filtered, and ssh times out against it.
	s := deckSession(t, w)
	if out, _ := s.Exec("scan okuda.grid"); !strings.Contains(out, "all scanned ports filtered") {
		t.Fatalf("okuda.grid should scan filtered before the console: %q", out)
	}
	if out, _ := s.Exec("ssh okuda.grid"); !strings.Contains(out, "timed out") {
		t.Fatalf("ssh should time out against the filtered port: %q", out)
	}

	// Overworld: to the lobby.
	hubs.Travel(w, "okuda")
	eng.Execute("north")
	w.Pending = nil

	// Charm 8 fails at difficulty 20: the world forks.
	if out := eng.Execute("charm receptionist"); !strings.Contains(out, "phone") {
		t.Fatalf("expected the first charm to fail at Charm 8: %q", out)
	}
	if !w.Flags["receptionist_suspicious"] {
		t.Fatalf("failure should make the receptionist suspicious")
	}
	if len(w.Pending) != 1 || !strings.Contains(w.Pending[0], "chair creaks") {
		t.Fatalf("suspicion should land as an event beat: %v", w.Pending)
	}
	w.Pending = nil
	if j := engine.JournalText(w); !strings.Contains(j, "flyer") {
		t.Fatalf("the journal should point at the flyer angle: %q", j)
	}

	// Suspicious (−2), a bare retry still fails.
	if out := eng.Execute("charm receptionist"); !strings.Contains(out, "phone") {
		t.Fatalf("suspicious retry should still fail: %q", out)
	}

	// The angle: notice the flyer, become Noodle. The dialogue choice
	// sets the flag (dialogue mechanics have their own tests).
	if out := eng.Execute("examine flyer"); !strings.Contains(out, "NOODLE") {
		t.Fatalf("the flyer should introduce Noodle: %q", out)
	}
	w.Flags["played_lost_cat"] = true

	// 8 −2 +6 = 12 clears difficulty 20; award pays 20−12.
	out := eng.Execute("charm receptionist")
	if !strings.Contains(out, "lunch break") || w.Room().ID != "okuda_office" {
		t.Fatalf("the flyer-backed charm should reach the office: %q (room %s)", out, w.Room().ID)
	}
	if !strings.Contains(out, "+8 XP") {
		t.Fatalf("the pass should pay difficulty minus effective: %q", out)
	}

	// The handshake: the console opens the port for the deck.
	if out := eng.Execute("use console"); !strings.Contains(out, "rejoined the net") {
		t.Fatalf("the console should throw the port switch: %q", out)
	}
	if !w.Flags["okuda_port_open"] {
		t.Fatalf("the console should set the port flag")
	}
	if out, _ := s.Exec("scan okuda.grid"); !strings.Contains(out, "open") {
		t.Fatalf("the port should scan open once the switch is thrown: %q", out)
	}
	if out, _ := s.Exec("ssh okuda.grid"); !strings.Contains(out, "OKUDA GRID") {
		t.Fatalf("ssh should now connect and print the banner: %q", out)
	}
	if s.HostName() != "okuda.grid" {
		t.Fatalf("the session should be on okuda.grid, got %s", s.HostName())
	}
	if j := engine.JournalText(w); !strings.Contains(j, "answer the deck") {
		t.Fatalf("the open port should surface in the journal: %q", j)
	}
}

// The stealth/agility route: Buddy's build (Agility 12, Stealth 10)
// clears the fire escape and the corridor guard on the pinned seed —
// the heavily-guarded way in rewards the cat who took it.
func TestOkudaStealthRoute(t *testing.T) {
	w := game.NewWorld()
	eng := engine.New(w)
	hubs.Travel(w, "okuda")
	eng.Execute("east") // service alley
	w.Pending = nil

	// The fire escape only speaks parkour.
	if out := eng.Execute("charm fire escape"); !strings.Contains(out, "remains a fire escape") {
		t.Fatalf("charming the fire escape should refuse: %q", out)
	}
	// "jump ladder": both the verb (jump → parkour) and the noun
	// ("ladder" aliases the fire escape) are what players reach for.
	out := eng.Execute("jump ladder") // Agility 12 vs 20: clears
	if !strings.Contains(out, "painted shut") || w.Room().ID != "okuda_corridor" {
		t.Fatalf("the climb should reach the corridor: %q (room %s)", out, w.Room().ID)
	}

	// The guard is immune to cute; the ceiling offers no route.
	if out := eng.Execute("charm guard"); !strings.Contains(out, "immune") {
		t.Fatalf("charming the guard should refuse: %q", out)
	}
	out = eng.Execute("sneak past guard") // Stealth 10 vs 22: clears
	if !strings.Contains(out, "touches nothing") || w.Room().ID != "okuda_office" {
		t.Fatalf("the sneak should reach the office: %q (room %s)", out, w.Room().ID)
	}
	if !strings.Contains(out, "+12 XP") {
		t.Fatalf("the pass should pay difficulty minus effective: %q", out)
	}
}

// The breaker is the corridor's mug: knocking the lights out is a +6
// circumstance the sneak roll consumes (the guard resets the panel).
func TestOkudaBreakerIsConsumed(t *testing.T) {
	w := game.NewWorld()
	eng := engine.New(w)
	hubs.Travel(w, "okuda")
	eng.Execute("east")
	// "climb" is what players actually type; the parser aliases it
	// (with "jump") to the parkour approach.
	eng.Execute("climb fire escape")
	w.Pending = nil

	if out := eng.Execute("knock breaker"); !strings.Contains(out, "gone dark") {
		t.Fatalf("knocking the breaker should kill the lights: %q", out)
	}
	out := eng.Execute("sneak past guard") // 10 +6 = 16 vs 22: clears
	if !strings.Contains(out, "touches nothing") || w.Room().ID != "okuda_office" {
		t.Fatalf("the dark sneak should clear the guard: %q (room %s)", out, w.Room().ID)
	}
	if w.Flags["corridor_lights_out"] {
		t.Fatalf("the darkness should be spent by the roll")
	}
}

// The door contract, pointed the other way (#25): the archive door in
// the records office answers only okuda.grid. Walk the whole loop —
// console opens the port, ssh in, run the maglock release, and the
// overworld door that was shut is now a doorway.
func TestOkudaArchiveDoorOpensFromTheNet(t *testing.T) {
	w := game.NewWorld()
	eng := engine.New(w)
	hubs.Travel(w, "okuda")
	eng.Execute("east")
	eng.Execute("climb fire escape")
	eng.Execute("sneak past guard")
	if w.Room().ID != "okuda_office" {
		t.Fatalf("setup should reach the office, got %s", w.Room().ID)
	}
	w.Pending = nil

	// Shut until the net says otherwise.
	if out := eng.Execute("east"); !strings.Contains(out, "maglock") || w.Room().ID != "okuda_office" {
		t.Fatalf("the archive door should hold: %q (room %s)", out, w.Room().ID)
	}

	eng.Execute("use console")
	s := deckSession(t, w)
	if out, _ := s.Exec("ssh okuda.grid"); !strings.Contains(out, "OKUDA GRID") {
		t.Fatalf("ssh should connect with the port open: %q", out)
	}
	if out, _ := s.Exec("run /srv/ctl/unlock.bin"); !strings.Contains(out, "release... ok") {
		t.Fatalf("unlock.bin should release the maglock: %q", out)
	}
	if !w.Flags["okuda_annex_unlocked"] {
		t.Fatalf("running unlock.bin should set the door flag")
	}

	eng.Execute("east")
	if w.Room().ID != "okuda_annex" {
		t.Fatalf("the archive door should be open now, got room %s", w.Room().ID)
	}
	if j := engine.JournalText(w); !strings.Contains(j, "maglock") {
		t.Fatalf("the release should surface in the journal: %q", j)
	}
}
