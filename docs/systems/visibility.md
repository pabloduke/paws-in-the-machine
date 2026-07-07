# Visibility & Scope

Status: core model agreed and built.

## The model: invisible means enclosed

There is no knowledge tracking and no "noticed" state. What the player
can see and reach is a pure function of world state:

- **Everything visible is truthfully labeled.** If the bookcase is
  ajar, it is listed as "ajar bookcase" from the moment Buddy enters
  the room (state-dependent labels via the Aspect component). The world
  hides nothing that is in view.
- **Invisible means physically enclosed.** The only way something is
  out of sight is being inside a closed container (Openable component,
  closed until its flag is set) or otherwise absent from the room's
  subtree. A ruby in a shut drawer is not in scope: "take ruby" says
  you don't see any ruby — because you don't.
- **Scope stops at closed containers.** Name resolution and the YOU
  SEE list walk the room subtree but do not descend into a closed
  Openable. The container itself stays visible and targetable; its
  contents don't exist for the player until it opens.

## Observer vs actor verbs

- **Observers** (look, examine, stats, inventory) mutate nothing —
  examining the bookcase does not change the bookcase, or anything
  else. Objects are not quantum.
- **Actors** (take, turn, knock, use, move, ...) may mutate state, and
  changing enclosure is how things are revealed: turn the candlestick →
  flag set → drawer open → the ruby (always physically present in the
  drawer's contents) enters scope, appears in YOU SEE, and becomes
  takeable. Room descriptions read the same flags, so the story pane
  updates with the same action.

## UI contract

- Right panel, top: BUDDY — level and XP at a glance, plus a pending
  stat-point marker. The full stat sheet and the inventory live in
  centered modals (`stats`, `inventory`/`i`); Buddy still knows what he
  carries — it's one keypress away, and the modal scrolls when the
  haul outgrows the screen. Leveling up opens a must-spend stat picker
  automatically (deferred to conversation end if it lands mid-dialogue).
- Right panel, middle: YOU SEE — **two-tier, og-adventure style**.
  Portable entities are listed by default; the Notable marker promotes
  plainly-visible non-portables (an NPC). Everything else is scenery:
  present, examinable, targetable, but discovered by *reading the
  prose* — noticing "bolted to the desk" is the gameplay. Exits are
  always listed (navigation is not a puzzle).
- Center panel = the room's description only — the story text, which
  grows richer through state-dependent Description.Fn as the player
  acts on the world.

## Prose-noun coverage (content authoring invariant)

Both directions must hold:

- Every entity is either listed (Portable/Notable) or mentioned in the
  room's prose — nothing undiscoverable.
- Every noun the prose mentions is examinable. "You don't see any
  door" after the narrator described a door is a content bug.

Scenery descriptions are signals: "a normal door, pretty boring"
politely closes a thread; "I wonder who it really belongs to" opens
one.

## Engine surface

- `World.Visible()` — visible entities in tree order (excludes the
  player and carried items).
- `World.InScope(name)` — name resolution over the visible subtree
  plus inventory.
- `Openable{Flag}` — conceals contents until flag set.
- `Aspect{Fn}` — state-dependent display name; `DisplayName(w, e)`
  resolves it.

## Observer-side perception (built)

What OBSERVERS can see — the stealth-modifier side of visibility —
lives in the checks system: a `Guarded` obstacle names a `Watcher`
whose blindness (absent from Buddy's room via presence, or declared
`Oblivious` flag circumstances) applies the `Unwatched` delta to
every attempt. Perception stays a pure function of world state and
never mutates — see docs/systems/stealth.md, "Observers have
perception".

## Later (not built)

- Darkness/light, hiding spots, and smell-based perception for dogs
  (an observer type that ignores visual circumstances).
