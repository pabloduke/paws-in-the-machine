package checks

import (
	"fmt"
	"hash/fnv"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Approach is how Buddy tries to get past an obstacle. Each approach is
// a player verb backed by a stat.
type Approach string

const (
	Sneak   Approach = "sneak"   // Stealth
	Parkour Approach = "parkour" // Agility
	Charm   Approach = "charm"   // Charm
)

// stat reads the backing stat for an approach.
func (a Approach) stat(w *engine.World) int {
	switch a {
	case Sneak:
		return w.Stats.Stealth
	case Parkour:
		return w.Stats.Agility
	case Charm:
		return w.Stats.Charm
	}
	return 0
}

// Check resolves a hidden d20 roll: roll + stat >= difficulty. The roll
// is a pure function of (world seed, id, stat) — the XCOM rule — so
// repeating an identical attempt repeats its result. id must uniquely
// name the attempt ("hound.sneak").
func Check(w *engine.World, id string, stat, difficulty int) bool {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d|%s|%d", w.Seed, id, stat)
	roll := int(h.Sum64()%20) + 1
	return roll+stat >= difficulty
}

// Cond is a flag condition — the same shape event rules use: every
// If flag true, every Unless flag false.
type Cond struct {
	If     []string
	Unless []string
}

// applies reports whether the condition currently holds.
func (c Cond) applies(w *engine.World) bool {
	for _, f := range c.If {
		if !w.Flags[f] {
			return false
		}
	}
	for _, f := range c.Unless {
		if w.Flags[f] {
			return false
		}
	}
	return true
}

// Mod is a situational modifier: a circumstance, read from flags,
// that shifts an attempt (docs/systems/stealth.md). Circumstances
// are telegraphed in prose by content; the numbers stay hidden.
type Mod struct {
	If     []string // flags that must all be true
	Unless []string // flags that must all be false
	Delta  int      // + works for Buddy, − against him
	// Consume names a flag spent by rolling with this mod, pass or
	// fail — a distraction is used the moment you move on it.
	Consume string
}

// applies reports whether the circumstance currently holds.
func (m Mod) applies(w *engine.World) bool {
	return Cond{If: m.If, Unless: m.Unless}.applies(w)
}

// Attempt is one open door through an obstacle: a difficulty, the
// prose for each outcome, and what the attempt does to the world.
type Attempt struct {
	Difficulty int
	Success    string
	Failure    string

	// Mods are the situational modifiers that can shift this attempt.
	Mods []Mod
	// OnFail flags are set when the attempt fails: failure is a story
	// state, not a retry gate — events, dialogue, presence, and Mods
	// on later attempts all read these off the blackboard.
	OnFail []string
	// Seals names a flag that closes this approach for good (a burned
	// route); SealedText is its refusal prose.
	Seals      string
	SealedText string
}

// Guarded makes an entity an obstacle that gates a destination. Content
// declares the approach matrix: approaches present in the map can be
// tried; approaches absent are impossible and refuse without a roll.
//
// On a successful attempt the player moves to Dest and the flag
// "<entityID>_bypassed" is set — once past, the obstacle stays solved
// and further attempts aren't needed.
type Guarded struct {
	// Dest is the room ID a successful approach leads to.
	Dest string
	// Approaches maps each possible approach to its attempt config.
	Approaches map[Approach]Attempt
	// Refusals overrides the default text for impossible approaches.
	Refusals map[Approach]string

	// Watcher names the observer whose perception gates this obstacle.
	// Empty means the entity itself is the observer; naming another
	// entity lets the gate and the eyes differ (a back door watched by
	// a hound) — and composes with presence: a watcher Placed out of
	// the room cannot observe (docs/systems/presence.md).
	Watcher string
	// Oblivious lists circumstances under which the watcher, though
	// present, cannot see (asleep, lured to a scrap bowl). Perception
	// only reads flags — observing never mutates.
	Oblivious []Cond
	// Unwatched is the delta every attempt gains while the watcher
	// can't see. Zero means perception doesn't matter here.
	Unwatched int
}

// Watched reports whether the obstacle's observer can currently see
// Buddy: the watcher is in Buddy's room and no Oblivious circumstance
// holds. A pure function of world state — dice never enter into what
// an observer perceives (docs/systems/visibility.md).
func (g Guarded) Watched(w *engine.World, self *engine.Entity) bool {
	watcher := self
	if g.Watcher != "" {
		watcher = w.FindID(g.Watcher)
	}
	if watcher == nil || !inRoom(w, watcher) {
		return false
	}
	for _, c := range g.Oblivious {
		if c.applies(w) {
			return false
		}
	}
	return true
}

// inRoom reports whether e is in the player's room subtree.
func inRoom(w *engine.World, e *engine.Entity) bool {
	for p := e; p != nil; p = p.Parent {
		if p == w.Room() {
			return true
		}
	}
	return false
}

func (g Guarded) Handle(w *engine.World, self *engine.Entity, cmd engine.Command) (string, bool) {
	approach := Approach(cmd.Verb)
	if _, isApproach := approachStats[approach]; !isApproach {
		return "", false
	}

	if w.Flags[self.ID+"_bypassed"] {
		dest := w.FindID(g.Dest)
		dest.Add(w.Player)
		return fmt.Sprintf("%s already knows to ignore you.", engine.Capitalize(self.Name)), true
	}

	attempt, possible := g.Approaches[approach]
	if !possible {
		if msg, ok := g.Refusals[approach]; ok {
			return msg, true
		}
		return fmt.Sprintf("That's not going to work on %s.", self.Name), true
	}

	if attempt.Seals != "" && w.Flags[attempt.Seals] {
		if attempt.SealedText != "" {
			return attempt.SealedText, true
		}
		return fmt.Sprintf("That door through %s is closed for good.", self.Name), true
	}

	// Circumstances shift the attempt; changed circumstances are new
	// inputs, so the XCOM rule re-rolls them (Check hashes the
	// effective stat). Distractions are spent by the roll either way.
	effective := approach.stat(w)
	for _, mod := range attempt.Mods {
		if !mod.applies(w) {
			continue
		}
		effective += mod.Delta
		if mod.Consume != "" {
			delete(w.Flags, mod.Consume)
		}
	}

	// Perception gates the odds, not the verb: an unwatched obstacle
	// still rolls, with the observer's blindness as one big modifier.
	// The effective stat feeds the hash, so a lapsed watcher is a
	// genuinely new roll (XCOM rule).
	if g.Unwatched != 0 && !g.Watched(w, self) {
		effective += g.Unwatched
	}

	id := self.ID + "." + string(approach)
	if !Check(w, id, effective, attempt.Difficulty) {
		// Failure is a fork, not a wall: it writes to the blackboard
		// and the world reacts (events, dialogue, later Mods).
		for _, f := range attempt.OnFail {
			w.Flags[f] = true
		}
		return attempt.Failure, true
	}

	w.Flags[self.ID+"_bypassed"] = true
	dest := w.FindID(g.Dest)
	dest.Add(w.Player)
	// The XP award is the roll you needed: harder-for-you pays more,
	// and grown stats (or a stacked deck of modifiers) shrink the
	// reward (self-balancing).
	award := attempt.Difficulty - effective
	if award < 1 {
		award = 1
	}
	return attempt.Success + "\n\n" + engine.AwardXP(w, award), true
}

// approachStats exists to recognize approach verbs in Handle.
var approachStats = map[Approach]bool{Sneak: true, Parkour: true, Charm: true}
