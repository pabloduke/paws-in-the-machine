# Systems

The build order for PAWS_IN_THE_MACHINE: get every system working before
writing story content. Story beats then compose systems that already exist.

Each system is designed spec-first (`docs/systems/<name>.md`), then built as
one package under `internal/systems/<name>/` — see Architecture in the readme.

## Status legend

- ✅ working
- 📝 spec in progress
- ⬜ not started

## The list (dependency order)

### Working

1. ✅ **Core loop** — parser → intent → component dispatch → story output.
   Lives in `internal/engine`.
2. ✅ **World model** — entity tree (composite), components (composable
   behavior), flags, rooms/exits, inventory. Lives in `internal/engine`.

### Foundational (other systems sit on these)

3. ✅ **Turns & time** — spec'd, nothing to build: pure state machine, no
   clock, no ticks, no counters. Only player actions change state.
   See `systems/turns.md`.
4. ✅ **Visibility & scope** (player side; observer-side perception for
   stealth modifiers still later) — invisible means physically enclosed:
   scope stops at closed containers (Openable); labels always reflect
   true state (Aspect); observer verbs never mutate. YOU SEE panel +
   `World.Visible()`. See `systems/visibility.md`.
5. ✅ **Events/triggers** — conditional events ("when flag X and Buddy
   enters Y"), all state-based per turns.md: `When` rules polled at
   the end of every player action, presented as story-beat modals.
   See `systems/events.md`.

### Headline systems

6. ✅ **Stats/checks/obstacles** (skeleton; XP earn/spend, failure
   consequences, and modifiers still open) — the core mechanic: nobody
   suspects a
   cat. Three stats, each a verb (Stealth=sneak past, Agility=parkour
   around, Charm=charm); XP is the growth currency spent to raise them.
   Obstacles declare an approach matrix (per-approach difficulty, or
   impossible). Hidden seeded rolls, XCOM rule (identical attempt,
   identical result), prose-telegraphed circumstances. The only dice in
   the game. See `systems/stealth.md`.
7. 📝 **NPCs/dialogue** — first dialogue slice is built: `talk <npc>`
   enters a numbered node graph with visible-choice requirements and
   flag/XP effects. NPC presence and broader memory are still open.
   Positions are a function of game state (flags), never simulated
   movement. A dog = NPC component + observer configuration on one
   entity. See `systems/dialogue.md`.
8. 📝 **Hacking** — first slice built: logging into the deck opens a full-screen
   terminal (shell over in-memory hosts/files/processes: ls, cd, cat,
   grep, cp, ssh, curl, ps, kill, run) with a quest panel. Buddy stays
   at the deck the whole time — it's a terminal session. Hooks set flags.
   No Hacking stat, no rolls: the player performs the hack through
   play; ICE is puzzle, not check — ICE, credential gates, and XP
   payouts still open. See `systems/hacking.md`.
9. ✅ **Hubs & travel** — city topology: hubs (districts) containing
   explorable rooms, all known/travelable from the start (Buddy's lived
   here all his life); game state gates relevance, not access. Travel
   via focusable side panel (Tab, arrows, Enter). See `systems/hubs.md`.

### Supporting

10. ✅ **Quest/objective tracking** — journal built on flags: derived,
    never stored (`Entry{Flag, Text}` content + `journal` modal).
    See `systems/events.md`.
11. ⬜ **Save/load** — serialize entity positions + flags (components hold
    config only, so nothing else needs persisting).
12. ⬜ **UI growth** — status bar, story/system text styling, command
    history.
