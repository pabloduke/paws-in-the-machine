package engine

// Presence (docs/systems/presence.md): a story-owned entity's
// position as a function of world state — a Moore machine, derived
// every checkpoint, never stored. Presence says where; event rules
// say what you saw. Lives in engine core for the same reason events
// do: it runs inside the end-of-turn checkpoint.

// Placed declares where an entity is. Fn returns a room ID, or ""
// for offstage (out of every room, invisible, out of scope). Placed
// entities are story-owned: combining Placed with Portable is a
// content bug — the story and the player would fight over them.
type Placed struct {
	Fn func(w *World) string
}

func (Placed) Handle(*World, *Entity, Command) (string, bool) {
	return "", false
}

// applyPlacements moves every Placed entity to where its function
// says. Runs first in CheckEvents — before arrival detection and the
// rule poll — so rules always read a consistent board. Skipped while
// HoldPlacements is set (an open conversation): flags change live,
// but the stage doesn't re-arrange until the scene ends.
func (w *World) applyPlacements() {
	type move struct {
		e    *Entity
		dest string
	}
	var moves []move
	w.Root.Walk(func(e *Entity) bool {
		if p, ok := Part[Placed](e); ok && p.Fn != nil {
			moves = append(moves, move{e, p.Fn(w)})
		}
		return true
	})
	for _, mv := range moves {
		if mv.dest == "" {
			if mv.e.Parent != w.Offstage {
				w.Offstage.Add(mv.e)
			}
			continue
		}
		dest := w.FindID(mv.dest)
		if dest == nil {
			w.Pending = append(w.Pending,
				"(bug) placement of "+mv.e.ID+": no room "+mv.dest)
			continue
		}
		if mv.e.Parent != dest {
			dest.Add(mv.e)
		}
	}
}
