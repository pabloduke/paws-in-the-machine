package checks

import (
	"fmt"
	"hash/fnv"
	"strings"

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

// Attempt is one open door through an obstacle: a difficulty and the
// prose for each outcome.
type Attempt struct {
	Difficulty int
	Success    string
	Failure    string
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
}

func (g Guarded) Handle(w *engine.World, self *engine.Entity, cmd engine.Command) (string, bool) {
	approach := Approach(cmd.Verb)
	if _, isApproach := approachStats[approach]; !isApproach {
		return "", false
	}

	if w.Flags[self.ID+"_bypassed"] {
		dest := w.FindID(g.Dest)
		dest.Add(w.Player)
		return fmt.Sprintf("%s already knows to ignore you.", capitalized(self.Name)), true
	}

	attempt, possible := g.Approaches[approach]
	if !possible {
		if msg, ok := g.Refusals[approach]; ok {
			return msg, true
		}
		return fmt.Sprintf("That's not going to work on %s.", self.Name), true
	}

	id := self.ID + "." + string(approach)
	if !Check(w, id, approach.stat(w), attempt.Difficulty) {
		return attempt.Failure, true
	}

	w.Flags[self.ID+"_bypassed"] = true
	dest := w.FindID(g.Dest)
	dest.Add(w.Player)
	return attempt.Success, true
}

// approachStats exists to recognize approach verbs in Handle.
var approachStats = map[Approach]bool{Sneak: true, Parkour: true, Charm: true}

func capitalized(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
