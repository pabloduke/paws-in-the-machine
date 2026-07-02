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

3. ⬜ **Turns & time** — turn counter plus per-turn tick hooks. Stealth, NPC
   schedules, and timed events all hang off this. Build first.
4. ⬜ **Visibility & scope** — formalize what anyone (Buddy, dogs, cameras)
   can perceive: nested contents, containers/surfaces, hiding spots.
   Stealth is the inverse of this system.
5. ⬜ **Events/triggers** — scheduled and conditional events ("in 5 turns the
   owner returns", "when flag X and Buddy enters Y"). Generalizes the
   `engine.On` escape hatch.

### Headline systems

6. ⬜ **Stealth/detection** — the core mechanic: nobody suspects a cat.
   Observers (dogs, cameras, humans) with perception rules; Buddy's profile
   (baseline invisibility, modified by behavior — carrying human objects,
   pawing keyboards, restricted zones); suspicion states; consequences;
   counters (neon camouflage, hiding, distractions).
7. ⬜ **NPCs** — presence, movement on the turn tick, dialogue, memory.
   A dog = NPC component + observer component on one entity.
8. ⬜ **Hacking/cyberspace** — jack in/out (plane switch), subnets as a
   second tree region, ICE as cyber-observers (stealth system reused),
   data as items.
9. ⬜ **Hubs & travel** — city topology: hubs (Plaza, Industrial Zone,
   Microslop HQ, ...) containing explorable rooms; hub-to-hub travel.

### Supporting

10. ⬜ **Quest/objective tracking** — journal built on flags.
11. ⬜ **Save/load** — serialize entity positions + flags (components hold
    config only, so nothing else needs persisting).
12. ⬜ **UI growth** — status bar, story/system text styling, command
    history.
