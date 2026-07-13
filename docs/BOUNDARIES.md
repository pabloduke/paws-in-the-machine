# Boundaries

Status: draft, under discussion.

Rules for parallel work: multiple agents/branches at once, no merge
conflicts. The system packages are already isolated (see Architecture
in the readme); these rules cover everything around them — the
integration layer, shared files, and how features enter the tree.

## Ownership map

A feature branch for system `<name>` may touch:

| Path | Ownership |
|---|---|
| `internal/systems/<name>/` | owned outright |
| `docs/systems/<name>.md` | owned outright |
| `internal/ui/<name>_ui.go` | owned outright |
| `internal/game/<name>.go` | owned outright (test/demo content) |
| shared files (below) | append-only, one line |

Shared files — append-only registration points, never restructured on
a feature branch:

- `internal/ui/model.go` — core Model, layout, focus routing
- `internal/game/world.go` — `NewWorld` assembling per-area builders
- `docs/SYSTEMS.md` — status flips happen on main, not feature branches

Off-limits on a feature branch:

- `internal/engine/` — engine changes ship separately and first (below)
- another system's files, ever

## The litmus test

If your branch's diff touches files owned by another system, or
non-append lines in shared files: stop. Either the boundary is wrong
or the change belongs in a separate engine/integration PR. A reviewer
should be able to judge a PR's blast radius from its file list alone.

## Extending an existing system

- New files inside `internal/systems/<name>/`; spec grows in
  `docs/systems/<name>.md`; UI grows in `internal/ui/<name>_ui.go`.
- One agent per system at a time — the boundary unit is the system.
- Engine gaps: land the engine addition as its own small PR to main
  (addition + tests, nothing else), then rebase the feature branch on
  it. Never bury engine edits inside a feature branch.

## Adding a new system

1. Spec first: `docs/systems/<name>.md`.
2. New package `internal/systems/<name>/` — one interface + `New()`,
   never imports sibling systems (readme rules apply).
3. Footprint on shared files is append-only, one line each:
   - register its UI surface in `internal/ui/model.go`
   - add its builder to `internal/game/world.go` (if it needs content)
   - add its row to `docs/SYSTEMS.md` (on main)
4. Everything else it owns: package, spec, `<name>_ui.go`,
   `game/<name>.go`.

## Integration layer layout

The two former monoliths, split so ownership is per-file:

`internal/ui/` (one package, per-system files):

- `model.go` — Model struct, Init/Update/View shells, prompt and
  history, the surface registry and interfaces (shared, append-only)
- `styles.go` — layout constants and lipgloss styles (shared, rarely
  touched)
- `panels.go` — room view, right panel, LOG panel (core layout)
- `panel_city.go` — hubs travel panel (citySurface)
- `dialogue_ui.go`, `hacking_ui.go` — per-system surfaces
- `modals.go` — level-up / inventory / stats overlays (modalSurface)

`internal/game/` (one package, per-area files):

- `world.go` — `NewWorld` stitches the per-area builders; the only
  place hubs are declared and areas cross-reference (shared,
  append-only)
- `flags.go` — every story flag, declared once
- `lair.go`, `coffeeshop.go`, `plaza.go` — one file
  per area, each exposing `build<Area>() *engine.Entity`
- `net.go` — the hacking net content

## Namespaces

Systems integrate through shared state (blackboard pattern: one
`World`, systems read/write it, never each other). Git catches code
collisions; it cannot catch collisions in the *names* systems
communicate through. Three namespaces, three rules:

**Flags** — every flag has exactly one writer-owner. The owning system
exports it as a typed constant; raw flag strings never appear outside
that one declaration:

```go
// internal/systems/hacking/flags.go (owner declares)
const FlagHeardWhisper = "heard_whisper"

// internal/game/lair.go (wiring references — only game/ may
// reference another system's constants; systems still never
// import each other)
{Flag: hacking.FlagHeardWhisper, ...}
```

A typo or rename becomes a compile error in the wiring layer instead
of a silently-false flag. Flags owned by story content (not any
system) are declared once in `internal/game/flags.go` — today every
flag is content-owned and lives there; the system-owned form above
applies once a system's own mechanics (not content hooks) set a flag.

**Entity IDs** — an area file owns its IDs. Cross-area references
(exits between hubs, hub entry rooms) live only in `world.go`, where
the areas are stitched together.

**Verbs** — systems claim verbs from the parser directly, so constants
can't police them. Each system's spec (`docs/systems/<name>.md`) lists
the verbs it claims; review checks new claims against the other specs.

## Concurrency

The `World` is not thread-safe (bare maps, no locks) and must never
need to be: it is confined to the Bubble Tea update loop, the game's
single event thread. Turns.md guarantees nothing happens between
player actions, so no system may spawn a goroutine that touches the
World. If a feature ever needs background work (timers, animation),
the goroutine communicates only by sending a `tea.Msg` into the update
loop — never by holding a `*World`.

## UI surface registry

New systems must not require surgery on `model.go`'s Update/View. A
system UI implements `surface` (Active + HandleKey) in its own
`<name>_ui.go`, plus whichever optional capabilities it needs —
`fullscreenSurface` (Screen), `overlaySurface` (Overlay),
`resizableSurface` (Resize), `interceptorSurface` (Intercept, to
claim prompt verbs like "meow" before the engine sees them) — and
registers with one appended line in `model.go`:

```go
var surfaces = []surface{
    shellSurface{},    // hacking terminal (fullscreen)
    modalSurface{},    // level-up / inventory / stats
    dialogueSurface{}, // conversation overlay
    citySurface{},     // travel panel focus
}
```

Order is input priority: the first active surface owns the keyboard.
Overlays composite in reverse registry order, so earlier surfaces
render on top.
