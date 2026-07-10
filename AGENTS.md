# Agent Rules

Hard rules for any agent working on this repo (user-declared
2026-07-10). These override default agent behavior.

## Content authority

1. **Do not create any lore.** World facts, history, names, factions,
   NPC personalities, story beats, reveals — all of it is
   user-declared. If it isn't in `docs/draft.md`, a user ruling, or
   user-written prose, it doesn't exist and you don't invent it.
2. **Do not create missions.** No quests, mission steps, objectives,
   or story structure of your own devising. Missions are declared by
   the user in `docs/draft.md`; agents build what's declared.

Where a mechanism structurally needs text the user hasn't written yet
(a banner, an error line, a room description), use the smallest
possible stub, mark it `(Placeholder)`, keep it free of new world
facts, and surface it to the user for replacement.

## Review duty

3. **Do point out gaps** — a declared mission step with no mechanism,
   a hint pointing at content that doesn't exist, a system with no
   test pinning its ruling.
4. **Do point out contradictions** — new declarations that conflict
   with prior rulings (`docs/systems/*.md`), with the world's rules
   (Masquerade, flags-only state, lawful geometry), or with each
   other. Raise it; don't silently pick a side.
5. **Do point out flaws** — design weaknesses, soft-locks, walls where
   there should be forks, geometry that cheats the player, puzzles
   that break the "no checks in the terminal" split.

When declared content and an existing system disagree, say so and ask
— the user decides. Repo conventions (boundaries, flags, seeds,
testing) live in `docs/BOUNDARIES.md` and `docs/systems/*.md`; state
of the world in `docs/PROJECT_STATE.md`.
