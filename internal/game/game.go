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

// Intro seeds the overworld LOG. The game boots into the terminal
// (main calls BootIntoDeck), so this is what greets Buddy the first
// time he logs out into the lair.
const Intro = `PAWS IN THE MACHINE

(Placeholder) The deck's glow fades behind you. The lair: rain on the
window, neon through the blinds, a city that owes you nothing yet.
Type "help" for commands.`

// awardOnce grants XP the first time flag trips; returns the XP line
// (with leading separator) or "" if already earned.
func awardOnce(w *engine.World, flag string, xp int) string {
	if w.Flags[flag] {
		return ""
	}
	w.Flags[flag] = true
	return "\n\n" + engine.AwardXP(w, xp)
}
