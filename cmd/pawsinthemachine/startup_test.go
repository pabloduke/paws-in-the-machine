package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pabloduke/paws-in-the-machine/internal/ui"
)

func writePickerWorld(t *testing.T, dir string) {
	t.Helper()
	hub := "00000000-0000-4000-8000-000000000001"
	loc := "00000000-0000-4000-8000-000000000002"
	for name, data := range map[string]string{
		"hubs.json":                 `{"version":1,"hubs":[{"id":"` + hub + `","name":"Test Hub"}]}`,
		"locations.json":            `{"version":1,"locations":[{"id":"` + loc + `","name":"Test Cell","description":"(Placeholder)"}]}`,
		"location_assignments.json": `{"version":1,"assignments":[{"location_id":"` + loc + `","hub_id":"` + hub + `"}]}`,
		"location_placements.json":  `{"version":1,"placements":[{"location_id":"` + loc + `","hub_id":"` + hub + `","x":0,"y":0,"z":0,"w":0}]}`,
		"play_settings.json":        `{"version":1,"start":{"kind":"location","id":"` + loc + `"},"hub_arrivals":{"` + hub + `":"` + loc + `"}}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func pickerKey(m startupModel, key tea.KeyType) startupModel {
	next, _ := m.Update(tea.KeyMsg{Type: key})
	return next.(startupModel)
}
func TestPickerRetriesAndStartsAuthoredSession(t *testing.T) {
	dir := t.TempDir()
	m := newStartup([]worldChoice{{Name: "Test World", Dir: dir}}, nil)
	m = pickerKey(m, tea.KeyEnter)
	if !m.reviewing || m.pending.Ready() || m.session != nil || !strings.Contains(m.View(), "Not ready") {
		t.Fatal("invalid world launched")
	}
	writePickerWorld(t, dir)
	m = pickerKey(m, tea.KeyEnter)
	if !m.pending.Ready() || m.session != nil {
		t.Fatal("retry did not prepare fresh world")
	}
	first := m.pending.World
	first.Flags["test_only"] = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = next.(startupModel)
	if m.pending.World == first || m.pending.World.Flags["test_only"] {
		t.Fatal("reload reused running state")
	}
	m = pickerKey(m, tea.KeyEnter)
	session, ok := m.session.(ui.Model)
	if !ok || !session.Playtest || session.SavePath != "" {
		t.Fatal("authored session not isolated")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("look")})
	m = next.(startupModel)
	m = pickerKey(m, tea.KeyEnter)
	if !strings.Contains(m.View(), "> look") {
		t.Fatal("session does not accept prompt input")
	}
	if !strings.Contains(m.View(), "TEST CELL") {
		t.Fatal("authored session did not start in overworld")
	}
}
func TestPickerBuiltInAndNavigation(t *testing.T) {
	m := newStartup([]worldChoice{{Name: "Built-in game", BuiltIn: true}, {Name: "Other", Dir: t.TempDir()}}, nil)
	m = pickerKey(m, tea.KeyDown)
	if m.selected != 1 {
		t.Fatal("down failed")
	}
	m = pickerKey(m, tea.KeyEnter)
	m = pickerKey(m, tea.KeyEsc)
	if m.reviewing || m.pending.World != nil {
		t.Fatal("back did not clear pending world")
	}
	m = pickerKey(m, tea.KeyUp)
	m = pickerKey(m, tea.KeyEnter)
	session, ok := m.session.(ui.Model)
	if !ok || session.Playtest {
		t.Fatal("built-in game did not launch normally")
	}
	// Built-in startup still opens the deck rather than the overworld room view.
	if !strings.Contains(m.View(), "CYBERDECK // DECK") {
		t.Fatal("built-in startup did not open deck")
	}
}
func TestWorldDiscovery(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zeta", "Alpha", "main", "has space"} {
		if err := os.Mkdir(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "stray"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	choices, err := discoverWorlds("", root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range choices {
		names = append(names, c.Name)
	}
	if strings.Join(names, ",") != "Built-in game,main (editor),Alpha,zeta" {
		t.Fatal(names)
	}
	pinned, err := discoverWorlds(filepath.Join(root, "Alpha"), root)
	if err != nil || len(pinned) != 1 || pinned[0].BuiltIn || pinned[0].Dir != filepath.Join(root, "Alpha") {
		t.Fatal(pinned, err)
	}
}
