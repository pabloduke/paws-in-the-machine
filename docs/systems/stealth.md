# Stealth, Checks & XP

Status: core model agreed; open questions listed at bottom.

## Scope: what rolls and what doesn't

Boundary checks (below) are the ONLY dice in the game. They cover the
cat talents — sneaking past observers, physical feats (leaps, ledges),
and charming humans — all resolved against one stat for now (see XP).
Hacking is the opposite by design: no Hacking stat, no "hack terminal"
command with a success chance. Hacking is performed by the player —
jacking in leads to cyberspace as real explorable rooms where the hack
happens through play (navigate, examine, take, manipulate). Getting
past ICE is a puzzle you solve, not a check you pass. Player skill,
not character skill.

## XP: the one stat (for now)

Buddy has a single visible stat: XP. Earned by actions (never by time),
and read by every check regardless of flavor — sneaking, leaping,
charming. Splitting into separate stats (Stealth/Agility/Charm, the
Deus Ex direction) is deliberately deferred; if the game later wants
build variety, the check mechanic just reads a different number and
content re-tags its checks. Until then: one number, less bookkeeping.
There is no Perception stat either — noticing things is handled
deterministically by the visibility system, not rolled.

## Model

A check happens at a boundary, not in a simulation. Guards, dogs, and
cameras do not patrol or run on a clock; they gate specific transitions
(an exit, or an action like "paw the keyboard in view of the counter").
Attempting a gated transition triggers one check:

    hidden roll + XP + situational modifiers  vs  difficulty

Succeed: the transition happens, narrated as the slip-past.
Fail: the transition is refused and the failure changes the situation
(see open questions) — it never simply says "try again."

## Determinism (the XCOM rule)

Checks are seeded: the outcome is a pure function of

    (world seed, check identity, XP + modifiers)

Repeating an identical attempt gives an identical result. Save-scumming
is not forbidden; it is useless. The roll changes only when the inputs
change: a better stat, a new modifier (distraction arranged, item used,
different lighting, different route), or a different check entirely.
Failure's message is always "change something."

The world seed is generated at new-game time and persists in world state
(and therefore in future save files).

## What the player sees

- Dice are never shown. No roll output, no target numbers, no "4d6".
- XP is visible on request (e.g. a `stats` command: "XP 10").
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

## Engine surface (planned)

- Seeded RNG + XP live in engine core world state.
- `internal/systems/checks/` package, one exposed surface: a `Check`
  function plus a `Guarded` component that content attaches to exits or
  entities (observer type, difficulty, modifier hooks as configuration).
  XP increases are one legitimate way to turn a failed check into a
  passable one (a changed input reseeds the roll).

## Open questions

1. Failure consequences, concretely: shooed away (soft), area alert flag
   raised (stateful), route burned (hard)? Probably varies by observer
   type — decide before implementation.
2. XP growth mechanism: story milestones? training with resistance
   mentors? per-use practice? (Must be action-based per turns.md.)
3. Difficulty scale and starting numbers (what does XP 10 mean against
   what difficulty range) — decide when implementing.
