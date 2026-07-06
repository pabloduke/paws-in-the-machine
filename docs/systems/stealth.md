# Stats, Checks & Obstacles

Status: agreed; built (modifiers and failure consequences included).
Remaining: difficulty-scale calibration (a tuning pass, not a
mechanism).

## The stat rule: a stat must be a verb

A stat earns its place only if it is something Buddy *does to an
obstacle* with a direct, visible effect. Three stats qualify:

- **Stealth** — sneak past. Not being noticed.
- **Agility** — parkour around. Leaps, ledges, vents, physical feats.
- **Charm** — charm. Working humans face-to-face: purring, adopt-me
  eyes, getting picked up and carried through a locked door.

Rejected by the rule:

- **Perception** — doesn't act on an obstacle; it multiplies content
  branches (things you can/can't see), which is an authoring decision,
  not a dice decision. Noticing things is deterministic (visibility
  system).
- **Hacking** — deliberately statless. The player performs hacks through
  play at the deck's terminal; ICE is a puzzle you solve, not a check
  you pass. Player skill, not character skill.

## XP, levels, and stat points

- **Earning.** A successful check pays XP equal to the roll you needed:
  `difficulty − stat`, min 1. Needing a 20 pays ~20; needing a 2 pays 2.
  Self-balancing: as stats grow, old obstacles pay less, pushing the
  player toward harder targets. Successful hacks pay XP by complexity —
  content assigns the amount, since hacks are played, not rolled. Story
  beats and discoveries may award one-time XP (flag-guarded). All
  earning follows the turns.md rule: actions, never time. Bypass flags
  prevent re-roll farming.
- **Leveling.** XP fills a level track with escalating costs: level
  n → n+1 costs `5 × n` XP (L1→2 = 5, L2→3 = 10, ...). Each level-up
  grants one stat point.
- **Spending.** `train <stat>` spends one point to raise a stat by one.
  Stats are roll modifiers, so training directly re-seeds previously
  failed checks.
- Level, XP progress, stats, and unspent points are visible via `stats`.

## Obstacles declare an approach matrix

An obstacle (guard, dog, camera, gap, locked-counter human...) declares
which approaches are possible and how hard each is — per-approach
difficulty and flavor text. Approaches not in the matrix are impossible
and refuse without a roll, with flavor ("the hound stares through your
adopt-me eyes; dogs are immune to cute").

    corpo hound:   sneak 12   parkour 15   charm —
    bored barista: sneak 8    parkour 10   charm 6
    ceiling gap:   sneak —    parkour 13   charm —

The player picks the approach by picking the verb: `sneak past the
hound`, `parkour around the hound`, `charm the barista`. One obstacle,
up to three doors through it, each rolled separately.

## The check

    hidden roll + stat + situational modifiers  vs  difficulty

Succeed: the transition happens, narrated as prose.
Fail: refused, and the failure should change the situation — never a
bare "try again" (see open questions).

## Determinism (the XCOM rule)

Checks are seeded: the outcome is a pure function of

    (world seed, check identity, stat + modifiers)

Repeating an identical attempt gives an identical result. Save-scumming
is not forbidden; it is useless. The roll changes only when the inputs
change: a raised stat, a new modifier, or a different approach. Failure's
message is always "change something." The world seed is generated at
new-game time and persists in world state (and future save files).

## What the player sees

- Dice are never shown. No roll output, no target numbers, no "4d6".
- Stats and XP are visible on request (`stats`).
- Difficulty and modifiers are telegraphed in prose, not numbers: "the
  hound is half-asleep" vs "ears up"; "your fur disappears in the neon
  wash."

## Modifiers (the flavor lives here)

Situational, composable, content-defined — and declarative: an
`Attempt` carries `Mods []Mod`, where a `Mod` is flag conditions
(`If`/`Unless`, same shape as event rules) plus a `Delta` on the
roll. The effective check is `roll + stat + Σ(applicable deltas) vs
difficulty`; the effective stat feeds the roll hash, so any changed
circumstance is a genuinely new roll (XCOM rule preserved), and the
XP award shrinks by the help you had (`difficulty − effective`,
floor 1 — a cheesed check pays less). A `Mod` may name a `Consume`
flag: spent the moment the roll uses it, pass or fail (a distraction
is used when you move on it).

Examples:

- Neon-lit rooms favor Buddy's orange fur (Stealth bonus).
- Carrying a human object penalizes Stealth: a cat carrying nothing is
  invisible; a cat carrying a keycard is a story.
- A prepared distraction (crow diversion, a mug shattering next door)
  grants a bonus, usually consumed on use.
- Observer type shapes the matrix and the modifiers:
  - Humans: charmable; perception filters cats out (low sneak
    difficulty) unless Buddy is doing something impossible-for-a-cat.
  - Dogs: never charmable; smell defeats visual camouflage (no neon
    bonus); failures escalate — dogs report up the hierarchy.
  - Cameras: pattern-matching; visual modifiers apply, scent doesn't;
    the most puzzle-like observer.

## Engine surface

- Stats, XP, and the world seed live in engine core world state.
- `internal/systems/checks/` package, one exposed surface: a `Check`
  function plus a `Guarded` component content attaches to obstacle
  entities (the approach matrix as configuration).

## Failure is a fork, not a wall

A failed check is a story state the world reacts to, never a bare
"try again." One mechanism covers every severity:

- An `Attempt` declares `OnFail []string` — flags set when it fails.
  Everything downstream is ordinary blackboard content: an event
  beat narrates the alert, a journal entry records it, a negative
  `Mod` on later attempts makes the alerted world genuinely harder,
  dialogue and presence can branch on it. Failure changes the
  inputs, so the XCOM rule's message ("change something") is
  enforced by the mechanics, not just the prose.
- **Soft**: no `OnFail` — refusal prose only.
- **Stateful**: `OnFail` sets an alert flag; content wires it into
  mods/events/dialogue (dogs escalate this way).
- **Hard**: `Seals` names a flag that closes the approach for good
  (route burned), refusing with `SealedText` — no roll, no XP, no
  consumption.

This is cheap branching: one flag per interesting failure, picked up
independently by every system that reads flags. Not every failure
needs a flag — content spends them only where failure is
interesting.

Demo beat (placeholder content): failing the corpo hound sets
`hound_alerted` (−2 on later attempts, an event beat, a journal
entry); knocking the espresso machine shatters a mug
(`mug_shattered`, +4, consumed by the roll it covers). Fail → knock
→ sneak clears.

## Open questions

1. Difficulty scale calibration (what Stealth 10 means against what
   range).
