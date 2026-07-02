package engine

// Component is a composable behavior attached to an entity.
//
// Handle is offered every command whose target is the owning entity.
// Returning handled=true consumes the command; handled=false passes it
// to the next component and finally to the engine default for the verb.
// A component that wants to customize a verb but keep its default
// effect calls the exported engine helper (Take, Go, ...) explicitly.
//
// Discipline rule for content: components hold configuration, not
// mutable state. Anything that changes during play belongs in
// World.Flags or in the shape of the entity tree. This is what keeps
// future save/load and data-file loading cheap.
type Component interface {
	Handle(w *World, self *Entity, cmd Command) (out string, handled bool)
}

// Part returns the first component of type T attached to e.
func Part[T Component](e *Entity) (T, bool) {
	for _, p := range e.Parts {
		if t, ok := p.(T); ok {
			return t, true
		}
	}
	var zero T
	return zero, false
}

// --- Stock components -------------------------------------------------

// Description gives an entity examine text. Set Fn instead of Text for
// state-dependent descriptions.
type Description struct {
	Text string
	Fn   func(w *World) string
}

func (d Description) render(w *World) string {
	if d.Fn != nil {
		return d.Fn(w)
	}
	return d.Text
}

func (d Description) Handle(w *World, self *Entity, cmd Command) (string, bool) {
	if cmd.Verb == "examine" {
		return d.render(w), true
	}
	return "", false
}

// Portable marks an entity as takeable/droppable. It handles no verbs
// itself; the default take/drop logic checks for its presence.
type Portable struct{}

func (Portable) Handle(*World, *Entity, Command) (string, bool) { return "", false }

// On adapts a closure to a Component for a single verb — the escape
// hatch for bespoke puzzle logic. Every use of On is an inventory item
// for a future declarative effect vocabulary, so prefer reusable
// components once a behavior repeats.
type On struct {
	Verb string
	Do   func(w *World) string
}

func (o On) Handle(w *World, self *Entity, cmd Command) (string, bool) {
	if cmd.Verb == o.Verb {
		return o.Do(w), true
	}
	return "", false
}

// Exits makes an entity a room: Dirs maps directions to destination
// entity IDs. Blocked, if set, replaces the stock "you can't go that
// way" message.
type Exits struct {
	Dirs    map[string]string
	Blocked string
}

func (Exits) Handle(*World, *Entity, Command) (string, bool) { return "", false }
