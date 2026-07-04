package hubs

import "github.com/pabloduke/paws-in-the-machine/internal/engine"

// Hub marks an entity as a city district. It handles no verbs — it is
// a structural marker with configuration, like engine.Exits.
type Hub struct {
	// Entry is the room ID where travel to this hub lands.
	Entry string
}

func (Hub) Handle(*engine.World, *engine.Entity, engine.Command) (string, bool) {
	return "", false
}

// List returns every hub: direct children of the root carrying Hub,
// in tree order.
func List(w *engine.World) []*engine.Entity {
	var out []*engine.Entity
	for _, e := range w.Root.Contents {
		if _, ok := engine.Part[Hub](e); ok {
			out = append(out, e)
		}
	}
	return out
}

// Current returns the hub containing the player, or nil (e.g. in test
// fixtures whose rooms hang directly off the root).
func Current(w *engine.World) *engine.Entity {
	for e := w.Player.Parent; e != nil; e = e.Parent {
		if _, ok := engine.Part[Hub](e); ok {
			return e
		}
	}
	return nil
}

// Travel moves the player to the hub's entry room. Like engine.Go it
// returns no description — the UI renders the current room from state.
// Unknown hub IDs and broken entry references are content bugs; Travel
// reports them rather than panicking.
func Travel(w *engine.World, hubID string) string {
	for _, h := range List(w) {
		if h.ID != hubID {
			continue
		}
		cfg, _ := engine.Part[Hub](h)
		dest := w.FindID(cfg.Entry)
		if dest == nil {
			return "(bug) hub " + hubID + " has no entry room " + cfg.Entry
		}
		dest.Add(w.Player)
		// Travel completes a player action (docs/systems/events.md).
		w.CheckEvents()
		return ""
	}
	return "(bug) unknown hub " + hubID
}
