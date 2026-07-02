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
4. ⬜ **Visibility & scope** — formalize what anyone (Buddy, dogs, cameras)
   can perceive: nested contents, containers/surfaces, hiding spots.
   Stealth is the inverse of this system.
5. ⬜ **Events/triggers** — conditional events ("when flag X and Buddy
   enters Y"), all state-based per turns.md. Generalizes the `engine.On`
   escape hatch.

### Headline systems

6. 📝 **Stealth/checks/XP** — the core mechanic: nobody suspects a cat.
   Boundary checks, not simulation: hidden seeded rolls (XCOM rule —
   identical attempt, identical result) against a single XP stat +
   prose-telegraphed circumstances; observer types (human/dog/camera).
   The only dice in the game. See `systems/stealth.md`.
7. ⬜ **NPCs** — presence, dialogue, memory. Positions are a function of
   game state (flags), never simulated movement. A dog = NPC component +
   observer configuration on one entity.
8. ⬜ **Hacking/cyberspace** — jack in/out (plane switch), subnets as a
   second tree region, data as items. No Hacking stat, no rolls: the
   player performs the hack through play; ICE is puzzle, not check.
9. ⬜ **Hubs & travel** — city topology: hubs (Plaza, Industrial Zone,
   Microslop HQ, ...) containing explorable rooms; hub-to-hub travel.

### Supporting

10. ⬜ **Quest/objective tracking** — journal built on flags.
11. ⬜ **Save/load** — serialize entity positions + flags (components hold
    config only, so nothing else needs persisting).
12. ⬜ **UI growth** — status bar, story/system text styling, command
    history.
