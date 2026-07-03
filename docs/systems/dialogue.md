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
- apply effects such as setting/clearing flags, emitting prose, or
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
systems read the same flags. There is no separate NPC memory database in
the first slice.

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

- `dialogue.Talkable` — component attached to an NPC or terminal.
- `dialogue.Node` and `dialogue.Choice` — declarative graph data.
- `dialogue.Start` — opens a `Session` for UI/headless callers.
- `Session.Render`, `Session.Options`, `Session.Choose`, `Session.Done`
  — active conversation API. `Options` pairs each visible choice with
  its lock state and derived tag.
- Requirement helpers: `Flag`, `MissingFlag`, `HasItem`, `StatAtLeast`
  (`StatAtLeast` gates visibly; the others hide).
- Effect helpers: `SetFlag`, `ClearFlag`, `Say`, `AwardOnce`.

## Later

- NPC presence rules: entities appearing or moving as a function of flags.
- Larger memory vocabulary if repeated patterns exceed simple flags.
- Dialogue-driven objective/journal updates once the journal system
  exists.
