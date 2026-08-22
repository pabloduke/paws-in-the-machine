package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const seed = `{
  "version": 1,
  "charts": [
    {
      "id": "neighborhood",
      "cells": {
        "0,0": "lair",
        "0,1": "coffeeshop"
      }
    }
  ]
}
`

func newTestModel(t *testing.T) (*model, string) {
	return newTestModelWithSeed(t, seed, testCatalog())
}

func testCatalog() catalog {
	return catalog{
		"lair":       {ID: "lair", Name: "Buddy's Lair", ParentID: "neighborhood"},
		"coffeeshop": {ID: "coffeeshop", Name: "The Coffee Shop", ParentID: "neighborhood"},
		"noodle_bar": {ID: "noodle_bar", Name: "(Placeholder)", ParentID: "neighborhood"},
	}
}

func newTestModelWithSeed(t *testing.T, contents string, entities entityCatalog) (*model, string) {
	t.Helper()
	lipgloss.SetColorProfile(termenv.Ascii)
	path := filepath.Join(t.TempDir(), "charts.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	m, err := newModel(path, entities)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m, path
}

func press(t *testing.T, m *model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		m.Update(msg)
	}
}

func typeText(t *testing.T, m *model, s string) {
	t.Helper()
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// The editor's reason to exist: place a room and immediately see the
// geometry the game will use.
func TestPlacingARoomDerivesExitsLive(t *testing.T) {
	m, _ := newTestModel(t)

	// Stand east of the lair — empty, so no exits yet.
	press(t, m, "l")
	if got := m.View(); !strings.Contains(got, "empty") {
		t.Fatalf("cell east of the lair should read empty:\n%s", got)
	}

	press(t, m, "n")
	typeText(t, m, "noodle_bar")
	press(t, m, "enter")

	view := m.View()
	if !strings.Contains(view, "west→lair") {
		t.Fatalf("the new room should derive an exit west to the lair:\n%s", view)
	}
	// And the lair gains the reciprocal, for free.
	press(t, m, "h")
	if view := m.View(); !strings.Contains(view, "east→noodle_bar") {
		t.Fatalf("the lair should now reach the new room east:\n%s", view)
	}
}

// Save writes the file the game embeds, and reloading sees the edit.
func TestSaveRoundTripsThroughDisk(t *testing.T) {
	m, path := newTestModel(t)
	press(t, m, "l", "n")
	typeText(t, m, "noodle_bar")
	press(t, m, "enter")

	if !m.dirty {
		t.Fatal("placing a room should mark the weave dirty")
	}
	press(t, m, "s")
	if m.dirty {
		t.Fatalf("saving should clear dirty; status was %q", m.status)
	}

	again, err := newModel(path, m.entities)
	if err != nil {
		t.Fatalf("reloading: %v", err)
	}
	if got, ok := again.chart().At(m.cur); !ok || got != "noodle_bar" {
		t.Fatalf("the saved file should hold the new room, got %q", got)
	}
}

// Quitting with unsaved work must not silently discard it.
func TestQuitWarnsWhenDirty(t *testing.T) {
	m, _ := newTestModel(t)
	press(t, m, "d") // delete the lair under the cursor
	if !m.dirty {
		t.Fatal("deleting should mark dirty")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatal("q must not quit while there are unsaved changes")
	}
	if !strings.Contains(m.status, "unsaved") {
		t.Fatalf("the warning should say so, got %q", m.status)
	}
}

// The fourth axis is authorable but never player-facing: standing one
// step ana shows an empty slice, not a new direction.
func TestAnaSliceIsSeparateSpace(t *testing.T) {
	m, _ := newTestModel(t)
	press(t, m, "]") // w+1
	if got := m.View(); !strings.Contains(got, "w=1") {
		t.Fatalf("the header should show the slice:\n%s", got)
	}
	if got := m.View(); !strings.Contains(got, "empty") {
		t.Fatalf("nothing is authored one step ana yet:\n%s", got)
	}
	// The room below is on the other slice and must not bleed through.
	if got := m.View(); strings.Contains(got, "coffeeshop") {
		t.Fatalf("slices must not leak into each other:\n%s", got)
	}
}

func TestEscapeCancelsNaming(t *testing.T) {
	m, _ := newTestModel(t)
	press(t, m, "l", "n")
	typeText(t, m, "mistake")
	press(t, m, "esc")
	if _, ok := m.chart().At(m.cur); ok {
		t.Fatal("escaping must not place anything")
	}
	if m.dirty {
		t.Fatal("a cancelled edit should leave the weave clean")
	}
}

func TestKnownEntityAcceptedFromInjectedCatalog(t *testing.T) {
	m, _ := newTestModel(t)
	press(t, m, "l", "n")
	typeText(t, m, "noodle_bar")
	press(t, m, "enter")

	if got, ok := m.chart().At(m.cur); !ok || got != "noodle_bar" {
		t.Fatalf("catalogued entity should be placed, got %q, ok=%v", got, ok)
	}
	if !m.dirty {
		t.Fatal("accepted placement should mark the weave dirty")
	}
}

func TestUnknownEntityPlacementIsRejected(t *testing.T) {
	m, _ := newTestModel(t)
	press(t, m, "l", "n")
	typeText(t, m, "not_in_the_game")
	press(t, m, "enter")

	if _, ok := m.chart().At(m.cur); ok {
		t.Fatal("unknown entity must not be placed")
	}
	if m.dirty {
		t.Fatal("rejected placement must leave the weave clean")
	}
	if !strings.Contains(m.status, `placement rejected: unknown entity "not_in_the_game"`) {
		t.Fatalf("unexpected rejection: %q", m.status)
	}
}

func TestDuplicateEntityPlacementIsRejected(t *testing.T) {
	m, _ := newTestModel(t)
	press(t, m, "l", "n")
	typeText(t, m, "lair")
	press(t, m, "enter")

	if _, ok := m.chart().At(m.cur); ok {
		t.Fatal("duplicate entity must not be placed")
	}
	if m.dirty {
		t.Fatal("rejected duplicate must leave the weave clean")
	}
	if !strings.Contains(m.status,
		`placement rejected: entity "lair" is already placed at neighborhood (0, 0)`) {
		t.Fatalf("unexpected rejection: %q", m.status)
	}
}

func TestDuplicatePlacementWithinChartIsReported(t *testing.T) {
	const duplicate = `{
  "version": 1,
  "charts": [
    {
      "id": "neighborhood",
      "cells": {
        "0,0": "lair",
        "1,0": "lair"
      }
    }
  ]
}
`
	m, _ := newTestModelWithSeed(t, duplicate, testCatalog())
	if !strings.Contains(m.status,
		`entity "lair" is placed more than once: neighborhood (0, 0), neighborhood (1, 0)`) {
		t.Fatalf("duplicate locations should be reported deterministically: %q", m.status)
	}
}

func TestDuplicatePlacementAcrossChartsIsReported(t *testing.T) {
	const duplicate = `{
  "version": 1,
  "charts": [
    {"id": "neighborhood", "cells": {"0,0": "lair"}},
    {"id": "plaza", "cells": {"2,3,1": "lair"}}
  ]
}
`
	m, _ := newTestModelWithSeed(t, duplicate, testCatalog())
	if !strings.Contains(m.status,
		`entity "lair" is placed more than once: neighborhood (0, 0), plaza (2, 3, 1)`) {
		t.Fatalf("cross-chart duplicate locations should be reported: %q", m.status)
	}
}

func TestInvalidSaveLeavesExistingFileUnchanged(t *testing.T) {
	const invalid = `{
  "version": 1,
  "charts": [
    {"id": "neighborhood", "cells": {"0,0": "missing_room"}}
  ]
}
`
	m, path := newTestModelWithSeed(t, invalid, testCatalog())
	if !strings.Contains(m.status,
		`validation failed: unknown entity "missing_room" at neighborhood (0, 0)`) {
		t.Fatalf("loaded unknown reference should be reported: %q", m.status)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	press(t, m, "s")
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("blocked save changed the file:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
	if !strings.Contains(m.status, `save blocked: unknown entity "missing_room"`) {
		t.Fatalf("blocked save should explain the invalid reference: %q", m.status)
	}
}

func TestDuplicateSaveLeavesExistingFileUnchanged(t *testing.T) {
	const invalid = `{
  "version": 1,
  "charts": [
    {"id": "neighborhood", "cells": {"0,0": "lair", "1,0": "lair"}}
  ]
}
`
	m, path := newTestModelWithSeed(t, invalid, testCatalog())
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	press(t, m, "s")
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("duplicate-blocked save changed the file:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
	if !strings.Contains(m.status, `save blocked: entity "lair" is placed more than once`) {
		t.Fatalf("duplicate save should be blocked: %q", m.status)
	}
}

func TestValidUnchangedSaveIsByteIdentical(t *testing.T) {
	m, path := newTestModel(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	press(t, m, "s")
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("valid unchanged save should be deterministic:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
}

func TestAlternateChartPathUsesAssembledWorldIdentity(t *testing.T) {
	entities, err := assembledGameCatalog()
	if err != nil {
		t.Fatalf("cataloguing assembled game: %v", err)
	}
	record, ok := entities.Lookup("lair")
	if !ok {
		t.Fatal("assembled catalog should contain the lair")
	}
	if record.Name != "Buddy's Lair" || record.ParentID != "neighborhood" {
		t.Fatalf("unexpected assembled identity: %+v", record)
	}

	m, path := newTestModelWithSeed(t, seed, entities)
	if path == defaultPath {
		t.Fatal("test should exercise an alternate chart path")
	}
	if strings.Contains(m.status, "validation failed") {
		t.Fatalf("alternate path should validate against assembled identity: %q", m.status)
	}
}
