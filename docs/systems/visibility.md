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

- Right panel = YOU SEE: visible entities (current labels) + exits.
- Center panel = the room's description only — the story text, which
  grows richer through state-dependent Description.Fn as the player
  acts on the world.

## Engine surface

- `World.Visible()` — visible entities in tree order (excludes the
  player and carried items).
- `World.InScope(name)` — name resolution over the visible subtree
  plus inventory.
- `Openable{Flag}` — conceals contents until flag set.
- `Aspect{Fn}` — state-dependent display name; `DisplayName(w, e)`
  resolves it.

## Later (not built)

- Darkness/light, hiding spots, smell-based perception for dogs, and
  what OBSERVERS can see (the stealth modifier side of visibility).
