# Authored quests

First slice: ordered carry and visit objectives, authored in the editor and
loaded with its world. Existing Go-built missions are unaffected.

Definitions live in versioned `quests.json`. `internal/content` owns schema
and reference checks, editor commands own writes, and `internal/game` maps
catalog IDs to engine entity IDs. The evaluator lives in engine core beside
the checkpoint/event machinery. No system imports a sibling system.

The quest evaluator alone owns generated flags:
`quest:<quest-id>:active`, `quest:<quest-id>:step:<step-id>`, and
`quest:<quest-id>:complete`. This is the data-authored counterpart of the
single-writer flag convention; editor authors select targets, not raw flags.
There is no second mutable progress model. Debug controls use the same flag
owner. Stable step IDs survive renaming/reordering; deleting or replacing a
step ID intentionally changes its association with saved progress.

Evaluate at initialization and after normal event checkpoints, never on a
read-only inspection or developer command. Completed objectives remain true;
ordered, already-satisfied objectives can complete together. An exact current
cell matches a visit condition; carrying an existing entity matches carry.
Unavailable targets block progress and explain why. Completion queues its
message once. No rewards, scripting, branching, or dialogue hooks in this slice.

Commands: `quests`, `quests start <id>` in the overworld. F12 opens a local
developer console; `sudo devmode --meow` enables session-only controls,
`sudo devmode --off` disables them. Both commands also work in gameplay
terminal sessions when no password prompt is pending. Developer inspection
and forced changes do not run game event checks or award XP. Forced sessions
retain a MODIFIED marker, including in engine save snapshots.

The executable transition diagrams are in `docs/GAME_FLOW.md`, under
“Authored quests and developer inspection.”
