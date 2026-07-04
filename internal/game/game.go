// Package game holds the content of Paws in the Machine: the entity
// tree, descriptions, and story triggers. All mechanics live in the
// engine package; this package only declares.
//
// House rule: this package contains only literals, component values,
// and small hook funcs — never engine changes. A behavior needed by
// more than one entity graduates into a reusable component.
//
// Layout (docs/BOUNDARIES.md): world.go stitches per-area files
// (lair.go, coffeeshop.go, ...) into the tree; flags.go declares
// every story flag once; net.go declares the hacking net.
package game

import "github.com/pabloduke/paws-in-the-machine/internal/engine"

// Intro is shown once when the session starts.
const Intro = `PAWS IN THE MACHINE

(Intro text goes here. Type "help" for commands.)`

// awardOnce grants XP the first time flag trips; returns the XP line
// (with leading separator) or "" if already earned.
func awardOnce(w *engine.World, flag string, xp int) string {
	if w.Flags[flag] {
		return ""
	}
	w.Flags[flag] = true
	return "\n\n" + engine.AwardXP(w, xp)
}
