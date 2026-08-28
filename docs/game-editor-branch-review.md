# `game-editor` branch review and deferred execution handoff

Status: findings addressed 2026-08-27, except the branch-history decision,
which the user has deferred. Each finding below now carries a resolution note.

Reviewed: 2026-08-27
Implemented: 2026-08-27 (all six P1/P2 findings and the P3 documentation
items; work landed on `game-editor` with no history rewritten)

## Review scope

- Base: `main` / `origin/main` at `af8b2a6e20cf24a50e3d099f7a7c57d1f755aee0`.
- Branch tip: `game-editor` / `origin/game-editor` at
  `b4de9fd221e8983ce59a85f66a64c8f08698369b`.
- Committed branch delta: 10 commits, 84 files, 12,881 insertions and 7
  deletions.
- Also reviewed the working tree present on 2026-08-27: 15 modified tracked
  files plus the untracked relation-store, contents, overview, navigation,
  world-model, vendored HTMX, and contents-schema files shown by `git status`.
- The review covers both the committed delta and that uncommitted work. A later
  agent must run `git status -sb` first and reconcile this document with any
  changes made after the review date.

## Findings

### P1 — Conflicting gluings can violate lawful reciprocity

Evidence: `internal/systems/charts/charts.go:165-190` installs a forward and a
reverse face with unconditional map assignments. A second gluing may claim a
face already installed as the reverse of the first. The overwrite leaves the
first forward edge intact but changes its return edge, producing exactly the
scrambled geometry the system says is unrepresentable.

Example: glue `A east -> B`, then glue `C east -> B`. The second declaration
overwrites `B west -> A` with `B west -> C`; `A east` still reaches `B`.

Required change:

1. Before mutating `w.gluings` or `w.declared`, reject a declaration when
   either its forward face or generated reverse face is already claimed by a
   different target.
2. Return a content bug that identifies both the claimed face and conflicting
   declaration.
3. Keep the weave unchanged after a rejected declaration.
4. Add tests for a forward-face conflict, a reverse-face conflict, no partial
   mutation, and `Unmarshal` surfacing the conflict.

Acceptance: every installed gluing has exactly one reciprocal return edge and
the new conflict tests fail against the reviewed implementation.

**Resolved.** `Glue` now rejects a declaration claiming either an occupied
forward face or an occupied return face, reports the conflicting face and
target, and leaves the weave untouched. Re-declaring an identical gluing is
explicitly not a conflict. Tests in
`internal/systems/charts/integrity_test.go`.

### P1 — Duplicate chart identity can make runtime exits nondeterministic

Evidence: `internal/systems/charts/charts.go:147-155` silently replaces charts
with duplicate IDs. `internal/systems/charts/file.go:127-148` does not reject a
room/entity ID placed in multiple cells or charts. `Weave.Apply` iterates Go
maps and rewrites an entity's `engine.Exits` for every placement, so the last
random iteration wins.

The same loader accepts a gluing whose source or destination coordinate is
empty. `Exits` then lets the invalid gluing override ordinary adjacency while
producing no destination (`charts.go:230-239`). That silently removes a
passage instead of reporting malformed geometry.

Required change:

1. Validate unique, non-empty chart IDs.
2. Validate that every entity ID is non-empty and occurs in exactly one cell
   across the weave.
3. Validate that both endpoints of every gluing are occupied cells.
4. Reject duplicate JSON chart IDs and invalid endpoint declarations as
   content bugs before `Apply` can mutate the world.
5. Add tests for each invalid form and for deterministic, unchanged behavior
   of valid files.

Acceptance: malformed identity or endpoints cannot reach `Apply`, while the
existing byte-identical round-trip and game exit tests continue to pass.

**Resolved.** `Unmarshal` rejects empty and duplicate chart IDs, empty entity
IDs, and an entity placed in more than one cell across the weave, dropping the
offending cells so nothing nondeterministic reaches `Apply`. `Glue` requires
both endpoints to be occupied. Cell keys are walked in sorted order so a
duplicate is reported against the same cell every run. The repository's own
`charts.json` still loads with zero bugs and the `internal/game` exit tests
pass unchanged.

### P1 — Rooms under an orphan Location disappear from the assembled editor view

Evidence: an unassigned Location is explicitly a valid resting state. In
`cmd/editor/world_model.go:194-209`, a Room assigned to that Location is not an
orphan Room, while its Location is recorded as an orphan Location. The tree is
then built only by walking Hubs at `world_model.go:220-242`. The orphan section
at `world_model.go:259-271` adds only the Location itself, not its owned Rooms.

Consequences:

- the Room is absent from the Overview;
- the Room is absent from every cell-contents picker;
- contents already in that Room cannot be listed by
  `contentsPageData` (`cmd/editor/web_contents.go:249-261`);
- the Overview's claim to draw the authored world is false for a valid
  intermediate state.

Required change:

1. Represent orphan Location subtrees rather than flattening orphan Locations
   and orphan Rooms into unrelated lists.
2. Include Rooms owned by an orphan Location in `snapshot.Containers`, with a
   stable ancestry label such as the existing `(no Hub)` marker and no new
   world facts.
3. Make held contents discoverable for every container, not only containers
   reached through `snapshot.Hubs`.
4. Add a regression test that creates an orphan Location, assigns it a Room,
   places contents in both cells, and verifies the Overview and all relevant
   pickers.

Acceptance: every existing Location and Room appears exactly once in the
snapshot and remains authorable whether or not its ancestors are assigned.

**Resolved.** `worldSnapshot` now builds a node for every Location with its
Rooms attached, then files it under a Hub or under `Unrooted`; ancestry decides
where a node is filed, never whether it exists. The Overview draws unrooted
Locations as their own `(no Hub)` subtree, the picker offers their Rooms, and
`contentsPageData` reads `snapshot.ContentsOf` directly instead of walking the
Hub tree. Advisory counts walk all Locations via `AllLocations`.

### P2 — A stale remove form can remove an entity from the wrong cell

Evidence: `cmd/editor/web_contents.go:129-146` accepts a submitted `container`
but calls `contents.Remove` by entity kind and ID only. It never verifies that
the current record still belongs to the submitted cell. If another request has
moved the entity, submitting the old form removes it from its new cell while
reporting success on the old screen.

Required change:

1. Load the current relation with `ContainerOf`.
2. Require its parent kind and ID to match the submitted container before
   deleting it.
3. Treat a missing or mismatched relation as a stale request, preserve the
   current placement, and render an actionable error.
4. Add an HTTP regression test that renders or models the old cell, moves the
   entity, submits the old remove request, and asserts no mutation.

Acceptance: removal is scoped to the cell shown by the form.

**Resolved.** `removeContent` loads the current relation with `ContainerOf`
and requires its parent kind and ID to match the submitted cell. A moved or
already-removed entity yields an explanation naming where it went and leaves
storage untouched.

### P2 — Pre-write integrity validation omits cell contents

Evidence: `cmd/editor/web.go:296-300` promises to refuse writes while editor
relationships are invalid, but `validateEditorRelationships` in
`cmd/editor/web_terminal_access.go:261-268` checks only network, authentication,
and spatial relationships. It never invokes `contentsProblems` or an
equivalent fail-fast contents validator. A schema-valid `contents.json` with a
missing entity or container therefore does not block subsequent writes.

Required change:

1. Add a fail-fast contents relationship validator backed by the same logic as
   the Overview collector.
2. Include it in `validateEditorRelationships`.
3. Add tests for a missing entity and missing container, asserting an unrelated
   protected POST returns `409 Conflict` and does not alter storage.

Acceptance: the write gate and Overview agree on whether the complete
relationship graph is valid.

**Resolved.** `validateContentsRelationships` wraps `contentsProblems` and is
now part of `validateEditorRelationships`. A protected POST returns 409 and
mutates nothing while `contents.json` names a missing entity or cell.

### P2 — The Overview does not actually collect every relationship problem

Evidence: `docs/EDITOR.md:356-374` says every relationship problem is listed at
once. `cmd/editor/web_overview.go:178-196` collects all spatial and contents
problems, but calls the fail-fast `validateNetworkRelations` and
`validateAuthRelationships` once each. Those validators return on their first
bad record (`web_host_networks.go:431-470` and
`web_terminal_access.go:222-258`), hiding later problems in the same catalog.

Required change:

1. Extract collecting forms for network and authentication validation.
2. Keep small fail-fast wrappers for the write gate.
3. Have `allProblems` append every collected problem.
4. Prefer human-readable entity names when the definition still exists; use a
   UUID only for a dangling reference, matching the documented Overview rule.
5. Add a test with at least two independent network problems and two
   independent access problems.

Acceptance: the Overview count and list include every independently authored
relationship contradiction in one response.

**Resolved.** `networkProblems` and `authProblems` collect; the original
`validateNetworkRelations` and `validateAuthRelationships` are now thin
fail-fast wrappers over them. Both name surviving entities and print a UUID
only for the dangling end. Two pre-existing tests that pinned the old
UUID-only wording were updated to assert the name-first form.

### P3 — Documentation cleanup is needed after the fixes

**Resolved.** See the notes on each bullet below.

- `docs/systems/charts.md` says charts "never touch `internal/engine`", while
  `internal/systems/charts/charts.go` imports the engine and `Apply` mutates
  `engine.Exits`. Rewrite this as the actual boundary: charts do not require
  engine package changes, but the adapter currently lives in the charts
  package. — **Done**, and an "Integrity" section now documents the two
  invariants the fixes enforce.
- Recheck every claim in `docs/EDITOR.md` about comprehensive validation and
  the Overview after implementing the collectors above. — **Done.** The
  Overview claim now enumerates which catalogs are collected, and the write-gate
  paragraph states that both forms read the same records.
- If any fix changes player-visible navigation or game-state transitions,
  update code, tests, and `docs/GAME_FLOW.md` together. The reviewed chart
  retrofit is asserted to be behavior-preserving, so no diagram-only edit is
  requested here. — **No `GAME_FLOW.md` change made, deliberately.** The editor
  fixes are developer tooling. The chart fixes reject malformed content that
  previously produced silent, nondeterministic geometry; for content that was
  already valid — including this repository's `charts.json` — the derived exits
  are byte-for-byte unchanged and the `internal/game` exit tests pass without
  modification. No player-visible behavior or state transition changed.

## User decision required before implementation

`docs/BOUNDARIES.md` says a feature branch must not cross another system's
owned files and that shared integration points are append-only. This branch
combines the charts system, the browser editor, authored-content schemas, and
changes to both `docs/systems/hacking.md` and `docs/systems/hubs.md`.

**Status: the user deferred this decision on 2026-08-27** and authorized the
fixes to proceed on `game-editor` in the meantime. No history has been
rewritten, no commits moved, and no working-tree changes discarded. The choice
below is still open and must be settled before any PR.

The later agent must not silently choose a branch-history strategy. Ask the
user to select one of these approaches:

1. split chart/runtime work, foundational editor work, and the current
   contents/overview work into reviewable branches; or
2. explicitly treat `game-editor` as an authorized integration branch and
   keep the combined history.

Do not rewrite history, discard the current working tree, or move commits until
the user rules on this.

## Recommended execution order after authorization

1. Snapshot `git status -sb`, preserve all user changes, and resolve the branch
   strategy above.
2. Pin each P1 finding with a failing test before changing implementation.
3. Fix gluing conflicts and chart identity/endpoint validation.
4. Fix the world snapshot so valid orphan subtrees remain visible and
   authorable.
5. Pin and fix stale cell removal.
6. Unify collecting and fail-fast relationship validation, including contents,
   networks, and access grants.
7. Update only the documentation claims affected by the implemented fixes.
8. If player-visible behavior changes, update `docs/GAME_FLOW.md` in the same
   change; otherwise state explicitly that the work is editor-only or
   behavior-preserving.
9. Run the complete verification suite below and perform one real-browser
   smoke test with and without JavaScript/HTMX enhancement.

## Verification performed during this review

All of these passed against the reviewed working tree:

```sh
go test ./...
go test -race ./...
go vet ./...
git diff --check
node --check cmd/editor/static/editor.js
```

Not performed: a real-browser interaction pass, accessibility audit, or
manual editing of the repository's live content catalogs. Handler tests cover
the HTTP flows, but a later execution pass should still exercise Overview,
Content, Place, browser history, stale-form behavior, and no-JavaScript
fallback against a temporary content directory.
