# Save / Load

Status: agreed; built (engine.Snapshot/Apply + save/load verbs).

The payoff for every architecture decision so far: because truth is
state and everything else is a function of it, a save is small and a
load is "rebuild the world from code, then restore the state."

## What saves (the chess position)

- **Flags** — the whole story state, including event Once-markers and
  journal visibility (both are just flags).
- **Character numbers** — Stats, XP, Level, StatPoints, and Seed (the
  XCOM rule must survive a reload: no save-scumming a failed check).
- **Positions** — parent ID for every entity in the tree: where the
  player stands, what Buddy carries, what got dropped where.

## What does NOT save (recomputed or rebuilt)

- **Content** — rooms, components, dialogue graphs, the net: rebuilt
  by `game.NewWorld()` from code, as always.
- **Placed entities** — presence positions are functions of flags;
  saved positions for them are ignored and re-derived on load.
- **Derived views** — journal, YOU SEE, objectives: functions of the
  above.
- **Session state** — open terminal/dialogue/modals: saving mid-scene
  is not a thing; save is a prompt verb, available only at the room
  prompt (same reasoning as the checkpoint deferrals).

Known v1 gap: in-memory net mutations that aren't flag-backed
(files copied onto the deck, killed process rows, `mkdir`/`touch`
debris) rebuild fresh. Progression is unaffected — hooks set flags
and flags save — but the deck's DISCOVERIES list resets. Revisit if
it ever matters to a puzzle.

## Mechanism

Engine stays file-blind (testable, no I/O in core):

- `engine.Snapshot(w) SaveState` — flags, numbers, positions.
- `SaveState.Apply(w) `— set flags/numbers, re-home entities by ID
  (skipping Placed ones), re-derive placements, reset arrival
  tracking so loading never fires Enter rules.
- `SaveState` marshals as versioned JSON (stdlib only).

File I/O lives at the edge: the UI intercepts `save` and `load`
prompt verbs and reads/writes one slot at
`os.UserConfigDir()/pawsinthemachine/save.json`
(`Model.SavePath`, overridable). Headless engine path gets plain
text confirmations.

Loading applies onto the running world: positions cover every
entity, flags replace wholesale, so a dirty world is fully
overwritten — no restart required.

## Boundaries (docs/BOUNDARIES.md)

- `engine/save.go` + tests — its own engine commit.
- `internal/ui/save_ui.go` — intercept-only surface + registry line.
- Parser verbs `save` / `load`.
- No content changes needed at all — the strongest evidence the
  architecture earned this system cheaply.


## Authored playtests (2026-09-06)

User-ruled: authored-world sessions start fresh. Their `save` and `load` commands
explain that saving/loading is disabled and perform no file access or state
change. Restarting and reselecting the world reads saved editor content again.
The built-in game retains its existing save slot and snapshot format.
