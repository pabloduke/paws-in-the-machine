# Turns & Time

Status: spec agreed.

## Model

Strictly turn-based in the simplest sense: the world is frozen until the
player acts, and only player actions change state. There is no clock, no
tick, no scheduler, and no turn counter. The game is a pure state machine:

    state + executed command -> new state + story output

## No time-based anything

- No turn arithmetic: no lantern timers, hunger clocks, or doom counters.
  Progression lives in flags set by actions.
- No tick/daemon system: nothing "runs every turn." The world does not act
  behind the player's back.
- NPC positions are a function of game state, not simulation: an NPC is
  wherever the current flags say it is ("owner_away" false = corner table,
  true = restroom). Player actions flip the flags.
- Guard encounters are boundary checks, not patrols (see stealth.md).

## What counts as an action

Only commands that parse and resolve mutate state. Invalid input (unknown
verbs, unresolvable objects, empty lines) and meta commands (`help`,
`quit`) change nothing. A resolved command that fails in story terms
("take shelf" — it isn't portable) is still an action and may have
consequences, but state changes only through explicit effects, never
through the passage of turns.

## Consequences for the engine

Nothing to build. The current engine already works exactly this way; this
spec exists to record the decision and to constrain future systems:
any proposed mechanic that needs a clock or a per-turn tick is out of
line with this design and needs a state-based reformulation instead.
