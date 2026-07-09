package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// The PDA in the field: usable anywhere, menus only, and it shows the
// okuda.grid port flip without a shell in sight. use deck still
// refuses outside the lair — seeing is portable, touching is home.
func TestPDAMenusInTheField(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "north") // coffee shop

	mod = typeLine(mod, "use deck")
	if mod.(Model).shell != nil {
		t.Fatalf("the deck should not open outside the lair")
	}

	mod = typeLine(mod, "use pda")
	if mod.(Model).pdaMode != pdaMenu {
		t.Fatalf("use pda should open the PDA menu")
	}
	if view := mod.(Model).View(); !strings.Contains(view, "Check notes") || !strings.Contains(view, "Scan ports") {
		t.Fatalf("the menu should offer notes and scan:\n%s", view)
	}

	// Check notes → Buddy's notes render in the reader.
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if mod.(Model).pdaMode != pdaNotes {
		t.Fatalf("Check notes should open the file list")
	}
	if view := mod.(Model).View(); !strings.Contains(view, "~/notes/notes.md") {
		t.Fatalf("the list should show Buddy's notes:\n%s", view)
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if view := mod.(Model).View(); !strings.Contains(view, "Nothing solid yet") {
		t.Fatalf("reading notes should render their text:\n%s", view)
	}
	mod, _ = mod.Update(spec(tea.KeyEsc)) // reader → list
	mod, _ = mod.Update(spec(tea.KeyEsc)) // list → menu

	// Scan ports → okuda.grid reads closed; after the console flag,
	// the same screen reads open.
	mod, _ = mod.Update(spec(tea.KeyDown))
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if mod.(Model).pdaMode != pdaHosts {
		t.Fatalf("Scan ports should open the host list")
	}
	view := mod.(Model).View()
	if !strings.Contains(view, "okuda.grid") || !strings.Contains(view, "microslop") {
		t.Fatalf("the sniffer should list known hosts:\n%s", view)
	}
	mod, _ = mod.Update(spec(tea.KeyDown)) // microslop → okuda.grid
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if view := mod.(Model).View(); !strings.Contains(view, "SSH      | closed") {
		t.Fatalf("okuda.grid should report closed before the console:\n%s", view)
	}

	w.Flags["okuda_port_open"] = true     // the records-office console
	mod, _ = mod.Update(spec(tea.KeyEsc)) // report → hosts
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if view := mod.(Model).View(); !strings.Contains(view, "SSH      | open") {
		t.Fatalf("okuda.grid should report open after the console:\n%s", view)
	}

	// Esc all the way out pockets it and returns the prompt.
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if mod.(Model).pdaMode != pdaClosed {
		t.Fatalf("esc from the menu should pocket the PDA")
	}
	if joined := strings.Join(mod.(Model).entries, "\n"); !strings.Contains(joined, "pocket the PDA") {
		t.Fatalf("pocketing should land in the LOG: %q", joined)
	}
}
