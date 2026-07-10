package hacking

import "github.com/pabloduke/paws-in-the-machine/internal/engine"

// Messenger is the deck's instant messenger: the right-panel chat the
// `messenger` command opens (docs/systems/hacking.md). One contact for
// now — the draft's mission-giver — but messages carry their sender in
// the data so more contacts can arrive later without a rewrite.
//
// The thread is derived, never stored (the journal pattern): a message
// has "arrived" exactly when its When flag is true, so arrival is a
// consequence of play — never a timer (docs/systems/turns.md) — and
// save/load gets the whole thread right for free. The only stored
// state is the per-message read marker.
type Messenger struct {
	Contact string // the one contact; empty means no messenger service
	Msgs    []Msg
}

// Msg is one authored message. When is the world flag that makes it
// arrive; empty means it has been in the thread from the start.
type Msg struct {
	ID   string // stable content id; the read-marker flag derives from it
	When string
	Text string
}

// readFlag is the read marker for one message. These flags are
// system-owned (docs/BOUNDARIES.md): derived here, written only by
// MarkRead, never referenced by content.
func readFlag(id string) string { return "msg_read_" + id }

// Enabled reports whether the deck has messenger service at all.
func (mgr Messenger) Enabled() bool { return mgr.Contact != "" }

// Thread returns the arrived messages in authored order.
func (mgr Messenger) Thread(w *engine.World) []Msg {
	var out []Msg
	for _, msg := range mgr.Msgs {
		if msg.When == "" || w.Flags[msg.When] {
			out = append(out, msg)
		}
	}
	return out
}

// Unread counts arrived messages not yet marked read.
func (mgr Messenger) Unread(w *engine.World) int {
	n := 0
	for _, msg := range mgr.Thread(w) {
		if !w.Flags[readFlag(msg.ID)] {
			n++
		}
	}
	return n
}

// MarkRead marks every arrived message read — opening the panel reads
// the whole thread.
func (mgr Messenger) MarkRead(w *engine.World) {
	for _, msg := range mgr.Thread(w) {
		w.Flags[readFlag(msg.ID)] = true
	}
}
