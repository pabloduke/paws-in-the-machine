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

Unavailable choices are hidden, not greyed out. If Buddy cannot ask about
the data-shard because he is not carrying it, the option is absent.

## State And Memory

Dialogue memory is ordinary world state. Choices set flags like
`barista_softened`; later nodes, room descriptions, checks, and other
systems read the same flags. There is no separate NPC memory database in
the first slice.

Once-only beats use flags too. A one-time XP award is guarded by an XP
flag; a once-only response can be guarded by `Choice.OnceFlag`.

## UI Contract

Typing `talk <npc>` enters dialogue mode. While dialogue is active, number
keys select visible responses and `Esc` exits the conversation. Normal
prompt input and hub-panel focus are suspended until the conversation
ends.

## Engine Surface

- `dialogue.Talkable` — component attached to an NPC or terminal.
- `dialogue.Node` and `dialogue.Choice` — declarative graph data.
- `dialogue.Start` — opens a `Session` for UI/headless callers.
- `Session.Render`, `Session.Choices`, `Session.Choose`, `Session.Done`
  — active conversation API.
- Requirement helpers: `Flag`, `MissingFlag`, `HasItem`, `StatAtLeast`.
- Effect helpers: `SetFlag`, `ClearFlag`, `Say`, `AwardOnce`.

## Later

- NPC presence rules: entities appearing or moving as a function of flags.
- Larger memory vocabulary if repeated patterns exceed simple flags.
- Dialogue-driven objective/journal updates once the journal system
  exists.
