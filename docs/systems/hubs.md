# Hubs & Travel

Status: spec agreed.

## Model

The city is organized into hubs (districts): The Neighborhood, The
Plaza, Industrial Zone, Microslop HQ, ... Each hub contains explorable
rooms. In the world tree a hub is an ordinary entity:

    Root → hub → rooms → things

Buddy has lived in this city his whole life, so **every hub is known
and travelable from the start**. There are no locked or hidden hubs.
What game state gates is *relevance*: a hub can be visitable with
nothing story-relevant happening in it until flags say otherwise —
the same rule that governs NPC positions.

If a place must be physically inaccessible (Microslop HQ interior),
that is a Guarded obstacle or blocked exit *inside* the hub — never a
lock on the travel panel.

**Buildings deepen as the game does** (agreed 2026-07-07). Okuda HQ,
the first infiltration building, teaches "two ways in": getting inside
*is* the whole puzzle. Later buildings keep both doors but layer the
interior — badge-locked security doors, patrolled floors, keycards
lifted off desks — and the best of those locks live on the building's
own network, so a door can be opened from the terminal side (the
closed-ports flag contract, pointed the other direction). Getting in
is the early game; "in is only the start" is the late game.

The mechanism for those locks is built (`engine.Exits.Gated`): a gated
direction shows its `Shut` prose until its flag is true, then moves
normally. No dice — a lock is state, not an attempt; `checks.Guarded`
remains the tool for obstacles you try approaches against. First use:
Okuda's archive door, released by `run unlock.bin` on `okuda.grid`.

## Charts: the lattice under each hub

**RULED 2026-07-10, spec'd, not built** — implementation gets its own
branch after the mission-1 branch merges.

The world was originally ruled two-level (amended 2026-08-16 to
arbitrary depth — see "Nesting" below; both levels below survive the
amendment unchanged, they are simply no longer the only two):

- **Between hubs: a graph.** Hubs are nodes; travel is the edges. No
  geometric promise between hubs, ever — the subway model: nobody
  riding it knows or cares whether the plaza is "north" of the lair.
  Distance between districts is a menu of stops, not a lie about
  adjacency.
- **Inside a hub: a lattice ("chart").** Every room in a hub gets a
  coordinate — an int slice, N-capable; in practice four slots
  `(x, y, z, w)` — and walkable exits **derive from adjacency**:
  north is `y+1`, up is `z+1`. Reciprocity and geometric consistency
  hold by construction; the compass cannot lie.

## Nesting: charts inside charts

**RULED 2026-08-16 (user-declared), spec'd, not built.** Amends the
two-level model above to arbitrary depth. The containment chain:

    metamap → node → location → room → (optional metamap) → …

- **metamap** — a *directionless* container of nodes. No geometric
  promise between its children, ever. This is the subway model above,
  named and generalized: the existing hub graph is a metamap, and the
  rule that protected it now protects every metamap at every depth.
- **node** — a vertex of a metamap. Carries a grid; locations sit at
  `(x, y)` within it.
- **location** — a cell of a node's grid. Carries a grid of its own;
  rooms sit at `(x, y)` within it.
- **room** — a cell of a location's grid. Holds prose and objects, and
  is where the player stands. A room **may contain a metamap**, which
  is what makes the chain recursive rather than four fixed levels.

Node and location are the same structure at different depths — a grid
whose cells hold children. Metamap-in-a-room therefore needs no special
case; it is the recursion closing on itself.

**Nesting is containment, not subdivision.** Each grid has its own
local coordinate space, with no arithmetic relationship to its parent.
An interior may be larger than its exterior. This is required, not
incidental: aisle 410 sits one step ana of the Okuda archive, and the
whole fold system depends on space not being globally Euclidean. A
single global coordinate space would outlaw the game's own content.

**Level changes are gluings.** Descending from a location into a room's
metamap, or crossing between charts, uses the same gluing table that
authors folds — the table already permits identifications "possibly
across charts." One mechanism serves both, so a door into a building
and a fold into aisle 410 are the same kind of declaration, differing
only in what they connect.

**Coordinates live in nodes, never in metamaps (ruled 2026-08-16).** A
metamap has no coordinate space at all — not in play, and not as an
authoring convenience. Its nodes are an unordered set joined by travel,
exactly as the subway model requires. Everything positional happens one
level down, inside a node's grid. The editor follows suit: a metamap
screen lists and links its nodes; only a node screen draws a grid.

**Open questions (parked, not ruled):**

- Whether the existing `hubs.Hub` becomes the `node` level outright, or
  the two coexist.
- How deep act one actually goes; nothing requires using every level.

**Design rule: geometry is lawful.** The scrambled-exit fakery of
70s/80s text adventures (Zork mazes: north from A reaches B, south
from B reaches somewhere else) is banned outright — a player mapping
this game in a notebook must never be cheated. Hand-declared
connections remain allowed *on top of* the lattice for doors and
obstacles — `engine.Exits.Gated` locks and `checks.Guarded` passages
are content, adjacency is geometry — but they connect cells that are
actually adjacent (or are folds, below).

**Folded space is lawful too.** The lattice's fourth axis is where
hyperspatial content lives, authored as a **gluing table**: declared
identifications between cells/faces, possibly across charts — wraps,
twists (Klein-bottle-style orientation reversal), tesseract folds
(Heinlein's "—And He Built a Crooked House—" is the house style
precedent), or a backstage lattice whose kata-side faces touch thin
spots in several hubs at once (a wormhole network). A fold is never a
scrambled edge: it's a rule the player can discover, learn, and map.

**Vocabulary (user ruling 2026-07-10):** the player only ever sees
`north/south/east/west/up/down`. The folding is invisible in the
interface; discovery is cartographic — the map stops closing, a loop
comes home too short. The 4D direction words **ana/kata** (`w±1`,
Hinton's terms) are dev-facing only: docs, comments, coordinates.
They never appear in game text.

**The cat clause (user ruling 2026-07-10):** Buddy cannot sense folds
— he *survives* them. A fold transit reorients the traveler
(floor↔wall); a human comes out on their head, a cat lands on his
feet. Fold-transit prose always carries the lands-on-his-feet beat.
This is a lore gate: humans can't traverse folded space, which is why
the Resistance runs cat agents, and Masquerade-adjacent — corpos
can't even survey what's one step ana of their own archive. First
planned use: Okuda's missing aisle 410 sits at the archive's
coordinates, one step ana (`w+1`).

**Retrofit notes (for the build branch):**

- 10 existing rooms across 3 hubs need coordinates. The Neighborhood
  and the Plaza are charted (2026-08-16). **Okuda is not realizable on
  a lattice as wired** — the guard's passage joins two cells the other
  exits force diagonal; see docs/systems/charts.md "Okuda is not
  lattice-realizable" for the four possible resolutions, all of them
  content decisions.
- Okuda already uses `up`/`down` correctly; its two check/gate
  passages (fire escape, archive door) map onto real adjacencies.
- Hub travel is untouched: the chart replaces hand-wired `Exits.Dirs`
  inside hubs, nothing between them.

**Open questions (parked, not ruled):**

- Stations: hub travel is currently boardable from any room, including
  mid-infiltration. Whether travel should only be offered from a
  chart's street/entry rooms is undecided.
- Whether any fold content lands in act one, or the fourth axis stays
  authored-but-unvisited until later.

## Travel UX: the side panel

A persistent side panel next to the transcript lists all hubs, marking
the one Buddy is in.

- **Shift+Tab** shifts focus between the command prompt and the panel.
- With the panel focused: **Up/Down** select a hub, **Enter** travels
  (Buddy arrives at that hub's entry room; the room description prints
  to the transcript), **Esc/Shift+Tab** returns to the prompt without
  traveling.
- Typing at the prompt never affects the panel; panel keys never reach
  the prompt.

Traveling is a player action like any other (turns.md): it changes
where Buddy is, story output narrates it, and nothing else moves.

## Engine surface

- `internal/systems/hubs` package: `Hub` component (config: `Entry`,
  the room ID where travel lands), `List` (all hubs), `Current` (hub
  containing the player), `Travel` (move + describe).
- Content declares a hub by attaching `hubs.Hub` to an entity that is
  a direct child of Root and adding its rooms as children.
