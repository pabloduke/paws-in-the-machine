# Hubs & Travel

Status: spec agreed.

## Model

The city is organized into hubs (districts): The Neighborhood, The
Plaza, Industrial Zone, Microslop HQ, ... Each hub contains explorable
rooms. In the world tree a hub is an ordinary entity:

    Root → hub → rooms → things

Buddy has lived in this city his whole life, so **every hub is known
and travelable from the start**. There are no locked or hidden hubs.
What game state gates is *relevance*: a hub can be visitable with
nothing story-relevant happening in it until flags say otherwise —
the same rule that governs NPC positions.

If a place must be physically inaccessible (Microslop HQ interior),
that is a Guarded obstacle or blocked exit *inside* the hub — never a
lock on the travel panel.

**Buildings deepen as the game does** (agreed 2026-07-07). Okuda HQ,
the first infiltration building, teaches "two ways in": getting inside
*is* the whole puzzle. Later buildings keep both doors but layer the
interior — badge-locked security doors, patrolled floors, keycards
lifted off desks — and the best of those locks live on the building's
own network, so a door can be opened from the terminal side (the
closed-ports flag contract, pointed the other direction). Getting in
is the early game; "in is only the start" is the late game.

## Travel UX: the side panel

A persistent side panel next to the transcript lists all hubs, marking
the one Buddy is in.

- **Tab** shifts focus between the command prompt and the panel.
- With the panel focused: **Up/Down** select a hub, **Enter** travels
  (Buddy arrives at that hub's entry room; the room description prints
  to the transcript), **Esc/Tab** returns to the prompt without
  traveling.
- Typing at the prompt never affects the panel; panel keys never reach
  the prompt.

Traveling is a player action like any other (turns.md): it changes
where Buddy is, story output narrates it, and nothing else moves.

## Engine surface

- `internal/systems/hubs` package: `Hub` component (config: `Entry`,
  the room ID where travel lands), `List` (all hubs), `Current` (hub
  containing the player), `Travel` (move + describe).
- Content declares a hub by attaching `hubs.Hub` to an entity that is
  a direct child of Root and adding its rooms as children.
