package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"strings"
	"testing"
)

func TestQuestConsoleAndTerminalEnable(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod, _ = mod.Update(tea.KeyMsg{Type: tea.KeyF12})
	if !mod.(Model).questConsole {
		t.Fatal("F12 did not open console")
	}
	mod = typeLine(mod, "sudo devmode --meow")
	if !w.DevMode || !strings.Contains(mod.View(), "DEV") {
		t.Fatal("developer mode not visible")
	}
	mod = typeLine(mod, "quest-debug")
	if !strings.Contains(mod.(Model).questConsoleText, "No authored quests") {
		t.Fatal("missing debugger output")
	}
	mod, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if mod.(Model).questConsole || !strings.Contains(mod.View(), "DEV") {
		t.Fatal("close lost dev state")
	}
	mod = typeLine(mod, "log in")
	if mod.(Model).shell == nil {
		t.Fatal("terminal not open")
	}
	mod = typeLine(mod, "sudo devmode --off")
	if w.DevMode {
		t.Fatal("terminal did not disable")
	}
	mod = typeLine(mod, "sudo devmode --meow")
	if !w.DevMode {
		t.Fatal("terminal did not enable")
	}
	mod = typeLine(mod, "help")
	if strings.Contains(strings.Join(mod.(Model).shellEntries[len(mod.(Model).shellEntries)-1:], ""), "devmode") {
		t.Fatal("hidden command in help")
	}
}
