package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestHackingScreenSwapsAndRestores(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	before := mod.View()
	if !strings.Contains(before, "THE CITY") {
		t.Fatalf("expected normal room UI before login")
	}

	mod = typeLine(mod, "log in") // rewrite -> "use deck" -> shell
	mm := mod.(Model)
	if mm.shell == nil {
		t.Fatalf("'log in' should open the hacking terminal")
	}
	view := mod.View()
	if !strings.Contains(view, "SESSION // deck") {
		t.Fatalf("terminal title missing: %q", view)
	}
	for _, section := range []string{"OBJECTIVE", "LOCATION", "DISCOVERIES", "STATUS"} {
		if !strings.Contains(view, section) {
			t.Fatalf("quest panel missing %s", section)
		}
	}
	if strings.Contains(view, "THE CITY") || strings.Contains(view, "LOG") {
		t.Fatalf("normal panels should be hidden while in the terminal")
	}

	// Output goes to the terminal scrollback, not the LOG.
	logLen := len(mm.entries)
	mod = typeLine(mod, "ls")
	mm = mod.(Model)
	joined := strings.Join(mm.shellEntries, "\n")
	if !strings.Contains(joined, "notes.txt") {
		t.Fatalf("ls output should land in shell scrollback: %q", joined)
	}
	if len(mm.entries) != logLen {
		t.Fatalf("shell commands must not touch the LOG")
	}

	// Esc closes the terminal outright.
	mod, _ = mod.Update(spec(tea.KeyEsc))
	mm = mod.(Model)
	if mm.shell != nil {
		t.Fatalf("esc should close the terminal")
	}
	if after := mod.View(); !strings.Contains(after, "THE CITY") {
		t.Fatalf("room UI should be restored after closing the terminal")
	}
}

func TestTerminalExitReturnsToRoom(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "exit") // at the deck, exit closes the terminal
	if mod.(Model).shell != nil {
		t.Fatalf("'exit' at the deck should close the terminal")
	}
	if !strings.Contains(mod.View(), "THE CITY") {
		t.Fatalf("room UI should be restored after exit")
	}
}

func TestDeckLoginFromPanel(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	// The deck sits in the lair, in a green UPLINK section of the left
	// panel — separated from the hubs.
	if view := mod.View(); !strings.Contains(view, "UPLINK") || !strings.Contains(view, "the deck") {
		t.Fatalf("deck should be listed in the left panel while in the lair:\n%s", view)
	}

	// Focus the panel, arrow to the deck (last item), Enter logs in.
	mod, _ = mod.Update(spec(tea.KeyTab))
	items := mod.(Model).panelItems()
	if items[len(items)-1].kind != panelDeck {
		t.Fatalf("deck should be the last focusable panel item")
	}
	for mod.(Model).selected != len(items)-1 {
		mod, _ = mod.Update(spec(tea.KeyDown))
	}
	mod, _ = mod.Update(spec(tea.KeyEnter))
	if mod.(Model).shell == nil {
		t.Fatalf("Enter on the deck row should log in")
	}
}

func TestDeckHiddenWhenNotInScope(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "north") // leave the lair for the coffee shop
	if view := mod.View(); strings.Contains(view, "UPLINK") {
		t.Fatalf("deck should not be reachable from another room:\n%s", view)
	}
	if items := mod.(Model).panelItems(); len(items) > 0 {
		for _, it := range items {
			if it.kind == panelDeck {
				t.Fatalf("no deck should be in scope from the coffee shop")
			}
		}
	}
}

func TestHackingBeatSetsFlags(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh sunfarm.arc")
	mod = typeLine(mod, "cat /var/log/burial.log")
	if !w.Flags["heard_whisper"] {
		t.Fatalf("reading burial.log should set heard_whisper; flags=%v", w.Flags)
	}
	mod = typeLine(mod, "cp /srv/archive/sun.frag ~/")
	if !w.Flags["got_sun_fragment"] {
		t.Fatalf("downloading sun.frag should set got_sun_fragment; flags=%v", w.Flags)
	}
	if view := mod.View(); !strings.Contains(view, "sun.frag") {
		t.Fatalf("DISCOVERIES should list the download")
	}
}

func TestHackingScreenNarrowTerminal(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod, _ = mod.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	// Must render without panicking and keep the terminal present.
	if view := mod.View(); !strings.Contains(view, "SESSION // deck") {
		t.Fatalf("narrow render lost the terminal: %q", view)
	}
}
