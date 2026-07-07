package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// The full whisper beat: flags set inside the terminal stay quiet
// until logout, then the event modal presents, then Buddy's notes have
// the entry. Exercises engine poll, shell deferral, modal, and notes.
func TestWhisperEventBeat(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)

	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "ssh sunfarm.arc")
	mod = typeLine(mod, "cat /var/log/burial.log")
	if !w.Flags["heard_whisper"] {
		t.Fatalf("flag should be set live, in the terminal")
	}
	if len(w.Pending) != 0 {
		t.Fatalf("rules must not evaluate while the shell is open: %v", w.Pending)
	}

	// Logout: the world reacts.
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if len(w.Pending) != 1 {
		t.Fatalf("logout should fire the whisper rule: %v", w.Pending)
	}
	view := mod.View()
	if !strings.Contains(view, "░▒▓") || !strings.Contains(view, "fans spin down") {
		t.Fatalf("event modal missing:\n%s", view)
	}

	// While the modal is up, keys belong to it — typing must not reach
	// the prompt.
	mod, _ = mod.Update(kr('x'))
	if v := mod.(Model).input.Value(); v != "" {
		t.Fatalf("typing leaked past the event modal: %q", v)
	}

	// Enter dismisses; the beat lands in the LOG.
	mod, _ = mod.Update(spec(tea.KeyEnter))
	mm := mod.(Model)
	if len(w.Pending) != 0 {
		t.Fatalf("dismissal should drain the beat")
	}
	if !strings.Contains(strings.Join(mm.entries, "\n"), "fans spin down") {
		t.Fatalf("dismissed beat should be in the LOG")
	}

	// Once: logging in and out again stays quiet.
	mod = typeLine(mod, "use deck")
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if len(w.Pending) != 0 {
		t.Fatalf("Once rule fired twice: %v", w.Pending)
	}

	// The notes file now carries the whisper entry.
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "cat ~/notes/notes.md")
	if reader := readerText(mod); !strings.Contains(reader, "sunfarm.arc") {
		t.Fatalf("notes should show the whisper entry:\n%s", reader)
	}
}

func TestNotesEmptyState(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "use deck")
	mod = typeLine(mod, "cat ~/notes/notes.md")
	if reader := readerText(mod); !strings.Contains(reader, "Nothing solid yet") {
		t.Fatalf("empty notes state missing:\n%s", reader)
	}
}
