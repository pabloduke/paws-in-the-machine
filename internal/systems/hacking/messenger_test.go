package hacking_test

import (
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

func testMessenger() hacking.Messenger {
	return hacking.Messenger{
		Contact: "resistance",
		Msgs: []hacking.Msg{
			{ID: "hello", Text: "channel's clean."},
			{ID: "later", When: "did_the_thing", Text: "we saw that. nice work."},
		},
	}
}

// The thread is derived from flags, the journal pattern: a message has
// arrived exactly when its When flag is true, so arrival is a
// consequence of play, never a timer, and save/load reconstructs the
// thread for free.
func TestMessengerThreadDerivesFromFlags(t *testing.T) {
	w := engine.NewWorld()
	mgr := testMessenger()

	if got := mgr.Thread(w); len(got) != 1 || got[0].Text != "channel's clean." {
		t.Fatalf("only the unconditioned message should have arrived: %+v", got)
	}
	if n := mgr.Unread(w); n != 1 {
		t.Fatalf("the opening message should count unread, got %d", n)
	}

	mgr.MarkRead(w)
	if n := mgr.Unread(w); n != 0 {
		t.Fatalf("MarkRead should clear the count, got %d", n)
	}

	w.Flags["did_the_thing"] = true
	if got := mgr.Thread(w); len(got) != 2 {
		t.Fatalf("the flag should deliver the second message: %+v", got)
	}
	if n := mgr.Unread(w); n != 1 {
		t.Fatalf("only the new arrival should be unread, got %d", n)
	}
}

// The zero Messenger means no service: nothing enabled, nothing
// arrived — a deck without a messenger stays a plain deck.
func TestMessengerZeroValueDisabled(t *testing.T) {
	w := engine.NewWorld()
	var mgr hacking.Messenger
	if mgr.Enabled() {
		t.Fatalf("the zero messenger must report disabled")
	}
	if got := mgr.Thread(w); len(got) != 0 {
		t.Fatalf("the zero messenger should have no thread: %+v", got)
	}
}

// The messenger command only signals; the data hangs off the Deck
// component and the UI owns the panel.
func TestMessengerCommandSignalsUI(t *testing.T) {
	w := engine.NewWorld()
	s, err := hacking.NewSession(w, testNet(), "deck")
	if err != nil {
		t.Fatalf("opening the session: %v", err)
	}
	result := s.ExecDetailed("messenger")
	if !result.Messenger {
		t.Fatalf("messenger should set the panel signal: %+v", result)
	}
	if result.Output != "" || result.Done {
		t.Fatalf("messenger should produce no output and not end the session: %+v", result)
	}
}
