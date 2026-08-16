# Charts

Status: spec agreed (from the hubs.md chart ruling 2026-07-10 and the
nesting amendment 2026-08-16); building.

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

This keeps the system inside its boundary: charts never touch
`internal/engine`, and every existing gate and check keeps working
unchanged on top of derived adjacency.

## Constraint: no mutable state

Charts hold configuration, never mutable state (`engine/component.go`'s
discipline rule). Nothing about a chart changes during play. A passage
that opens or closes is a flag on a gate, not a mutated coordinate —
which is what keeps positions derivable, saves small, and the editor's
flag-state view honest.

## Boundaries

- `internal/systems/charts/` + `docs/systems/charts.md` — owned outright
- `internal/game/` — content declares charts; the composition root wires
- `docs/SYSTEMS.md` — one row, added on main
- No engine changes required
