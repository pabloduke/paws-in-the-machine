package engine

import (
	"fmt"
	"strings"
)

// Engine drives the core loop: player command -> parsed intent ->
// component dispatch -> world update -> story output.
type Engine struct {
	World *World
}

// New wraps a fully constructed world.
func New(w *World) *Engine {
	return &Engine{World: w}
}

// Execute runs one turn. The command is offered to the target entity's
// components in attachment order; if none handles it, the engine
// default for the verb runs.
func (e *Engine) Execute(input string) string {
	w := e.World
	input = w.Rewrite(input)
	cmd, ok := Parse(input)
	if !ok {
		if cmd.Verb == "" {
			return ""
		}
		return fmt.Sprintf("You don't know how to %q. (Try \"help\".)", cmd.Verb)
	}

	// Room-scoped verbs target the current room.
	switch cmd.Verb {
	case "look":
		if cmd.Object == "" {
			return dispatch(w, w.Room(), cmd, func() string { return Look(w) })
		}
		cmd = Command{Verb: "examine", Object: cmd.Object}
	case "go":
		return dispatch(w, w.Room(), cmd, func() string { return Go(w, cmd.Object) })
	case "inventory":
		return Inventory(w)
	case "stats":
		return StatSheet(w)
	case "train":
		return Train(w, cmd.Object)
	case "help":
		return helpText
	case "quit":
		w.Quit()
		return "The rain keeps falling. Somewhere, the sun waits."
	}

	// Everything else targets a named entity.
	if cmd.Object == "" {
		return fmt.Sprintf("%s what?", Capitalize(cmd.Verb))
	}
	target := w.InScope(cmd.Object)
	if target == nil {
		return fmt.Sprintf("You don't see any %q here.", cmd.Object)
	}
	return dispatch(w, target, cmd, func() string { return defaultFor(w, target, cmd) })
}

// dispatch offers cmd to the target's components, falling back to def.
func dispatch(w *World, target *Entity, cmd Command, def func() string) string {
	for _, p := range target.Parts {
		if out, handled := p.Handle(w, target, cmd); handled {
			return out
		}
	}
	return def()
}

// defaultFor is the engine's built-in handling for entity-targeted verbs.
func defaultFor(w *World, target *Entity, cmd Command) string {
	switch cmd.Verb {
	case "examine":
		return fmt.Sprintf("You see nothing special about %s.", target.Name)
	case "take":
		return Take(w, target)
	case "drop":
		return Drop(w, target)
	case "use":
		return "You paw at it, but nothing happens."
	case "knock":
		return "You give it a speculative shove. It stays put. Disappointing."
	case "turn":
		return "You nose at it, but it doesn't turn."
	case "sneak":
		return fmt.Sprintf("There's no sneaking past %s. It isn't in your way.", target.Name)
	case "parkour":
		return fmt.Sprintf("You size up %s for a route. There's nothing to parkour around.", target.Name)
	case "charm":
		return fmt.Sprintf("You aim the adopt-me eyes at %s. Nothing to gain here.", target.Name)
	case "talk":
		return fmt.Sprintf("%s has nothing to say.", Capitalize(target.Name))
	}
	return "Nothing happens."
}

// StatSheet renders Buddy's visible numbers.
func StatSheet(w *World) string {
	sheet := fmt.Sprintf(
		"Level %d   XP %d/%d\nStealth %d   Agility %d   Charm %d",
		w.Level, w.XP, w.NextLevelCost(),
		w.Stats.Stealth, w.Stats.Agility, w.Stats.Charm)
	if w.StatPoints > 0 {
		sheet += fmt.Sprintf("\nStat points to spend: %d (train <stat>)", w.StatPoints)
	}
	return sheet
}

// Train spends one stat point to raise a stat by one.
func Train(w *World, stat string) string {
	target := w.Stats.ByName(stat)
	if target == nil {
		return "Train what? (stealth, agility, charm)"
	}
	if w.StatPoints == 0 {
		return "No stat points to spend — level up first."
	}
	w.StatPoints--
	*target++
	return fmt.Sprintf("%s %d → %d. Points left: %d",
		Capitalize(stat), *target-1, *target, w.StatPoints)
}

// --- Default verb implementations, exported so components can invoke
// --- the stock behavior after their own logic runs.

// Look renders the current room: name, description, exits. Visible
// entities are not listed here — they're the UI's YOU SEE panel
// (World.Visible); headless consumers can query it directly.
func Look(w *World) string {
	room := w.Room()

	var b strings.Builder
	b.WriteString(room.Name)
	if d, ok := Part[Description](room); ok {
		b.WriteString("\n\n" + d.render(w))
	}
	if x, ok := Part[Exits](room); ok && len(x.Dirs) > 0 {
		dirs := make([]string, 0, len(x.Dirs))
		for dir := range x.Dirs {
			dirs = append(dirs, dir)
		}
		b.WriteString("\n\nExits: " + strings.Join(dirs, ", "))
	}
	return b.String()
}

// Go moves the player through a room exit. Movement returns only extra
// narration (a room's "enter" component, if any) — describing the new
// room is the UI's job, which renders the current room from state.
func Go(w *World, dir string) string {
	if dir == "" {
		return "Go where?"
	}
	x, ok := Part[Exits](w.Room())
	blocked := "You can't go that way."
	if ok && x.Blocked != "" {
		blocked = x.Blocked
	}
	if !ok || x.Dirs[dir] == "" {
		return blocked
	}
	dest := w.FindID(x.Dirs[dir])
	if dest == nil {
		return blocked
	}
	dest.Add(w.Player)
	return dispatch(w, dest, Command{Verb: "enter"}, func() string { return "" })
}

// Take moves a Portable entity into the player's inventory.
func Take(w *World, target *Entity) string {
	if w.Carried(target) {
		return "You already have it."
	}
	if _, ok := Part[Portable](target); !ok {
		return fmt.Sprintf("%s isn't going anywhere.", Capitalize(target.Name))
	}
	w.Player.Add(target)
	return fmt.Sprintf("You take %s.", target.Name)
}

// Drop moves a carried entity into the current room.
func Drop(w *World, target *Entity) string {
	if !w.Carried(target) {
		return fmt.Sprintf("You aren't carrying %s.", target.Name)
	}
	w.Room().Add(target)
	return fmt.Sprintf("You drop %s.", target.Name)
}

// Inventory lists what the player carries.
func Inventory(w *World) string {
	if len(w.Player.Contents) == 0 {
		return "You are carrying nothing. Traveling light."
	}
	var b strings.Builder
	b.WriteString("You are carrying:")
	for _, c := range w.Player.Contents {
		b.WriteString("\n  " + c.Name)
	}
	return b.String()
}

// Capitalize upper-cases the first letter — the shared prose helper
// for building sentences from entity and stat names.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

const helpText = `Commands:
  look (l)              describe your surroundings
  examine <thing> (x)   look closely at something
  go <direction>        move (or just: north, n, in, out ...)
  take / drop <thing>   manage your possessions
  use <thing>           operate something (also: jack)
  knock <thing>         you are a cat (also: bat, swat, paw)
  turn <thing>          rotate something (also: twist)
  sneak past <thing>    stealth your way through
  parkour <thing>       the acrobatic route (also: leap, vault)
  charm <thing>         weaponized cuteness (also: purr)
  talk <person>         start a conversation
  stats                 your numbers
  train <stat>          spend a stat point (earned by leveling up)
  inventory (i)         what you're carrying
  quit (q)              end the session`
