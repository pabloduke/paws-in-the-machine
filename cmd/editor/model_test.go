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
	t.Helper()
	lipgloss.SetColorProfile(termenv.Ascii)
	path := filepath.Join(t.TempDir(), "charts.json")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	m, err := newModel(path)
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

	again, err := newModel(path)
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
