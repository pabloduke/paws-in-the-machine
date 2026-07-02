# Stealth & Detection

Status: core model agreed; open questions listed at bottom.

## Scope: what rolls and what doesn't

Stealth checks are the ONLY dice in the game. Sneaking is Buddy's innate
cat talent, so it's resolved by hidden rolls. Hacking is the opposite by
design: no Hacking stat, no "hack terminal" command with a success
chance. Hacking is performed by the player — jacking in leads to
cyberspace as real explorable rooms where the hack happens through play
(navigate, examine, take, manipulate). Getting past ICE is a puzzle you
solve, not a check you pass. Player skill, not character skill.

## Model

Stealth is a check at a boundary, not a simulation. Guards, dogs, and
cameras do not patrol or run on a clock; they gate specific transitions
(an exit, or an action like "paw the keyboard in view of the counter").
Attempting a gated transition triggers one check:

    hidden roll + stealth stat + situational modifiers  vs  difficulty

Succeed: the transition happens, narrated as the slip-past.
Fail: the transition is refused and the failure changes the situation
(see open questions) — it never simply says "try again."

## Determinism (the XCOM rule)

Checks are seeded: the outcome is a pure function of

    (world seed, check identity, effective stat + modifiers)

Repeating an identical attempt gives an identical result. Save-scumming
is not forbidden; it is useless. The roll changes only when the inputs
change: a better stat, a new modifier (distraction arranged, item used,
different lighting, different route), or a different check entirely.
Failure's message is always "change something."

The world seed is generated at new-game time and persists in world state
(and therefore in future save files).

## What the player sees

- Dice are never shown. No roll output, no target numbers, no "4d6".
- Buddy's stealth stat is visible on request (e.g. `stats`: "Stealth 10").
- Circumstances are telegraphed in prose, not numbers: "the dog is
  half-asleep" vs "the dog's ears are up" carries the difficulty; "your
  fur disappears in the neon wash" signals a favorable modifier.

## Modifiers (the flavor lives here)

Situational, composable, content-defined. Examples:

- Neon-lit rooms favor Buddy's orange fur (camouflage bonus).
- Carrying a human object penalizes: a cat carrying nothing is invisible;
  a cat carrying a keycard is a story.
- A prepared distraction (crow diversion, a mug shattering in the next
  room) grants a bonus, usually consumed on use.
- Observer type matters — different senses, different counters:
  - Humans: lowest difficulty; human perception filters cats out unless
    Buddy is doing something visibly impossible-for-a-cat.
  - Dogs: highest difficulty; smell defeats visual camouflage (no neon
    bonus); a failed check escalates — dogs report up the hierarchy.
  - Cameras: pattern-matching; visual modifiers apply, scent does not;
    the most predictable observer — closest to a pure puzzle.

## Stats & growth

Buddy's stealth stat is a visible number (starting point: Stealth 10).
It can improve over the game, and a stat increase is one legitimate way
to turn a failed check into a passable one. Growth must follow the
turns.md rule: earned by actions, not by time. Mechanism TBD below.

## Engine surface (planned)

- Seeded RNG + player stats live in engine core world state.
- `internal/systems/stealth/` package, one exposed surface: a `Check`
  function plus a `Guarded` component that content attaches to exits or
  entities (observer type, difficulty, modifier hooks as configuration).

## Open questions

1. Failure consequences, concretely: shooed away (soft), area alert flag
   raised (stateful), route burned (hard)? Probably varies by observer
   type — decide before implementation.
2. Stat growth mechanism: story milestones? training with resistance
   mentors? per-use practice? (Must be action-based.)
3. Difficulty scale and starting numbers (what does Stealth 10 mean
   against what difficulty range) — decide when implementing.
