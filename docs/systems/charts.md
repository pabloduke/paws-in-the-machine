# Charts

Status: spec agreed (from the hubs.md chart ruling 2026-07-10 and the
nesting amendment 2026-08-16); core geometry built. The original chart-editor
slice was retired when `cmd/editor` restarted as a navigation prototype on
2026-08-22.

The lattice under a node. `docs/systems/hubs.md` owns the *rulings*
(subway model, lawful geometry, folds, vocabulary, the cat clause);
this file owns the *mechanism*.

## What a chart is

A chart is a grid. Cells sit at integer coordinates `(x, y, z, w)` and
hold one entity each — a location or a room, depending on depth. A
chart is sparse: unoccupied coordinates are simply nothing.

    metamap → node → location → room → (optional metamap)
              ^^^^   ^^^^^^^^
              both of these are charts

Node and location are the same structure at different depths, so one
type serves both. The metamap level is *not* a chart: it is
directionless and has no coordinate space at all (hubs.md, ruled
2026-08-16). Today the metamap level is the existing hub graph; whether
`hubs.Hub` becomes the node level outright is still open.

## Exits derive from adjacency

Walkable exits are computed from coordinates, never hand-declared:

| Direction | Delta |
|---|---|
| north / south | `y+1` / `y-1` |
| east / west | `x+1` / `x-1` |
| up / down | `z+1` / `z-1` |

If the neighbouring coordinate is occupied, the exit exists. If it
isn't, it doesn't. Reciprocity is therefore free: adjacency is
symmetric, so north-then-south always returns you where you started.
The compass cannot lie, and a scrambled edge is not representable —
there is no way to author one.

**The fourth axis derives nothing.** `w±1` (ana/kata) is dev-facing
vocabulary only and never becomes a player exit — the player's words
stay `north/south/east/west/up/down` (hubs.md ruling). Movement along
`w` happens only through an authored gluing, which presents itself as
an ordinary compass direction. That is what makes folding invisible in
the interface and discoverable only cartographically.

## Integrity

Two invariants are checked when a weave is assembled, because violating
either makes the derived geometry meaningless:

- **Identity.** Chart IDs are unique and non-empty, and an entity
  occupies exactly one cell across the whole weave. `Apply` iterates
  maps and rewrites an entity's exits per placement, so an entity in two
  cells would take whichever geometry the iteration reached last.
- **Face ownership.** A gluing owns two faces: its own and the return
  face it installs. A later declaration claiming either face is rejected
  as a content bug rather than overwriting it. Overwriting a return face
  would leave the earlier passage reaching its destination while the way
  back led somewhere else — a scrambled edge, which is exactly what this
  system forbids. Re-declaring an identical gluing is not a conflict.

Both endpoints of a gluing must be occupied cells. A gluing overrides
derived adjacency for its direction, so one pointing at an empty cell
would silently delete a passage rather than create one.

A rejected declaration leaves the weave unchanged, and every check
reports a content bug rather than panicking, so a bad file surfaces as a
visible complaint instead of nondeterministic geometry.

## Gluings

A gluing is a declared identification: leaving one cell in a given
direction arrives at another cell, possibly in another chart. One
mechanism covers three jobs:

- **folds** — a compass step that lands one cell ana (aisle 410)
- **level changes** — descending from a location into a room's metamap,
  or crossing between charts
- **wraps and twists** — Klein-bottle orientation reversal, tesseract
  folds

Gluings are **bidirectional by construction**: declaring one installs
its reverse. A one-way passage would be a scrambled edge, which the
lawful-geometry rule bans. Content that wants a door you can't come
back through uses a gate or an obstacle — that is content, not
geometry.

A gluing overrides derived adjacency for that direction, so a fold can
sit where a plain neighbour would otherwise be.

## Integration

Charts produce `engine.Exits` and nothing else. Geometry is static —
it is not a function of flags — so exits are computed once when the
world is built, and the story layers on top exactly as it does today:

- `engine.Exits.Gated` locks a derived direction behind a flag
- `checks.Guarded` puts an obstacle on it
- authored `Blocked` prose still covers directions with no neighbour

This keeps the system inside its boundary in the sense that matters:
charts require no changes to `internal/engine`, and every existing gate
and check keeps working unchanged on top of derived adjacency. The
package does import the engine and `Apply` writes `engine.Exits` onto
room entities — the adapter from geometry to components lives here, in
the charts package, rather than in the engine.

## Constraint: no mutable state

Charts hold configuration, never mutable state (`engine/component.go`'s
discipline rule). Nothing about a chart changes during play. A passage
that opens or closes is a flag on a gate, not a mutated coordinate —
which is what keeps positions derivable, saves small, and the editor's
flag-state view honest.

## Content file

Geometry is data, not Go — the first content to cross that line. The
file is versioned JSON (stdlib only, the same discipline as
`engine.SaveState`), embedded with `go:embed` so the shipped binary
stays self-contained while the editor reads and writes the same file on
disk. One format, one source of truth, no generated Go to clobber.

    {
      "version": 1,
      "charts": [
        {
          "id": "neighborhood",
          "cells": {
            "0,0": "lair",
            "0,1": "coffeeshop"
          }
        }
      ]
    }

Cells are keyed by coordinate — `"x,y"`, `"x,y,z"` or `"x,y,z,w"`, with
trailing zeros dropped — so a room is one line and moving one is a
one-line diff. Content review is done by a human; the format is shaped
for that.

`Marshal` is deterministic, so saving an unchanged weave is byte-identical and
future authoring tools need not manufacture diff noise. A round-trip test pins
it.

## The editor

The current editor scope and future work are documented in
[`docs/EDITOR.md`](../EDITOR.md). `cmd/editor` now persists UUID-based
Location-to-Hub coordinates with a 10×10 default sparse viewport and
Room-to-Location coordinates with a 5×5 default sparse viewport. It does not
rewrite the existing runtime `charts.json`; bridging those editor
relationships into runtime chart geometry remains separate work.
Parent ownership is stored independently from coordinates, so assigned but
unplaced Locations and Rooms are valid editor content.

The retired chart-grid implementation remains in Git history. When geometry
authoring returns, it belongs beneath the editor's declared `Place` workflow.
It must restore assembled-world identity validation, reject unknown IDs and
duplicate placements, and keep invalid saves from modifying the chart file.

## Okuda is not lattice-realizable as wired (raised 2026-08-16)

Charting the existing rooms surfaced a contradiction in Okuda HQ that
predates charts. Two hubs retrofitted cleanly; Okuda cannot, and the
fix is a content decision.

Okuda's six rooms carry these connections today — three plain exits and
three `checks.Guarded` passages, which the lawful-geometry rule says
must also join genuinely adjacent cells:

| From | To | How |
|---|---|---|
| street | lobby | plain, north |
| street | alley | plain, east |
| office | lobby | plain, down |
| corridor | alley | plain, down |
| lobby | office | Guarded — receptionist charm |
| alley | corridor | Guarded — fire escape |
| **corridor** | **office** | **Guarded — past the guard** |

The first four pin every coordinate. Taking street as the origin:

    lobby    = street + north      = (0, 1, 0)
    alley    = street + east       = (1, 0, 0)
    office   = lobby  + up         = (0, 1, 1)
    corridor = alley  + up         = (1, 0, 1)

That leaves the corridor and the office **diagonal** — two steps apart,
never adjacent — so the guard's passage between them cannot be a
lawful edge. The layout is over-constrained: no assignment of
coordinates satisfies all seven connections at once.

Worse, the only placement that keeps the annex east of the office puts
it at `(1, 1, 1)`, directly north of the corridor — which would
*derive* a brand-new corridor↔annex exit, bypassing the flag-gated
archive door and breaking the closed-ports puzzle.

Resolutions, all of them content decisions and none taken here:

1. **Add a connecting cell** — a landing or hallway between the
   corridor and the office, making the guard's passage a real edge.
   This is new content (a room), which is user-owned.
2. **Move a room** — e.g. re-site the alley or the corridor so the
   second floor closes. Changes the map players walk.
3. **Re-route the guard** — have that passage lead somewhere adjacent
   instead. Changes the puzzle's shape.
4. **Leave Okuda uncharted** — hand-declared exits keep working
   indefinitely; charts and hand-wiring coexist by design.

Until it is ruled, Okuda keeps its hand-declared exits and is absent
from `gameCharts()`. Nothing is broken: this is the chart system doing
its job, catching geometry that a notebook-mapping player would
eventually have caught instead.

## Boundaries

- `internal/systems/charts/` + `docs/systems/charts.md` — owned outright
- `internal/game/` — content declares charts; the composition root wires
- `docs/SYSTEMS.md` — one row, added on main
- No engine changes required
