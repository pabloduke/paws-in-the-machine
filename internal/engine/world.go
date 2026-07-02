package engine

// World is the complete game state: one entity tree plus story flags.
type World struct {
	// Root anchors the tree; rooms are its children.
	Root *Entity
	// Player is an entity inside a room; its Contents are the inventory.
	Player *Entity
	// Flags is free-form mutable story state ("heard_whisper", ...).
	// Components read and write flags rather than holding state.
	Flags map[string]bool
	// Rewrites maps whole input phrases to replacement commands,
	// letting content define idioms ("jack in" -> "use deck").
	Rewrites map[string]string

	quitting bool
}

// NewWorld returns a world containing only the root and the player.
// Content must place the player into a room.
func NewWorld() *World {
	return &World{
		Root:     NewEntity("root", "root"),
		Player:   NewEntity("player", "yourself", "self", "me", "buddy"),
		Flags:    map[string]bool{},
		Rewrites: map[string]string{},
	}
}

// Room returns the entity the player is directly inside.
func (w *World) Room() *Entity {
	return w.Player.Parent
}

// FindID locates an entity anywhere in the world by ID.
func (w *World) FindID(id string) *Entity {
	var found *Entity
	w.Root.Walk(func(e *Entity) bool {
		if e.ID == id {
			found = e
			return false
		}
		return true
	})
	return found
}

// InScope resolves a player-typed name against everything reachable:
// the current room's subtree (which includes the player and inventory).
func (w *World) InScope(name string) *Entity {
	var found *Entity
	w.Room().Walk(func(e *Entity) bool {
		if e != w.Room() && e.Matches(name) {
			found = e
			return false
		}
		return true
	})
	return found
}

// Carried reports whether the player holds e (directly or nested).
func (w *World) Carried(e *Entity) bool {
	for p := e.Parent; p != nil; p = p.Parent {
		if p == w.Player {
			return true
		}
	}
	return false
}

// Quit marks the session as over; the UI shuts down after the current
// output is shown.
func (w *World) Quit() { w.quitting = true }

// Quitting reports whether Quit has been called.
func (w *World) Quitting() bool { return w.quitting }
