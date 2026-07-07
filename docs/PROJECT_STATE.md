# Project State

Current branch: `terminal`

> **Lane handoff (2026-07-06).** Until now the repo ran two lanes:
> Codex on the terminal/hacking lane, Claude on the overworld lane
> (engine, checks, dialogue/presence/events content, room UI). Claude
> is off the project; Codex owns both lanes. The overworld's state,
> and the rulings it must keep, are in "Overworld lane" below. Design
> truth lives in `docs/systems/*.md` (each system's spec, statuses
> current), `docs/SYSTEMS.md` (the build-order index), and
> `docs/BOUNDARIES.md` (ownership and import rules).

## Current Focus

The game is building toward a cat-hacker loop where Buddy uses normal room
actions, dialogue, stealth, and a fake terminal shell together. The current
beat is the first Microslop Corp terminal puzzle.

## Implemented

- Dedicated hacking terminal mode exists and replaces the normal room UI while
  active.
- The terminal has a fake shell with simplified commands including `ls`, `cd`,
  `pwd`, `cat`, `grep`, `cp`, `mkdir`, `touch`, `scan`, `ssh`, `curl`, `ps`,
  `kill`, `run`, `exit`, and `help`.
- Buddy's deck is now carried, so `use deck` / `log in` works outside the lair.
- SSH hosts can require a route flag and/or a fake password.
- Hosts can declare fake services/ports, `scan <host>` lists configured port
  states, and `ssh <host> -p <port>` targets SSH on a specific port.
- `grep -ir` searches recursively and case-insensitively through fake host
  filesystems.
- The hacking terminal has a terminal-only green phosphor CRT look.
- The terminal right panel is intentionally reserved but blank for now.
- Buddy's notes live at `~/notes/notes.md`, generated from discovered flags
  and readable/searchable with shell commands instead of a journal menu.
- Microslop is currently gated by:
  - social intel from the barista, who was fired from Microslop and reveals the
    password `apple`;
  - physical access in the coffee shop back room, reached by bypassing the
    corpo hound;
  - using the backroom server rack to patch the deck into the local network.
- Once connected, Microslop has a small fake filesystem with logs pointing to
  `/srv/archive/sun_notice.txt`.
- Reading and copying the Microslop notice set flags that feed Buddy's
  generated notes.

## Recent Verification

`go test ./...` passes after the Microslop route/password flow changes.

## Next Useful Steps

- Make the social-engineering path less linear: add alternate ways to learn or
  infer `apple`.
- Add clearer in-game hints when `ssh microslop` reports network unreachable.
- Give the backroom/rack interaction a stronger stealth consequence or tension.
- Expand Microslop's filesystem into a real mini puzzle instead of one log and
  one archive file.
- Decide whether terminal history should persist across deck sessions or
  reset per login.

## Overworld lane

### Implemented (all merged to main, all tested)

- **Checks system is complete** (`internal/systems/checks/`,
  `docs/systems/stealth.md`): approach matrix (sneak/parkour/charm),
  hidden seeded rolls, XP economy (`difficulty − effective stat`,
  floor 1; level n→n+1 costs 5×n), circumstance modifiers
  (`Mod{If, Unless, Delta, Consume}`), failure forks
  (`OnFail`/`Seals`), and observer perception
  (`Watcher`/`Oblivious`/`Unwatched`). Only open design question:
  difficulty-scale calibration — a tuning pass, not a mechanism.
- **Presence** (`engine.Placed`): positions are a pure function of
  flags, applied at the end-of-turn checkpoint. Never simulate
  movement.
- **Events + journal**: `When` rules polled after every action;
  journal entries derived from flags, never stored.
- **Overworld UI**: neon truecolor restyle, animated rain
  (100ms `tea.Tick`, per-drop staggered speeds), neon sign flicker,
  `[` / `]` rain opacity (level 0 = off).
- Demo content on the pinned seed: the coffeeshop hound exercises the
  whole checks system (fail → `hound_alerted` −2 → mug distraction +4
  consumed → clears; or barista lures the hound, `hound_lured`,
  unwatched +10). `internal/game/checks_demo_test.go` walks it.

### Rulings to preserve (agreed with the user; don't regress)

- **turns.md**: the World is a pure state machine — no timers, ticks,
  or counters touch game state. The single sanctioned exception is
  the UI rain tick, which may only advance `Model.phase`
  (presentation); `TestRainTickIsPresentationOnly` enforces it.
- **XCOM rule**: a check's outcome is a pure function of
  (seed, check id, effective stat). The effective stat feeds the
  hash, so any changed circumstance is a genuinely new roll.
  Save-scumming stays useless.
- **Failure is a fork, not a wall**: never a bare "try again."
  Dice, difficulties, and deltas are never shown; prose telegraphs
  circumstances.
- **Style namespaces**: `internal/ui/styles.go` is overworld-only;
  `internal/ui/styles_terminal.go` holds the `term*` styles and is
  the only style source `hacking_ui.go` consumes. New terminal
  styles belong in `styles_terminal.go`.
- **Flags**: declared once in `internal/game/flags.go`, one
  writer-owner each, raw flag strings nowhere else
  (`docs/BOUNDARIES.md`).
- **Pinned seed**: `w.Seed = 3` in `internal/game/world.go` tunes the
  hound demo numbers. Changing seed, starting stats, difficulties, or
  deltas will fail `checks_demo_test.go` — that's the test doing its
  job; retune them together (and remove the pin for release).
- **Systems never import each other**; `internal/game` is the
  composition root and the only cross-system wiring point.

### Open overworld issues (design intent in each issue)

- **#14 NPC memory** — cheapest next step: it's flag conventions over
  dialogue/events/checks that already exist, not a new system.
- **#15 UI growth: status bar** — independent, any time.
- **#16 closed-ports loop** — now UNBLOCKED by the terminal merge:
  the first overworld↔terminal crossover beat (scan shows a closed
  port → backroom rack action in the overworld → port opens).
