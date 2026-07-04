package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// The full whisper beat: flags set inside the terminal stay quiet
// until logout, then the event modal presents, then the journal has
// the entry. Exercises engine poll, shell deferral, modal, journal.
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
	if !strings.Contains(view, "* * *") || !strings.Contains(view, "fans spin down") {
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

	// The journal now carries the whisper entry.
	mod = typeLine(mod, "journal")
	mm = mod.(Model)
	if !mm.journalOpen {
		t.Fatalf("'journal' should open the journal modal")
	}
	if v := mod.View(); !strings.Contains(v, "JOURNAL") || !strings.Contains(v, "sunfarm.arc") {
		t.Fatalf("journal should show the whisper entry:\n%s", v)
	}
	mod, _ = mod.Update(spec(tea.KeyEsc))
	if mod.(Model).journalOpen {
		t.Fatalf("esc should close the journal")
	}
}

func TestJournalEmptyState(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	mod = typeLine(mod, "journal")
	if v := mod.View(); !strings.Contains(v, "city keeps its secrets") {
		t.Fatalf("empty journal state missing:\n%s", v)
	}
}
