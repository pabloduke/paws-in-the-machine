package game_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// pdaCfg pulls the PDA component off the carried slab.
func pdaCfg(t *testing.T, w *engine.World) hacking.PDA {
	t.Helper()
	p, ok := engine.Part[hacking.PDA](w.FindID("pda"))
	if !ok {
		t.Fatalf("the pda entity should carry the hacking.PDA component")
	}
	return p
}

// The PDA travels; the deck doesn't. Buddy carries the slab
// everywhere while the deck stays lair-only (deck_test.go).
func TestPDACarriedEverywhere(t *testing.T) {
	w := game.NewWorld()
	if w.InScope("pda") == nil {
		t.Fatalf("the PDA should be in scope in the lair")
	}
	hubs.Travel(w, "okuda")
	if w.InScope("pda") == nil {
		t.Fatalf("the PDA should be in scope at Okuda")
	}
	if len(hacking.DecksInScope(w)) != 0 {
		t.Fatalf("the PDA must not count as a loggable deck")
	}
}

// The field-intel loop the PDA exists for: notes are readable, the
// sniffer sees okuda.grid closed, the office console flips it open —
// and the PDA can see that from anywhere, but can never act on it.
func TestPDAFieldIntelLoop(t *testing.T) {
	w := game.NewWorld()
	p := pdaCfg(t, w)

	// The deck mirror shows the field notes and mission 0 (the bootstrap
	// note, present from the first boot until the messenger is opened).
	files := hacking.TextFiles(w, p.Net[p.Host])
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	if strings.Join(paths, ",") != "~/notes/notes.md,~/notes/use_the_messenger.md" {
		t.Fatalf("the deck mirror should list Buddy's notes and mission 0: %+v", files)
	}
	if doc := files[0].Read(w); !strings.Contains(doc.Text, "Nothing solid yet") {
		t.Fatalf("reading notes should render the dynamic file: %q", doc.Text)
	}

	hosts := hacking.KnownHosts(p.Net, p.Host)
	want := []string{"microslop", "okuda.grid", "sunfarm.arc", "undernet.relay"}
	if strings.Join(hosts, ",") != strings.Join(want, ",") {
		t.Fatalf("known hosts should be every net host but the deck: %v", hosts)
	}

	if out := hacking.PortReport(w, p.Net, "okuda.grid"); !strings.Contains(out, "22    SSH      | closed") {
		t.Fatalf("okuda.grid should sniff closed before the console: %q", out)
	}
	w.Flags["okuda_port_open"] = true // the records-office console (okuda.go)
	if out := hacking.PortReport(w, p.Net, "okuda.grid"); !strings.Contains(out, "22    SSH      | open") {
		t.Fatalf("okuda.grid should sniff open after the console: %q", out)
	}
}

// The PDA mirrors the deck's storage live: a file copied home in a
// deck session shows up in the PDA's notes list, because both devices
// share one net.
func TestPDASeesCopiedFiles(t *testing.T) {
	w := game.NewWorld()
	p := pdaCfg(t, w)

	s, err := hacking.NewSession(w, p.Net, p.Host)
	if err != nil {
		t.Fatalf("opening the deck session: %v", err)
	}
	s.Exec("ssh microslop")
	s.Exec("apple")
	if out, _ := s.Exec("cp /srv/archive/sun_notice.txt ~/notes/"); out != "" && strings.Contains(out, "cp:") {
		t.Fatalf("copying the notice home should work: %q", out)
	}

	files := hacking.TextFiles(w, p.Net[p.Host])
	found := false
	for _, f := range files {
		if f.Path == "~/notes/sun_notice.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the PDA should see the copied notice: %+v", files)
	}
}
