# Game editor

Status: navigation-only menu prototype implemented; content creation, editing,
placement, and persistence are not implemented (2026-08-22).

## What the editor is

The editor is a developer-facing Bubble Tea terminal UI intended to grow into
a full game editor. Its current executable is deliberately a menu and form
prototype so navigation can be exercised before writable content schemas are
chosen.

The implementation lives in `cmd/editor/`. The declared shared frame, menu
hierarchy, form fields, and interpretation rules live in
[`EDITOR_MENUS.md`](EDITOR_MENUS.md).

This prototype replaces the earlier chart-grid editor. The chart subsystem and
its versioned geometry file still exist and remain part of the game, but the
current editor command does not read or write them.

## Running it

From the repository root:

```sh
go run ./cmd/editor
```

From `cmd/editor/`:

```sh
go run .
```

The command no longer depends on the process working directory to locate a
content file.

## Implemented navigation

```text
Game Editor
├── Create
│   ├── Create Item
│   │   ├── Create World Item
│   │   └── Create Terminal
│   ├── Create NPC
│   ├── Create Quest
│   ├── Create Room
│   └── Create Hub
├── Edit
└── Place
```

Every screen uses the shared bordered `PAWS_IN_THE_SHELL` frame. The baseline
panel footprint is 50% larger than the original 34-by-16 rough, while sizing
and spacing still adapt to the available terminal. Branding and screen titles
are centered geometrically inside the padded frame.

The menu prototype uses a truecolor cyberpunk palette: a near-black canvas,
dark panel, bright cyan frame and branding, magenta screen titles and selector,
white menu text, and a cyan inverse highlight on the selected entry. The
terminal emulator owns actual font size; the application increases panel size,
spacing, contrast, and text weight but cannot resize terminal glyphs.

Menus support two selection styles:

- press the displayed number to activate an entry directly; or
- move the visible `>` selector with Up/Down and press Enter.

Escape returns one menu level. `q` exits from the main menu, and Ctrl+C exits
from anywhere.

Choices whose forms have not been declared yet open a bordered, titled screen
containing only a selectable `Back` entry. This makes every declared route
navigable without inventing fields or behavior.

## Implemented forms

### Create World Item

The prototype exposes:

- item name;
- item type (`Takeable`, `Fixed`, or `Scenery`), cycled with Left/Right;
- short description;
- full description; and
- selectable `Save` and `Cancel` actions.

### Create Terminal

The prototype exposes:

- username;
- host name; and
- selectable `Save` and `Cancel` actions.

Within a form, Up/Down or Tab/Shift+Tab moves the `>` selector between fields
and actions. Enter advances from a field to the next row. Enter on `Save` or
`Cancel` returns to `Create Item`.

The description controls are single-line inputs in this navigation prototype.
The eventual multiline editing behavior remains undecided.

## No persistence yet

Neither `Save` nor `Cancel` writes game content. Both return to the parent menu,
and reopening a form starts with empty fields and the default `Takeable` item
type. The prototype does not create entities, terminals, filesystems, rooms,
NPCs, quests, hubs, charts, or placements.

This is intentional. It lets the menu structure and focus behavior be refined
before selecting writable schemas and migration rules.

## Existing systems left intact

Removing the old chart-editor UI did not remove:

- `internal/systems/charts/`;
- `internal/game/content/charts.json`;
- chart application during world construction;
- lawful-geometry tests; or
- the editor's prior commits in Git history.

Chart authoring can return beneath the declared `Place` workflow once its new
navigation and relationship to entity creation are specified.

## Future TODOs

These are capability gaps, not declarations of world content or missions.

### Refine the prototype

- [x] Add the shared bordered frame and branded title.
- [x] Add numbered selection and the `>` selector.
- [x] Add hierarchical Create, Edit, and Place navigation.
- [x] Add navigation-only world-item and terminal forms.
- [ ] Rule multiline field behavior.
- [ ] Rule validation and error presentation.
- [ ] Rule successful-save confirmation behavior.
- [ ] Replace Back-only screens as their forms are declared.

### Establish writable content

- [ ] Define a versioned content schema and migration path before enabling
  `Save`.
- [ ] Define stable entity-ID generation, validation, rename, and reference
  handling.
- [ ] Implement world-item persistence and map `Takeable`, `Fixed`, and
  `Scenery` back to engine behavior.
- [ ] Add distinct short-listing and full-examine descriptions to the runtime
  model.
- [ ] Implement terminal persistence, default filesystem construction, and
  filesystem authoring without copying undeclared story content.
- [ ] Keep writes atomic and recoverable.

### Restore placement and geometry authoring

- [ ] Put chart/node-grid authoring beneath `Place` rather than opening it at
  program startup.
- [ ] Restore assembled-world identity validation and duplicate-placement
  checks when placement returns.
- [ ] Add chart, gluing, metamap, and node authoring only after their screens
  and relationships are declared.
- [ ] Report downstream references affected by unplacement.

### Full game editing

- [ ] Create NPCs and place them in the world.
- [ ] Create basic, data-driven quest lines. Advanced quests may remain
  hand-written in Go when their behavior does not fit the editor's model.
- [ ] Author room, item, and NPC descriptions and dialogue.
- [ ] Add and manage world keywords.
- [ ] Define dependency-aware deletion before allowing full entity removal.
- [ ] Preserve content authority: the editor stores designer-authored prose and
  structure but never generates missing lore or missions.

Any future executable change that alters player-visible game behavior or game
state must update tests and `docs/GAME_FLOW.md` with the code. The present
navigation prototype is developer tooling only and does not change game flow.
