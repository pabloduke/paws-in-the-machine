package engine

import (
	"strings"
	"testing"
)

// eventWorld is a two-room fixture: a hub containing home and away,
// player starting in home.
func eventWorld() *World {
	w := NewWorld()
	hub := NewEntity("hub", "The Hub")
	home := NewEntity("home", "Home")
	away := NewEntity("away", "Away")
	w.Root.Add(hub)
	hub.Add(home, away)
	home.Add(w.Player)
	return w
}

func TestRuleFiresOnFlag(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Flags: []string{"x"},
		Do:    func(*World) string { return "fired" },
	}}
	w.CheckEvents()
	if len(w.Pending) != 0 {
		t.Fatalf("rule fired without its flag: %v", w.Pending)
	}
	w.Flags["x"] = true
	w.CheckEvents()
	if len(w.Pending) != 1 || w.Pending[0] != "fired" {
		t.Fatalf("rule should fire once flag is true: %v", w.Pending)
	}
}

func TestOnceFiresOnce(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Flags: []string{"x"},
		Once:  "x_seen",
		Do:    func(*World) string { return "fired" },
	}}
	w.Flags["x"] = true
	w.CheckEvents()
	w.CheckEvents()
	w.CheckEvents()
	if len(w.Pending) != 1 {
		t.Fatalf("Once rule fired %d times", len(w.Pending))
	}
	if !w.Flags["x_seen"] {
		t.Fatalf("Once flag should be set after firing")
	}
}

func TestUnlessSuppresses(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Flags:  []string{"x"},
		Unless: []string{"y"},
		Do:     func(*World) string { return "fired" },
	}}
	w.Flags["x"] = true
	w.Flags["y"] = true
	w.CheckEvents()
	if len(w.Pending) != 0 {
		t.Fatalf("Unless flag should suppress the rule: %v", w.Pending)
	}
}

func TestEnterFiresOnArrivalOnly(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Enter: "away",
		Once:  "arrived",
		Do:    func(*World) string { return "you arrive" },
	}}
	w.CheckEvents() // in home: no fire
	if len(w.Pending) != 0 {
		t.Fatalf("Enter rule fired in the wrong room: %v", w.Pending)
	}
	w.FindID("away").Add(w.Player)
	w.CheckEvents()
	if len(w.Pending) != 1 {
		t.Fatalf("Enter rule should fire on arrival: %v", w.Pending)
	}
}

func TestEnterMatchesHubAncestor(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Enter: "hub", // rooms sit inside it
		Once:  "arrived_hub",
		Do:    func(*World) string { return "hub reached" },
	}}
	w.FindID("away").Add(w.Player)
	w.CheckEvents()
	if len(w.Pending) != 1 {
		t.Fatalf("Enter should match an ancestor hub: %v", w.Pending)
	}
}

func TestEnterNeedsMovement(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Flags: []string{"x"},
		Enter: "home",
		Do:    func(*World) string { return "welcome back" },
	}}
	w.CheckEvents() // first poll: arrival in home, but flag false
	w.Flags["x"] = true
	w.CheckEvents() // still in home, no movement: must not fire
	if len(w.Pending) != 0 {
		t.Fatalf("Enter rule fired without movement: %v", w.Pending)
	}
	w.FindID("away").Add(w.Player)
	w.CheckEvents()
	w.FindID("home").Add(w.Player)
	w.CheckEvents() // re-entered home with flag true
	if len(w.Pending) != 1 {
		t.Fatalf("Enter rule should fire on re-arrival: %v", w.Pending)
	}
}

func TestExecuteRunsCheckpoint(t *testing.T) {
	w := eventWorld()
	w.Rules = []When{{
		Flags: []string{"x"},
		Once:  "x_seen",
		Do:    func(*World) string { return "fired" },
	}}
	w.Flags["x"] = true
	New(w).Execute("look")
	if len(w.Pending) != 1 {
		t.Fatalf("Execute should poll the rules: %v", w.Pending)
	}
}

func TestJournalDerivedFromFlags(t *testing.T) {
	w := eventWorld()
	w.Journal = []Entry{
		{Flag: "a", Text: "first"},
		{Flag: "b", Text: "second"},
	}
	if got := w.JournalEntries(); len(got) != 0 {
		t.Fatalf("journal should start empty: %v", got)
	}
	if !strings.Contains(JournalText(w), "empty") {
		t.Fatalf("empty journal text: %q", JournalText(w))
	}
	w.Flags["b"] = true
	got := w.JournalEntries()
	if len(got) != 1 || got[0] != "second" {
		t.Fatalf("journal should show entries whose flags are true: %v", got)
	}
	w.Flags["a"] = true
	got = w.JournalEntries()
	if len(got) != 2 || got[0] != "first" {
		t.Fatalf("journal keeps declaration order: %v", got)
	}
}
