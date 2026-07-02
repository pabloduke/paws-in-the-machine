package engine

import (
	"fmt"
	"math/rand/v2"
)

// Stats are Buddy's capabilities. Each stat is a verb against an
// obstacle: Stealth = sneak past, Agility = parkour around,
// Charm = charm. See docs/systems/stealth.md.
type Stats struct {
	Stealth int
	Agility int
	Charm   int
}

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

	// Stats, XP, Level, and StatPoints are visible via the "stats"
	// command. XP is earned by actions and fills a level track with
	// escalating costs; each level grants a stat point; "train <stat>"
	// spends points. XP counts progress toward the NEXT level only.
	Stats      Stats
	XP         int
	Level      int
	StatPoints int
	// Seed makes checks deterministic (the XCOM rule): identical
	// attempts give identical results. Generated at new-game time;
	// content or tests may overwrite it before play.
	Seed int64

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
		Seed:     rand.Int64(),
		Level:    1,
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

// LevelCost is the XP needed to go from level n to n+1: LevelCostBase*n.
const LevelCostBase = 5

// NextLevelCost returns the XP required to reach the next level.
func (w *World) NextLevelCost() int { return LevelCostBase * w.Level }

// AwardXP is the single funnel for all XP gains (checks, hacks, story
// beats). It resolves any level-ups — escalating cost per level — and
// returns the player-visible message ("+5 XP — LEVEL 2! ...").
func AwardXP(w *World, n int) string {
	w.XP += n
	msg := fmt.Sprintf("+%d XP", n)
	leveled := 0
	for w.XP >= w.NextLevelCost() {
		w.XP -= w.NextLevelCost()
		w.Level++
		w.StatPoints++
		leveled++
	}
	if leveled > 0 {
		plural := "point"
		if w.StatPoints != 1 {
			plural = "points"
		}
		msg += fmt.Sprintf(" — LEVEL %d! %d stat %s to spend (train <stat>)",
			w.Level, w.StatPoints, plural)
	}
	return msg
}

// Quit marks the session as over; the UI shuts down after the current
// output is shown.
func (w *World) Quit() { w.quitting = true }

// Quitting reports whether Quit has been called.
func (w *World) Quitting() bool { return w.quitting }
