package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"strings"
	"testing"
)

func TestAuthoredBlockedPassages(t *testing.T) {
	c := authoredFixture()
	c.BlockedPassages.Passages = []content.BlockedPassage{{Kind: "location", AID: al, BID: al2}, {Kind: "room", AID: ar, BID: ar2}}
	r := AssembleAuthored(c)
	if !r.Ready() {
		t.Fatal(r.DiagnosticText())
	}
	w := r.World
	e := engine.New(w)
	for _, pair := range [][3]string{{"location", al, al2}, {"room", ar, ar2}} {
		a, b := w.FindID(authoredID(pair[0], pair[1])), w.FindID(authoredID(pair[0], pair[2]))
		for _, from := range []*engine.Entity{a, b} {
			ex, _ := engine.Part[engine.Exits](from)
			for _, dest := range ex.Dirs {
				if dest == a.ID || dest == b.ID {
					t.Fatal("wall was not reciprocal")
				}
			}
		}
	}
	start := w.Room()
	e.Execute("east")
	if w.Room() != start {
		t.Fatal("walked through wall")
	}
	e.Execute("enter")
	if w.Room().ID != authoredID("room", ar) {
		t.Fatal("wall removed interior entrance")
	}
	e.Execute("north")
	if w.Room().ID != authoredID("room", ar) {
		t.Fatal("walked through room wall")
	}
	e.Execute("out")
	if w.Room() != start {
		t.Fatal("interior return missing")
	}
	c.BlockedPassages = content.EmptyBlockedPassages()
	r = AssembleAuthored(c)
	e = engine.New(r.World)
	e.Execute("east")
	if r.World.Room().ID != authoredID("location", al2) {
		t.Fatal("reopen failed")
	}
	e.Execute("west")
	if r.World.Room().ID != authoredID("location", al) {
		t.Fatal("reciprocal reopen failed")
	}
	c.BlockedPassages.Passages = []content.BlockedPassage{{Kind: "room", AID: ar, BID: ar2}}
	c.RoomPlacements.Placements[1].Z = 1
	r = AssembleAuthored(c)
	if r.Ready() || !strings.Contains(r.DiagnosticText(), "same parent and floor") {
		t.Fatal("invalid wall geometry accepted")
	}
}
