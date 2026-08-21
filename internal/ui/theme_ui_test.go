package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestThemeCommandListsSelectsAndCycles(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	roomBefore, flagsBefore := w.Room().ID, len(w.Flags)

	mod = typeLine(mod, "theme list")
	mm := mod.(Model)
	if out := strings.Join(mm.shellEntries, "\n"); !strings.Contains(out, "chrome") || !strings.Contains(out, "* wet-neon") {
		t.Fatalf("theme list did not show registered/current themes: %q", out)
	}

	mod = typeLine(mod, "theme chrome")
	mm = mod.(Model)
	if mm.themeID != "chrome" {
		t.Fatalf("theme command selected %q, want chrome", mm.themeID)
	}
	if got, want := mm.shellInput.TextStyle.GetForeground(),
		mm.presentation().Terminal.local.bright; got != want {
		t.Fatalf("terminal input did not adopt chrome: got %v want %v", got, want)
	}

	mod = typeLine(mod, "theme previous")
	if got := mod.(Model).themeID; got != defaultThemeID {
		t.Fatalf("previous theme selected %q, want %q", got, defaultThemeID)
	}
	mod = typeLine(mod, "theme next")
	if got := mod.(Model).themeID; got != "chrome" {
		t.Fatalf("next theme selected %q, want chrome", got)
	}

	if w.Room().ID != roomBefore || len(w.Flags) != flagsBefore {
		t.Fatal("theme commands touched world state")
	}
	if out := strings.Join(mod.(Model).shellEntries, "\n"); strings.Contains(out, "command not found") {
		t.Fatalf("UI-local theme command leaked into fake shell: %q", out)
	}
}

func TestThemeCommandRejectsUnknownWithoutChangingTheme(t *testing.T) {
	mod := newSized(game.NewWorld())
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "theme missing")
	mm := mod.(Model)
	if mm.themeID != defaultThemeID {
		t.Fatalf("unknown theme changed selection to %q", mm.themeID)
	}
	if out := strings.Join(mm.shellEntries, "\n"); !strings.Contains(out, `unknown theme "missing"`) {
		t.Fatalf("unknown theme did not report available selection: %q", out)
	}
}

func TestThemeCommandCompletion(t *testing.T) {
	mod := newSized(game.NewWorld())
	mod = typeLine(mod, "use deck")
	mm := mod.(Model)

	mm.shellInput.SetValue("the")
	mod, _ = mm.Update(spec(tea.KeyTab))
	mm = mod.(Model)
	if got := mm.shellInput.Value(); got != "theme " {
		t.Fatalf("command completion got %q, want theme", got)
	}

	mm.shellInput.SetValue("theme c")
	mod, _ = mm.Update(spec(tea.KeyTab))
	if got := mod.(Model).shellInput.Value(); got != "theme chrome " {
		t.Fatalf("theme ID completion got %q, want chrome", got)
	}
}

func TestThemeIsNotInterceptedAsPassword(t *testing.T) {
	mod := newSized(game.NewWorld())
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh jane_doe@microslop")
	if !mod.(Model).shell.AwaitingPassword() {
		t.Fatal("test setup: expected password prompt")
	}

	mod = typeLine(mod, "theme chrome")
	mm := mod.(Model)
	if mm.themeID != defaultThemeID {
		t.Fatalf("password input changed theme to %q", mm.themeID)
	}
	if out := strings.Join(mm.shellEntries, "\n"); !strings.Contains(out, "Permission denied") {
		t.Fatalf("theme text was not handled as the pending password: %q", out)
	}
}

func TestShellHelpMentionsLocalThemeCommand(t *testing.T) {
	mod := newSized(game.NewWorld())
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "help")
	if out := strings.Join(mod.(Model).shellEntries, "\n"); !strings.Contains(out, themeUsage) {
		t.Fatalf("shell help omitted local theme command: %q", out)
	}
}
