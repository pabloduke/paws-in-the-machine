# Project State

Current branch: `main`

> **Lane handoff (2026-07-06, amended 2026-07-07).** Until now the
> repo ran two lanes: Codex on the terminal/hacking lane, Claude on
> the overworld lane (engine, checks, dialogue/presence/events
> content, room UI). Claude is off the project; Codex owns both
> lanes. The overworld's state, and the rulings it must keep, are in
> "Overworld lane" below. Design truth lives in `docs/systems/*.md`
> (each system's spec, statuses current), `docs/SYSTEMS.md` (the
> build-order index), and `docs/BOUNDARIES.md` (ownership and import
> rules).
>
> **Bonus Claude day (2026-07-07), all merged to main:** #16
> closed-ports loop built as Okuda HQ, the first infiltration
> building and the first overworld↔terminal handshake (PR #21);
> exits-line flicker fix (PR #30); the deck now lives in the lair —
> hacking starts at home (PR #31); the PDA, a read-only menu-driven
> field device — check notes + scan ports, never a shell (PR #32);
> parser learned jump/climb/hide. New rulings are marked "user
> ruling 2026-07-07" where they live (hubs.md: buildings deepen;
> hacking.md: deck-at-home + PDA). The road to the real game is
> issue-tracked as #22–#29 (see "Road to the real game" below).

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
- Buddy's deck lives in the lair and nowhere else (user ruling
  2026-07-07, reversing the earlier carried-deck change): hacking
  starts at home. The loop is scan from the lair → do the physical
  work in the world → come home and jack in. `TestDeckOnlyReachableInLair`
  and `TestDeckStaysHome` pin it.
- Buddy carries a **PDA** for field intel (user ruling 2026-07-07):
  read-only, menu-driven, never a shell — *Check notes* (deck's
  `.md`/`.txt` mirror in a reader) and *Scan ports* (per-host `scan`
  report). It shares the deck's net map (`hacking.PDA`, built once in
  `lair.go`), so it sees copied files and port flips live — scan
  okuda.grid from the field, see it open, go home to hack. UI is a
  centered overlay (`internal/ui/pda_ui.go`), overworld styles.
  Tests: `pda_test.go` (game), `pda_ui_test.go` (UI).
- SSH hosts can require a route flag and/or a fake password.
- Hosts can declare fake services/ports, `scan <host>` lists configured port
  states, and `ssh <host> -p <port>` targets SSH on a specific port.
- `grep -ir` searches recursively and case-insensitively through fake host
  filesystems.
- The hacking terminal has a terminal-only green phosphor CRT look.
- The terminal right panel is modal: quest/status by default, swapped
  for the reader, the editor, or the messenger — the deck's IM
  (`messenger`/`talk`), the Resistance-lite mission channel. Messages
  arrive on flags, never timers (docs/systems/hacking.md).
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
- **#16 closed-ports loop is built** (`internal/game/okuda.go` +
  `net.go`): Okuda HQ, the first infiltration building, two ways in —
  front-desk charm (fail forks to `receptionist_suspicious` −2; the
  missing-cat flyer angle, `played_lost_cat` +6, recovers) or
  fire-escape parkour + corridor sneak (breaker panel: lights-out +6,
  consumed). Both converge on the records-office console, which sets
  `okuda_port_open`; `okuda.grid`'s SSH service reads it via
  `OpenWhen`, so scan flips closed→open and ssh connects. First
  overworld↔terminal handshake, flag referenced only in
  `internal/game/` per BOUNDARIES. `okuda_demo_test.go` walks both
  routes and the handshake on the pinned seed. Companion ruling in
  `docs/systems/hubs.md`: buildings deepen later (interior security
  doors, terminal-openable locks); Okuda stays tutorial-sized.
- **#25 flag-gated doors are built** (`engine.Exits.Gated` +
  `engine.Gate{Flag, Shut}`): a locked direction is a lock, not an
  obstacle — no dice, shut prose until its flag is true (`Guarded`
  stays for things you attempt). First use: the Okuda records office's
  archive door, opened only by `run /srv/ctl/unlock.bin` on
  `okuda.grid` — the closed-ports contract pointed the other way
  (hack the net, open a physical door), leading to the Deep Archive
  annex (story stub: the missing aisle between 409 and 411).
  `TestGatedExit` (engine) + `TestOkudaArchiveDoorOpensFromTheNet`
  (full loop) pin it.

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
- **Seed policy (unpinned 2026-07-08)**: production seed is random —
  `engine.NewWorld` sets `Seed = rand.Int64()` and `game.NewWorld` no
  longer overrides it, so every playthrough rolls its own table and the
  seed persists through save/load (the XCOM rule stays honest,
  save-scumming stays useless). The demo *tuning* now lives in the
  tests that assert it: `checks_demo_test.go`, `okuda_demo_test.go`,
  and `TestMicroslopPasswordPuzzle` each set `w.Seed = 3` after
  `NewWorld`. **Any test asserting a check outcome must pin the seed**
  — an unpinned check test is flaky by construction. Starting stats
  (Stealth 10 / Agility 12 / Charm 8) and difficulties are still the
  demo-tuned values; recalibrating them for the real game is the
  remaining half of #27.
- **Systems never import each other**; `internal/game` is the
  composition root and the only cross-system wiring point.

### Open overworld issues (design intent in each issue)

- **#14 NPC memory** — PATTERN BUILT (worked example, not yet stamped
  across every NPC). It's flag conventions over dialogue/description
  that already exist, not a new system. The barista now remembers first
  contact on two per-NPC flags — `barista_softened` (warm) and
  `barista_burned` (cold, from knocking her tip jar over) — read by her
  greeting (via the new `dialogue.Node.TextFn`, the memory-aware line),
  her `Description{Fn}`, and the hound-lure favor she won't do for a
  cruel stray. Kept off the critical path: burning her never walls the
  `apple` reveal (failure is a fork, not a wall). Presence-withdrawal is
  the pattern's fourth lever but is for non-critical NPCs only — moving
  a critical NPC offstage would wall progression. Spec:
  `docs/systems/dialogue.md` "State And Memory"; tests:
  `coffeeshop_memory_test.go` (game). Replicating it across NPCs is
  content (user-owned).
- **#15 UI growth: status bar** — independent, any time.
- **#16 closed-ports loop** — BUILT (see Implemented above). By user
  ruling it grew past the issue's rack sketch: opening the port is a
  full infiltration of Okuda HQ, not a switch in the coffee shop back
  room. Still owed: what okuda.grid's vault actually holds —
  `/srv/vault/410.txt` is a marked story stub.

## Road to the real game (issues filed 2026-07-07)

The remaining gap between "systems demo" and "content production can
start" is fully issue-tracked; design intent lives in each issue.

Still open:

- **#22 ICE** — the last big undesigned mechanism (puzzle, never a
  check; spec first).
- **#38 AI sentinels** — the high tier that sits on top of #22:
  traced intrusion + a prompt-injection exploit line, pretend-AI voice
  over a deterministic state machine (no runtime model). Design #22 and
  #38 together; spec first.
- **#23 richer credential gates** — keyfiles, chains, alternate
  Microslop password paths (crack.bin rejected as gamey).
- **#27 difficulty calibration + unpin seed 3** — seed UNPINNED
  (2026-07-08): production randomizes, tests pin. Remaining half is
  difficulty/starting-stat calibration; release blocker.
- **#28 terminal rulings** — history persistence still open;
  right-panel content answered by the messenger (2026-07-09).
- **#29 Towers of Hanoi** — the backup-rotation shrine. It wouldn't
  be a puzzle game without it (user ruling).
- **#34 Resistance: quests, ranks, deck mail** — parked design; a
  scope question (decide in-or-out before it balloons the first
  content push).

Closed since this list was filed:

- **#24 hacking XP payouts** — BUILT (netEvents, `flags.go` XP markers).
- **#25 flag-gated doors/exits** — BUILT (see Overworld Implemented:
  `engine.Exits.Gated` + the Okuda archive door).
- **#26 dialogue headless dead end** — FIXED: `Talkable.Handle` now
  refuses honestly and points at `dialogue.Start`; headless drivers use
  the `Session` API.

After the open ones: content production — the city map, act one, and
replacing every `(Placeholder)` — which the user writes.
