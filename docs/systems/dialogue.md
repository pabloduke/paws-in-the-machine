# Dialogue

Status: first playable slice built; NPC presence and broader memory still open.

## Model

Dialogue is a state-based node graph attached to an entity with a
`dialogue.Talkable` component. The parser only recognizes `talk <npc>`;
the dialogue system owns the conversation state after that: which node is
active, which responses are visible, what effects apply, and whether the
conversation has ended.

The player talks by choosing numbered responses. There is no free-form
natural language parser inside dialogue. That keeps authoring explicit
and testable.

## Choices

Each node contains NPC text and zero or more choices. A choice may:

- require flags, missing flags, carried items, or stat thresholds;
- apply effects such as setting flags, emitting prose, or
  awarding XP once;
- move to another node;
- end the conversation.

Availability follows a Mass Effect / New Vegas hybrid (ruled July 2026):

- **Stat-gated choices are visible but locked.** A choice requiring
  `StatAtLeast("charm", 8)` always renders, prefixed with a derived
  `[Charm 8]` tag (FNV-style; the tag comes from the requirement, never
  hand-written into choice text). While Buddy's stat is below the bar the
  choice is locked — picking it prints a refusal and applies nothing.
  Locks are re-evaluated every render, so training Charm to 8 unlocks
  the option on the next visit (ME-style retroactive unlock). No
  attempt-and-fail, no lockout flags.
- **Knowledge-gated choices stay hidden.** Flag, missing-flag, item, and
  once-only requirements hide the choice entirely, as before. If Buddy
  cannot ask about the data-shard because he is not carrying it, the
  option is absent.
- **One-chance moments are authored, not systemic.** Scarcity comes from
  flags moving the conversation past a beat, the same way Mass Effect's
  story moments pass — the system itself never consumes a check.

## State And Memory

Dialogue memory is ordinary world state. Choices set flags like
`barista_softened`; later nodes, room descriptions, checks, and other
systems read the same flags. There is no separate NPC memory database.

**NPC memory (#14) is a content pattern, not a new mechanism.** An NPC
remembers Buddy through a small set of per-NPC flags — one or two facts
that matter, not a database — written by dialogue effects and read back
in three safe places:

- **The greeting line**, via `Node.TextFn`: the opening runs warm, cold,
  or neutral on what she remembers. `TextFn` mirrors
  `engine.Description{Fn}` — when set it overrides `Text` and derives the
  line from world state at render time.
- **The NPC's `Description{Fn}`**: her body language reflects the same
  facts.
- **Choice gates**: a `MissingFlag` on a favor she won't do for someone
  who wronged her.

There is deliberately no reputation *meter* — dice, deltas, and hidden
scalars are never shown (docs/systems/stealth.md), so disposition is
remembered *facts*, not a number. Presence (`Placed`) is a fourth lever
— an NPC can withdraw when soured — but never on a critical-path NPC,
where vanishing would wall progression (failure is a fork, not a wall).
The worked example is the coffee-shop barista: `barista_softened` (warm)
and `barista_burned` (cold, from knocking her tip jar over) drive her
greeting, her description, and whether she'll call the hound off — but
never the `apple` reveal, so cruelty costs goodwill, not the route.

Once-only beats use flags too. A one-time XP award is guarded by an XP
flag; a once-only response can be guarded by `Choice.OnceFlag`.

## UI Contract

Typing `talk <npc>` enters dialogue mode: a centered modal opens over
the main row (NPC line on top, choice menu below) and closes when the
conversation ends. Up/Down
and number keys `1`-`9` move the highlighted choice, `Enter` commits
to it, and `Esc` walks away. Locked stat-gated choices render
dimmed with their tag and a ✗. The LOG keeps the transcript of the
exchange. Normal prompt input and hub-panel focus are suspended until
the conversation ends.

## Engine Surface

- `dialogue.Talkable` — component attached to an NPC or terminal. It is
  data only: conversations are interactive, so they run through
  `Start`/`Session` (the UI intercepts `talk` after applying
  `World.Rewrite`), never through engine command dispatch. The one verb
  `Handle` claims is `talk`, and only to refuse honestly — without it
  the engine default would claim the NPC "has nothing to say", a lie
  when a graph is attached (#26). Headless drivers (tests, tools) must
  use `Start`/`Session` like the UI does.
- `dialogue.Node` and `dialogue.Choice` — declarative graph data.
- `dialogue.Start` — opens a `Session` for UI/headless callers.
- `Session.Render`, `Session.Options`, `Session.Choose`, `Session.Done`
  — active conversation API. `Options` pairs each visible choice with
  its lock state and derived tag.
- Requirement helpers: `Flag`, `MissingFlag`, `HasItem`, `StatAtLeast`
  (`StatAtLeast` gates visibly; the others hide).
- Effect helpers: `SetFlag`, `Say`, `AwardOnce`. (`ClearFlag` will
  return when a node actually needs to unset state.)

## Later

- NPC presence rules: entities appearing or moving as a function of flags.
- Larger memory vocabulary if repeated patterns exceed simple flags.
- Dialogue-driven notes updates once the content needs them.
