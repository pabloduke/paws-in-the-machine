package engine

// Events & triggers (docs/systems/events.md): rules that make the
// world react to game state. A rule is conditions over the World
// (flags, arrival) plus an effect. Rules are polled — evaluated at
// the end of every player action, never notified — so they judge
// state as it is, however it got that way. They live in engine core
// because every action path (Execute, hubs.Travel, dialogue picks,
// leaving the hacking terminal) funnels through CheckEvents, and
// systems may not import a sibling system.

// When is one event rule. Zero-value fields are unconstrained: a rule
// with only Flags fires after any action once they hold.
type When struct {
	Flags  []string // all must be true
	Unless []string // all must be false
	Enter  string   // room or hub ID; fires only on arriving there ("" = any action)
	Once   string   // flag set when fired; must be false to fire ("" = repeats)
	Do     func(w *World) string
}

// Entry is one journal entry, visible once its flag is true. The
// journal is derived, never stored: flags are its only persistence.
type Entry struct {
	Flag string
	Text string
}

// CheckEvents is the end-of-turn poll: every rule whose conditions
// hold fires, in declaration order; output queues on Pending for the
// UI to present. Every path that completes a player action calls it.
func (w *World) CheckEvents() {
	defer w.EvaluateQuests()
	if !w.HoldPlacements {
		w.applyPlacements()
	}
	room := w.Room()
	entered := room != w.prevRoom
	w.prevRoom = room
	for _, r := range w.Rules {
		if r.Once != "" && w.Flags[r.Once] {
			continue
		}
		if r.Enter != "" && (!entered || !within(room, r.Enter)) {
			continue
		}
		if !flagsAre(w, r.Flags, true) || !flagsAre(w, r.Unless, false) {
			continue
		}
		if r.Once != "" {
			w.Flags[r.Once] = true
		}
		if r.Do != nil {
			if out := r.Do(w); out != "" {
				w.Pending = append(w.Pending, out)
			}
		}
	}
}

// within reports whether room is, or sits inside, the entity with the
// given ID — so Enter can name a room or its hub.
func within(room *Entity, id string) bool {
	for e := room; e != nil; e = e.Parent {
		if e.ID == id {
			return true
		}
	}
	return false
}

// flagsAre reports whether every named flag has the wanted value.
func flagsAre(w *World, flags []string, want bool) bool {
	for _, f := range flags {
		if w.Flags[f] != want {
			return false
		}
	}
	return true
}

// JournalEntries returns the visible journal: every declared entry
// whose flag is true, in declaration order. Recomputed on every call.
func (w *World) JournalEntries() []string {
	var out []string
	for _, e := range w.Journal {
		if w.Flags[e.Flag] {
			out = append(out, e.Text)
		}
	}
	return out
}

// JournalText renders the journal for the transcript (headless path;
// the UI shows the same entries in a modal).
func JournalText(w *World) string {
	entries := w.JournalEntries()
	if len(entries) == 0 {
		return "The journal is empty. The city keeps its secrets, for now."
	}
	out := "JOURNAL"
	for _, e := range entries {
		out += "\n  · " + e
	}
	return out
}
