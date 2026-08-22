# Level editor

Status: chart geometry editor implemented; expansion into a full game editor
is the declared direction (2026-08-21).

## What the editor is

The editor is a developer-facing terminal UI for authoring the game's chart
geometry. It opens the versioned chart file used by the game, displays one
`z`/`w` slice of one chart at a time, and writes changes back to that file.

Today it is specifically a **chart editor**, not a general-purpose game or
level-content editor. A chart says where an existing room ID is placed and
which compass exits follow from adjacency. Rooms and all of their content are
defined elsewhere.

The implementation lives in `cmd/editor/`. The chart model and file format
live in `internal/systems/charts/`; shipped geometry lives in
`internal/game/content/charts.json`.

The declared shared frame, hierarchy, and interpretation rules for editor
menu sketches live in [`EDITOR_MENUS.md`](EDITOR_MENUS.md). Future sketches may
omit their borders; the shared bordered presentation is still implied.

## What it is for

The editor makes lawful map geometry visible while it is authored:

- rooms occupy integer `(x, y, z, w)` coordinates;
- north is up on screen;
- ordinary compass exits derive from neighboring cells;
- the inspector shows the exits the chart system derives at the cursor; and
- different floors and fourth-axis slices can be inspected separately.

This lets a designer see isolated rooms, accidental adjacency, and reciprocal
exits without maintaining a second map by hand.

## Running it

From the repository root:

```sh
go run ./cmd/editor
```

That opens `internal/game/content/charts.json`. A different chart file can be
passed explicitly:

```sh
go run ./cmd/editor path/to/charts.json
```

The game embeds the default file at build time. Saving in the editor changes
the source file on disk; it does not hot-reload an already running game
binary. Rebuild or restart the game to use the saved geometry.

## What it does today

| Key | Action |
|---|---|
| `h`/`j`/`k`/`l` or arrows | Move the cursor in `x`/`y` |
| `n` or Enter | Place or replace a room ID at the cursor |
| `d` | Remove the room ID from that chart coordinate |
| Tab | Cycle through charts already present in the file |
| `<` / `>` | Move between `z` floors |
| `[` / `]` | Move between `w` slices (kata/ana) |
| `s` | Save the chart file |
| `q` | Quit when clean; refuse to discard unsaved changes |
| `Q` | Quit and discard unsaved changes |

The editor also:

- displays only the occupied cells on the current slice, plus working space
  around them;
- validates chart references against a read-only catalog of the assembled game
  world;
- rejects unknown entity IDs and prevents one entity ID from being placed at
  more than one coordinate across the entire weave;
- marks unsaved work in the title;
- preserves deterministic JSON ordering, so an unchanged save is
  byte-identical; and
- loads existing gluings and includes their resulting exits in the inspector.

### Placement and deletion semantics

A room ID entered with `n` is a **reference to an existing game entity**. The
editor does not create that entity or any description, interaction, gate,
character, item, flag, or other content belonging to it. The editor resolves
the entered ID against the assembled game world before placing it. Unknown IDs
are rejected, as are IDs already placed at another coordinate in any chart.

An existing file containing an unknown ID or duplicate placement still opens
so it can be repaired. The editor reports every invalid location, but saving is
blocked until the errors are fixed; a blocked save leaves the file on disk
unchanged. This validates identity, not entity kind: the catalog does not yet
provide a room-type schema or make non-room content editable.

Likewise, `d` means **unplace this room ID from this coordinate**. It does not
delete the room entity or any content that refers to it. Unplacing a room can
change derived exits and can make content unreachable, so deletion from a
chart must not be treated as deletion from the game.

Removing every cell in a chart also does not remove its containing hub or
node. The editor cannot currently create, rename, or delete charts, hubs,
nodes, rooms, or other game entities.

For example, Okuda is not present in the chart file and therefore is not
editable in this UI, but it remains in the game because its hub, rooms,
network content, flags, events, journal entries, and tests are defined outside
the chart file.

## What it does not do

The current editor does not:

- create, rename, or delete game entities;
- create, rename, or delete charts, hubs, or metamap nodes;
- edit room prose, aliases, objects, characters, interactions, checks, gates,
  flags, events, journal entries, missions, or network content;
- show or edit authored `Blocked`, `Gated`, or `Guarded` behavior;
- author, change, or remove gluings;
- show every downstream reference before a geometry change;
- provide a metamap/node-management screen;
- provide undo/redo; or
- hot-reload geometry into a running game.

These limits describe only the current executable. The editor is intended to
grow beyond geometry into a full game editor.

## Source of truth and review

`internal/game/content/charts.json` is the geometry source of truth. The game
embeds it with `go:embed`, and the editor reads and writes the same format.
There is no generated Go copy. The editor's read-only entity catalog comes
from `game.NewWorld`, so an alternate chart path is still checked against the
identity of the assembled game rather than treated as a separate game.

Editor saves should be reviewed like code. Moving or unplacing one room can
change exits for that room and every adjacent room. Relevant verification is:

```sh
go test ./...
go vet ./...
git diff --check
```

Any change that alters player-visible movement or game-state transitions must
also update `docs/GAME_FLOW.md` under the repository's diagram-traceability
rule. A byte-for-byte geometry refactor that preserves behavior is exempt.

## Future TODOs

These are capability gaps, not declarations of new world content or missions.

### Make geometry editing safer

- [x] Validate entered entity IDs against the assembled game world during
  placement, load, and save.
- [x] Reject duplicate placements within or across charts and report every
  conflicting coordinate.
- [ ] Report downstream references affected by an unplacement.
- [ ] Add undo/redo for placement and deletion.
- [ ] Make saves atomic and offer a recoverable backup.
- [ ] Distinguish the UI wording for **unplace** from deletion of game
  content.

### Complete chart and weave authoring

- [ ] Add explicit creation, rename, and deletion workflows for charts.
- [ ] Add gluing creation, inspection, editing, and deletion.
- [ ] Visualize authored gates, guarded passages, and blocked directions on
  top of derived geometry.
- [ ] Add a validation view for isolated rooms, invalid gluing endpoints, and
  unintended adjacency before save.

### Reconcile the declared hierarchy

- [ ] Implement the metamap screen described in `docs/systems/hubs.md`, where
  nodes can be listed and linked without assigning coordinates to the
  metamap itself.
- [ ] Decide how existing `hubs.Hub` values map to metamap nodes before making
  node creation or deletion editable.
- [ ] Implement a flag-state preview only after its inputs and UI behavior are
  specified; the current editor has no flag-state view.

### Full game editing (declared future direction)

- [x] Establish that this command should grow into a full game editor rather
  than remain a geometry-only tool (declared 2026-08-21).
- [ ] Create NPCs and place them in the world.
- [ ] Create basic, data-driven quest lines. Advanced quests may remain
  hand-written in Go when their behavior does not fit the editor's quest
  model.
- [ ] Create world items, place them in the world, and author how they can be
  used, manipulated, and put into inventory.
- [ ] Create terminals with separate usernames and host names, use the host
  name as the network-map key, enter them from the editor, and author their
  fake directories and files from a deck-structured starting point.
- [ ] Author room descriptions and item descriptions.
- [ ] Author NPC dialogue.
- [ ] Add and manage world keywords.
- [ ] If full entity deletion is approved, define a dependency-aware workflow
  that finds hub registration, exits, gates, network hosts, flags, events,
  journal entries, mission references, tests, and diagrams before removal.
- [ ] Define the data schemas, validation, and reference handling for each
  content type before making it writable.
- [ ] Define the supported basic-quest vocabulary and the boundary where a
  quest should be implemented in Go instead.
- [ ] Preserve content authority: the editor stores designer-authored lore,
  dialogue, quests, and descriptions, but must not generate missing content
  on its own.

Until those rulings and mechanisms exist, removing a place from the game is a
coordinated code/content change, not an editor action.
