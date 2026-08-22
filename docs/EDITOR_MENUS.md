# Editor menu pattern

Status: user-declared editor UI convention (2026-08-22).

## Purpose

Editor menu sketches declare menu content, hierarchy, form fields, and general
composition. They are not diagrams drawn to scale. Panel width, height,
padding, and blank lines are responsive implementation details.

Future menu sketches may omit the surrounding frame. Every editor menu and
form is still assumed to use the shared bordered presentation described here.
The border does not need to be repeated in each sketch.

## Shared presentation

Every editor menu or form has:

- a visible box border;
- `PAWS_IN_THE_SHELL` branding near the top;
- a screen-specific title below the branding;
- menu choices or form fields in the body; and
- spacing and dimensions fitted to the available terminal rather than copied
  literally from a sketch.

Current presentation ruling (2026-08-22): the implemented baseline panel is
50% larger than the original rough dimensions, both title lines are centered,
and menus use a high-contrast truecolor cyberpunk palette. Actual glyph size is
controlled by the terminal emulator and cannot be changed by Bubble Tea.

A borderless declaration such as:

```text
Create Item

1. Create World Item
2. Create Terminal
```

therefore means the equivalent of:

```text
╔══════════════════════════════╗
║      PAWS_IN_THE_SHELL       ║
║         Create Item          ║
║                              ║
║   1. Create World Item       ║
║   2. Create Terminal         ║
║                              ║
╚══════════════════════════════╝
```

The example communicates the shared composition, not an exact character
width, height, or alignment measurement.

## Declared hierarchy so far

```text
Game Editor
├── Create
│   ├── Create Item
│   │   ├── Create World Item
│   │   │   ├── Enter Item Name
│   │   │   ├── Select Item Type: Takeable / Fixed / Scenery
│   │   │   ├── Enter Short Description
│   │   │   ├── Enter Full Description
│   │   │   └── Select Save or Cancel
│   │   └── Create Terminal
│   │       ├── Enter Username
│   │       ├── Enter Host Name
│   │       └── Select Save or Cancel
│   ├── Create NPC
│   ├── Create Quest
│   ├── Create Room
│   └── Create Hub
├── Edit
└── Place
```

This hierarchy records only navigation and user-authored labels. It does not
define entity schemas, quest behavior, persistence, validation, keyboard
bindings beyond the shared selection rules, or the contents of `Edit` and
`Place`.

## Create Terminal workflow

Declared behavior (2026-08-22):

1. `Create Item` -> `Create Terminal` opens the terminal creation form.
2. The designer enters a username and a host name.
3. The host name is also the terminal's network-map key. The editor does not
   ask for a separate map key.
4. Saving creates the terminal and makes it available for later editing.
5. The designer can enter the created terminal from the editor.
6. A new terminal starts with the same default filesystem structure as the
   deck.
7. While inside it, the designer can add directories and files.

"Enter the terminal" here is an editor-authoring operation: it exposes the
game's fake, in-memory terminal filesystem for content editing. It does not
grant access to the developer machine's real filesystem.

The existing hacking model stores a host name, username, filesystem root, and
home path separately. The creation workflow must apply the host-name value to
both `Host.Name` and the network-map key, while the username remains the fake
shell account. The remaining fields require declared defaults or additional
form fields.

The deck currently contains both ordinary structure and content backed by Go
functions and story flags. "Same default setup as deck" therefore establishes
the structural starting point, but does not yet rule whether a created
terminal receives only the deck's directory layout and ordinary starter
files, or also its dynamic and mission-specific nodes. The editor must not
copy or invent story content until that distinction is declared.

## Create World Item workflow

Initial declared form (2026-08-22):

```text
Create World Item

Enter Item Name:
Select Item Type: Takeable / Fixed / Scenery
Enter Short Description:
Enter Full Description:
Select Save or Cancel
```

Saving this form will create an entity with the authored item name, compact
listing text, full examine text, and selected world-item kind. Placement
remains a separate editor workflow; creating the item does not silently place
it in a room or in the player's inventory.

The short description is used when the world lists the item. The full
description is used whenever the item is examined, whether it is in the world
or in the player's inventory. The content schema may store these as
`short_description` and `full_description`; those storage names do not change
the human-facing form labels.

The current engine uses `Entity.Name` for visible lists and one
`Description.Text` value for examining. Supporting a distinct short listing
description therefore requires a deliberate model/rendering seam rather than
overloading or rewriting the authored item name.

The current engine also requires every entity to have a stable ID and may give
it aliases. Those values are not declared by this first form. The editor must
not guess whether the ID is derived from the item name, generated separately,
or entered through another field until that behavior is ruled.

### Item-kind implementation

User ruling (2026-08-22): the designer-facing item types are the contract; the
code may represent them with booleans, enums, concrete types, or another
appropriate mechanism as long as behavior maps back to those types.

The implementation will use one explicit base-kind enum rather than several
overlapping booleans:

```text
takeable
fixed
scenery
terminal
```

The mapping to the current engine is:

- `takeable` attaches portable behavior and can enter inventory;
- `fixed` is non-portable and appears in world listings;
- `scenery` is non-portable and remains targetable but is omitted from the
  automatic world listing; and
- `terminal` carries the specialized host and fake-filesystem definition.

Additional behavior is modeled independently from the base kind. Container,
openable, interaction, conditional-presence, aliases/keywords, and dynamic
description behavior are capabilities rather than mutually exclusive item
types. This keeps combinations such as a takeable container or interactive
fixed item representable without multiplying menu categories.

## Interpretation rules

When a future sketch contains only menu items or fields:

1. Preserve its labels and ordering.
2. Place the shared branding and screen title inside a border.
3. Treat indentation and nesting as navigation hierarchy.
4. Fit the panel to the terminal; do not copy rough dimensions literally.
5. Do not add undeclared menu choices, fields, lore, missions, or behavior.
6. Record missing navigation, controls, schemas, or persistence behavior as
   gaps for a user ruling.

## Selection and actions

User ruling (2026-08-22): every numbered menu supports both direct numeric
selection and a visible `>` selector.

```text
> 1. Create
  2. Edit
  3. Place
```

- Pressing an entry's number activates that entry directly.
- Up/Down moves the `>` selector through the available entries.
- Enter activates the selected entry.
- Escape returns one level from every submenu, form, or Back-only screen.
- `q` exits from the main menu; Ctrl+C exits from anywhere.

Creation forms present `Save` and `Cancel` as selectable actions rather than
asking the designer to type either word:

```text
> Save
  Cancel
```

Up/Down selects the action and Enter activates it. The exact spacing and
placement remain responsive like the rest of the menu frame.

In the navigation prototype, Up/Down or Tab/Shift+Tab moves between form
fields and actions. Enter advances to the next row. Left/Right changes the
selected world-item type. `Save` and `Cancel` both return to `Create Item`
without retaining or writing form data; persistence is deliberately deferred.

Until their forms are declared, `Edit`, `Place`, `Create NPC`, `Create Quest`,
`Create Room`, and `Create Hub` open framed screens with one selectable `Back`
entry.

## Open behavior gaps

The visual pattern does not yet declare:

- whether successful saves need confirmation beyond returning to a menu;
- validation and error presentation;
- the forms beneath most creation choices;
- the contents of `Edit` and `Place`;
- whether `Create Quest` authors the existing mission-file/flag/event
  composition or introduces a different runtime model;
- username validation and host-name validation;
- host-name uniqueness across the network and whether it also supplies an
  entity ID;
- the exact home path derived for a created terminal;
- which deck files, aliases, and dynamic nodes belong in the default terminal
  template;
- default services, ports, password, route requirement, banner, and process
  table for a created terminal;
- the data file/schema that makes editor-created hosts and filesystem nodes
  available to `game.NewWorld`;
- how a created terminal is placed into the overworld after its filesystem is
  authored;
- how a world item's stable entity ID is chosen and validated;
- whether world items can have aliases or world keywords at creation time;
- whether short and full item descriptions use multiline editors.
