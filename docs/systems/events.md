# Events & Triggers

Status: agreed; first slice built (rules, modal, derived journal).

The overworld's IF-THEN machine: rules that make the world react to
game state. Generalizes the `engine.On` escape hatch (system #5 in
docs/SYSTEMS.md). NPC presence, quest journal, and story beats are
all planned as content on top of this one mechanism.

## Model

An event is a rule: **conditions** over world state, plus **effects**.

    IF  <flags are true/false>  AND  <Buddy just entered room/hub R>
    THEN  <print story text, set flags, move entities>  [once]

Conditions read only the World (flags, player position). Effects are
the same small palette content already uses everywhere else: narrate,
set a flag, re-home an entity. Nothing else — an effect that wants
mechanics belongs in a system, not an event.

## When rules are checked: poll, don't listen

Rules are evaluated **at the end of every player action** — the one
moment state can change (turns.md). There are no notifications, no
subscriptions, no `SetFlag()` indirection: rules re-read the World
each turn, the same way the room view and YOU SEE panel re-render
from state each frame.

Why polling: at this scale (hundreds of rules, boolean checks) the
cost is nothing, and rules judged against *state as it is* cannot
miss a change — a rule added later, or a world rebuilt from a save,
just works. Listener wiring would add engine machinery and a new
silent-failure class (state changed via a path that forgot to
notify).

## Declaration

Content declares rules like it declares everything else — literal
config, engine evaluates:

```go
engine.When{
    Flags:  []string{flagHeardWhisper},   // all must be true
    Unless: []string{flagWhisperSilenced},// all must be false
    Enter:  "lair",                       // fires on entering ("" = any action)
    Once:   "seen_lair_whisper",          // self-marking flag ("" = repeats)
    Do: func(w *engine.World) string {
        return "The deck's cursor blinks faster tonight."
    },
}
```

- `Enter` may name a room or a hub; entering a hub via travel counts
  as entering its entry room.
- `Once` is just a flag the rule sets when it fires and requires
  false to fire — so fired-ness serializes with the rest of the
  flags and save/load stays free.
- Rules live in a world-level list (`World.Rules` or similar), filled
  by `internal/game` — declared per area in that area's file.

## Presentation: the modal

Event output is a story beat, not a log line. When the end-of-turn
poll fires rules with output, the UI presents it in a centered modal
(the same overlay machinery as level-up/inventory), one beat per
modal, dismissed with enter/esc; multiple firings queue. The text
also lands in the LOG so the transcript stays complete. The modal is
an eventsSurface in `internal/ui/events_ui.go` + one registry line
(docs/BOUNDARIES.md).

## Journal

The journal (system #10) is the first consumer, and it is **derived,
not stored**: journal entries are declared content keyed to flags —

```go
events.Entry{Flag: flagHeardWhisper,
    Text: "Traced the whisper to sunfarm.arc. Someone buried
           something they call the sun."}
```

— and the journal view is every declared entry whose flag is true,
in declaration order. Nothing persists beyond flags the World
already holds; save/load is untouched; the journal can never be
stale because it is recomputed from state on open (same rule as the
YOU SEE panel). Viewed via a `journal` verb → modal.

## Ordering

Rules are checked in declaration order; every rule whose conditions
hold fires that turn, output in order. No priorities, no cancellation
— if two rules genuinely conflict, that is a content bug to fix in
content, not a scheduling feature to build.

## Boundaries (docs/BOUNDARIES.md)

- The mechanism lives in **engine core** (`engine/events.go`:
  `When`, `Entry`, `World.CheckEvents`, `World.JournalEntries`) —
  decided at build time because every action path must call the
  checkpoint and systems may not import a sibling system.
- The checkpoint runs inside `Engine.Execute`, `hubs.Travel`, and
  `dialogue.Pick`; the UI calls it once more on terminal logout.
  Rules are **not evaluated while the hacking terminal is open** —
  flags write live, the world reacts when Buddy stands up.
- Rules and journal entries are content: declared in
  `internal/game/<area>.go` next to the rooms they concern
  (`lairEvents`, `lairJournal`), appended in `world.go`, using flag
  constants from `flags.go`.
- UI: `internal/ui/events_ui.go` — `eventsSurface` (beat modal, fed
  from `World.Pending`) and `journalSurface` (`journal` verb) — plus
  two registry lines in `model.go`.
