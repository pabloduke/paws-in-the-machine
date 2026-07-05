# Presence

Status: agreed; built (engine.Placed + checkpoint ordering + barista
as first consumer).

Where story-owned entities are, as a function of world state. The
dialogue spec's rule — "positions are a function of game state
(flags), never simulated movement" — finally gets its mechanism.
Applies to any entity, not just characters: the barista, the hound,
a parked van, the ruby once a fence takes it.

## Model: position is derived, not stored

A story-owned entity declares its placement as a function of the
World (a Moore machine: output depends on current state only, never
on the path that reached it):

```go
barista := engine.NewEntity("barista", "the barista", ...).With(
    engine.Placed{Fn: func(w *engine.World) string {
        if w.Flags[flagBaristaSawShard] {
            return "backroom"
        }
        return "coffeeshop"
    }},
    ...
)
```

Nobody ever "moves" her. Same board, same position, always —
regardless of how the flags got that way. Consequences:

- **No drift.** Position cannot disagree with flags; the function is
  the truth. There is no rule to forget when a new flag path is
  added — the function is re-read every turn.
- **No transition explosion.** N story states need one function with
  N branches, not N² move rules.
- **Save/load stays free.** Placed entities' positions need no
  serializing: rebuild the world, positions recompute from flags.
- **Chess property.** The board (flags + positions) is the game; any
  reachable configuration is fully meaningful without its history.

## What presence is NOT for

- **Player-moved things.** A shard Buddy drops in the plaza has no
  formula; its position is genuinely mutable state (and is what
  save/load serializes). `Placed` entities are story-owned and must
  never be `Portable` — both on one entity is a content bug.
- **Narrating the change.** Placement is silent: entities are simply
  elsewhere the next time the board is consulted. When a change
  deserves witnessing ("the hound's chain rattles as it's led
  away"), that is an event rule (docs/systems/events.md) keyed to
  the same flags. Presence says where; events say what you saw.

## Mechanism

`engine.Placed{Fn func(*World) string}` — a component, engine core
(same reasoning as events: it must run inside the end-of-turn
checkpoint, and systems may not import siblings; expected ~30
lines). Returning `""` places the entity offstage (out of every
room; not visible, not in scope).

`CheckEvents` becomes the full end-of-turn sequence:

1. **Apply placements** — every `Placed` entity moves to where its
   function says (a no-op almost always), so the board is consistent
   before anything reads it.
2. **Detect arrival** (existing Enter logic).
3. **Poll event rules** (existing) — rules therefore see
   post-placement truth, and an Enter rule can react to who is in
   the room as Buddy arrives.

No new checkpoint call sites: placement rides the one that exists.

## Timing feel

Positions change only at checkpoints, i.e. only when the player
acts. An NPC never visibly walks; Buddy returns from the terminal or
another room and the arrangement is different — which reads as
"while you were busy," the correct texture for a cat's-eye city.
While Buddy stands still doing nothing, nobody moves (turns.md).

## Boundaries (docs/BOUNDARIES.md)

- Mechanism: engine core (`engine/presence.go` + the CheckEvents
  ordering change). Ships as its own engine commit.
- Placement functions are content: declared on entities in
  `internal/game/<area>.go`, reading flag constants from `flags.go`.
- No UI surface, no verbs. The YOU SEE panel and room view already
  re-render from state; they pick placement changes up for free.

## First consumer

The barista: `coffeeshop` → `backroom` once `barista_saw_shard`,
plus an event rule narrating the empty counter the next time Buddy
enters the coffee shop. Exercises placement, offstage never, arrival
interplay, and the events handoff.

## Rulings (were open questions)

- **Dialogue holds the stage.** `World.HoldPlacements` is set while a
  conversation is open and released when it ends (UI `endScene`);
  flags, rules, XP all stay live — only placement application waits,
  so an interlocutor can't vanish mid-sentence. An NPC that *should*
  walk out mid-scene does it by ending the dialogue as an effect.
- **Offstage is a holding entity**: `World.Offstage`, a child of
  Root created by `NewWorld` — out of every room, out of scope,
  invisible; entities return from it like from anywhere else.
