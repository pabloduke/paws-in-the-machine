package charts

import (
	"strings"
	"testing"
)

func hasBug(bugs []string, substrings ...string) bool {
	for _, bug := range bugs {
		matched := true
		for _, want := range substrings {
			if !strings.Contains(bug, want) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// Reciprocity is the whole promise: declaring a passage declares the
// way back. A second gluing that claims a face already spoken for would
// rewrite the first gluing's return edge, leaving `A east -> B` while
// `B west` now goes to C — a scrambled edge, which the system says is
// not representable.
func TestConflictingGluingIsRejectedRatherThanSilentlyRewriting(t *testing.T) {
	a := New("a", map[Coord]string{{}: "a0"})
	b := New("b", map[Coord]string{{}: "b0"})
	c := New("c", map[Coord]string{{}: "c0"})
	w := NewWeave(a, b, c)

	if bugs := w.Glue("a", Gluing{From: Coord{}, Dir: "east", To: Coord{}, Chart: "b"}); len(bugs) != 0 {
		t.Fatalf("first gluing rejected: %v", bugs)
	}
	// c east -> b would overwrite the reverse face b west -> a.
	bugs := w.Glue("c", Gluing{From: Coord{}, Dir: "east", To: Coord{}, Chart: "b"})
	if len(bugs) == 0 {
		t.Fatal("a gluing claiming an already-installed reverse face was accepted")
	}
	if !hasBug(bugs, "b") {
		t.Errorf("conflict report does not identify the claimed face: %v", bugs)
	}

	// The first gluing must survive intact, both ways.
	if got := w.Exits("a", Coord{})["east"]; got != "b0" {
		t.Errorf("a east = %q, want b0", got)
	}
	if got := w.Exits("b", Coord{})["west"]; got != "a0" {
		t.Errorf("b west = %q, want a0 — the reverse edge was rewritten", got)
	}
	// The rejected declaration must leave nothing behind.
	if got := w.Exits("c", Coord{})["east"]; got != "" {
		t.Errorf("c east = %q, want no exit from a rejected gluing", got)
	}
	_, declared := w.Gluings()
	if len(declared) != 1 {
		t.Errorf("declared gluings = %d, want 1", len(declared))
	}
}

func TestConflictingForwardFaceIsRejected(t *testing.T) {
	a := New("a", map[Coord]string{{}: "a0"})
	b := New("b", map[Coord]string{{}: "b0"})
	c := New("c", map[Coord]string{{}: "c0"})
	w := NewWeave(a, b, c)

	if bugs := w.Glue("a", Gluing{From: Coord{}, Dir: "east", To: Coord{}, Chart: "b"}); len(bugs) != 0 {
		t.Fatalf("first gluing rejected: %v", bugs)
	}
	// The same forward face cannot lead to two places.
	if bugs := w.Glue("a", Gluing{From: Coord{}, Dir: "east", To: Coord{}, Chart: "c"}); len(bugs) == 0 {
		t.Fatal("a second gluing on the same forward face was accepted")
	}
	if got := w.Exits("a", Coord{})["east"]; got != "b0" {
		t.Errorf("a east = %q, want b0", got)
	}
}

// Re-declaring exactly the same gluing is not a conflict: it asks for
// what is already true.
func TestIdenticalGluingIsNotAConflict(t *testing.T) {
	a := New("a", map[Coord]string{{}: "a0"})
	b := New("b", map[Coord]string{{}: "b0"})
	w := NewWeave(a, b)
	g := Gluing{From: Coord{}, Dir: "east", To: Coord{}, Chart: "b"}
	if bugs := w.Glue("a", g); len(bugs) != 0 {
		t.Fatalf("first: %v", bugs)
	}
	if bugs := w.Glue("a", g); len(bugs) != 0 {
		t.Fatalf("re-declaring the same gluing was reported as a conflict: %v", bugs)
	}
}

// Two charts with one ID silently replaced each other, and Apply
// iterates maps, so which geometry won was nondeterministic.
func TestDuplicateChartIDIsReported(t *testing.T) {
	_, bugs, err := Unmarshal([]byte(`{"version":1,"charts":[
		{"id":"dup","cells":{"0,0":"one"}},
		{"id":"dup","cells":{"0,0":"two"}}]}`))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !hasBug(bugs, "dup") {
		t.Fatalf("duplicate chart ID was accepted silently: %v", bugs)
	}
}

// One entity in two cells means Apply rewrites its exits twice, in map
// order, so the room's exits depend on iteration luck.
func TestEntityInTwoCellsIsReported(t *testing.T) {
	_, bugs, err := Unmarshal([]byte(`{"version":1,"charts":[
		{"id":"a","cells":{"0,0":"room","1,0":"room"}}]}`))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !hasBug(bugs, "room") {
		t.Fatalf("an entity placed in two cells was accepted: %v", bugs)
	}
}

func TestEntityInTwoChartsIsReported(t *testing.T) {
	_, bugs, err := Unmarshal([]byte(`{"version":1,"charts":[
		{"id":"a","cells":{"0,0":"room"}},
		{"id":"b","cells":{"0,0":"room"}}]}`))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !hasBug(bugs, "room") {
		t.Fatalf("an entity placed in two charts was accepted: %v", bugs)
	}
}

func TestEmptyChartOrEntityIDIsReported(t *testing.T) {
	_, bugs, err := Unmarshal([]byte(`{"version":1,"charts":[
		{"id":"","cells":{"0,0":"room"}},
		{"id":"b","cells":{"0,0":""}}]}`))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(bugs) < 2 {
		t.Fatalf("empty chart and entity IDs were accepted: %v", bugs)
	}
}

// A gluing to an empty cell suppressed the ordinary adjacency and
// produced no exit, silently deleting a passage.
func TestGluingToAnEmptyCellIsReported(t *testing.T) {
	a := New("a", map[Coord]string{{}: "a0", {X: 1}: "a1"})
	b := New("b", map[Coord]string{{}: "b0"})
	w := NewWeave(a, b)
	bugs := w.Glue("a", Gluing{From: Coord{}, Dir: "east", To: Coord{Y: 9}, Chart: "b"})
	if len(bugs) == 0 {
		t.Fatal("a gluing pointing at an unoccupied cell was accepted")
	}
	// The rejected gluing must not have eaten the ordinary neighbour.
	if got := w.Exits("a", Coord{})["east"]; got != "a1" {
		t.Errorf("a east = %q, want the derived neighbour a1", got)
	}
}

func TestGluingFromAnEmptyCellIsReported(t *testing.T) {
	a := New("a", map[Coord]string{{}: "a0"})
	b := New("b", map[Coord]string{{}: "b0"})
	w := NewWeave(a, b)
	if bugs := w.Glue("a", Gluing{From: Coord{Y: 9}, Dir: "east", To: Coord{}, Chart: "b"}); len(bugs) == 0 {
		t.Fatal("a gluing from an unoccupied cell was accepted")
	}
}
