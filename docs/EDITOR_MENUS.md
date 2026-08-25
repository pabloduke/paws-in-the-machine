# Editor menu pattern

Status: Bubble Tea prototype reference. The bordered TUI presentation was
superseded by the local browser-editor design in `EDITOR.md` on 2026-08-22;
the declared workflow and field decisions remain design input.

## Purpose

These editor menu sketches declared the prototype's content, hierarchy, form
fields, and general composition. They were not diagrams drawn to scale. Panel
width, height, padding, and blank lines were responsive implementation details.

The browser editor replaces the numbered menu hierarchy with routable Header
tabs, Detail tabs, and Assignment screens. Future browser sketches do not
implicitly carry the TUI border or keyboard-selection rules below.

Browser CRUD ruling (2026-08-23): creation and editing use the same screen.
The Header tabs are `Content` and `Place`; Content Detail tabs select the
entity type. World Items, Hubs, Locations, Rooms, NPCs, Terminals, Corporations, and Users now keep a searchable
catalog on the left and a shared create/edit form on the right.

## TUI prototype presentation

Every editor menu or form has:

- a visible box border;
- `PAWS_IN_THE_SHELL` branding near the top;
- a screen-specific title below the branding;
- menu choices or form fields in the body; and
- spacing and dimensions fitted to the available terminal rather than copied
  literally from a sketch.

Historical presentation ruling (2026-08-22): the retired prototype's baseline
panel was 50% larger than the original rough dimensions, both title lines were
centered, and menus used a high-contrast truecolor cyberpunk palette. Actual
glyph size was controlled by the terminal emulator and could not be changed by
Bubble Tea.

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
│   │       ├── Enter Machine Hostname
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
2. The designer enters a machine hostname.
3. The machine hostname is also the terminal's network-map key. The editor does not
   ask for a separate map key.
4. Saving creates the terminal metadata and makes it available for later
   editing.
5. The future filesystem-authoring screen will let the designer enter the
   created terminal.
6. Once that screen is implemented, a new terminal will start with the same
   declared default filesystem structure as the deck.
7. The future screen will let the designer add directories and files.

The current terminal metadata schema stores an immutable UUID and `host_name`.
Host names use lower-case letters,
digits, underscores, hyphens, or dots and start and end with a letter or digit.
Host names are unique within an assigned Corporation network because they are that
network's map keys. Unassigned terminals and terminals on different networks
may share a host name.

Legacy version-1 terminal records may contain `username`. The editor preserves
that field but does not use it for new terminals. Accounts now live in
`users.json`, and the many-to-many grants in `terminal_access.json` attach
Users to Terminals without merging either definition.

"Enter the terminal" here is an editor-authoring operation: it exposes the
game's fake, in-memory terminal filesystem for content editing. It does not
grant access to the developer machine's real filesystem.

The existing runtime hacking model still stores a host name, optional username,
filesystem root, and home path directly. Loading the editor's User and access
records into that runtime model is deferred migration work.

The deck currently contains both ordinary structure and content backed by Go
functions and story flags. "Same default setup as deck" therefore establishes
the structural starting point, but does not yet rule whether a created
terminal receives only the deck's directory layout and ordinary starter
files, or also its dynamic and mission-specific nodes. The editor must not
copy or invent story content until that distinction is declared.

## Corporation workflow

User ruling (2026-08-24): the editor defines Corporations rather than exposing
Host Networks directly. Each Corporation is one real isolated host network,
and its Corporation name is also the network name. This does not add corporate
hierarchy, departments, roles, or employee modeling. Entry points, routes, and
runtime behavior remain deferred. Content > Corporations provides name-only
CRUD. A selected Corporation has Details and Terminals tabs.

The Terminals tab:

- lists terminals assigned to the selected network;
- creates a terminal and its assignment in one workflow;
- assigns an existing unassigned terminal; and
- unassigns a terminal without deleting its definition.

Definitions and relationships remain separate in `host_networks.json`,
`terminals.json`, and `network_assignments.json`. Each terminal may have zero or
one Corporation assignment. A Corporation may contain many terminals. Hostname
uniqueness is enforced within each Corporation network during assignment and while editing
an assigned terminal.

Unassigned terminals are valid. An assigned terminal cannot be deleted, and a
Corporation containing terminals cannot be deleted; the designer must unassign
first. Existing assignments must reference valid terminal and Corporation
UUIDs. How the player enters a network, crosses between networks, and discovers
hosts remains deliberately unresolved.

## User and terminal-access workflow

User ruling (2026-08-24): Content > Users provides CRUD for a reusable
fictional username and password. These are plain authored game data, never real
credentials. Users may remain unassigned. A selected Terminal has Details and
Access tabs; Access grants or revokes an existing User's permission to
authenticate and log in to that machine.

The relationship is many-to-many. One User may access many Terminals and one
Terminal may accept many Users. Usernames may repeat globally, but users with
the same username cannot both be granted to one terminal. Existing grants must
reference valid User and Terminal UUIDs. A User or Terminal with grants cannot
be deleted until those grants are revoked.

This model deliberately omits corporate hierarchy, departments, groups,
roles, ACLs, `sudo`, and POSIX `chmod` behavior. Simple file
readable/writable/executable restrictions remain possible future puzzle
mechanics, but are not part of account assignment. Runtime authentication does
not consume these files yet.

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

Saving this form creates a catalog record with the authored item name, compact
listing text, full examine text, and selected world-item kind. Placement
remains a separate editor workflow; creating the item does not silently place
it in a room or in the player's inventory.

The short description is used when the world lists the item. The full
description is used whenever the item is examined, whether it is in the world
or in the player's inventory. The version-1 JSON schema stores these as
`short_description` and `full_description`; those storage names do not change
the human-facing form labels.

The current engine uses `Entity.Name` for visible lists and one
`Description.Text` value for examining. Supporting a distinct short listing
description therefore requires a deliberate model/rendering seam rather than
overloading or rewriting the authored item name.

The current engine also requires every entity to have a stable ID and may give
it aliases. The browser editor generates an immutable UUIDv4 and keeps it out
of the normal designer-facing form; editable names remain the visible identity.
Alias authoring is still undeclared.

All four form values are required. Duplicate names remain valid because UUIDs
are authoritative. Successful Save keeps the created or updated item selected,
shows a compact confirmation, and writes the versioned
`internal/game/content/world_items.json` catalog atomically. This persistence
is editor-only for now; the game does not load the file yet.

A selected world item also exposes Delete. Delete requires explicit browser
confirmation, removes the catalog record atomically, and preserves the prior
catalog as the recovery copy. No authored relationship can currently reference
a world item; dependency checks become mandatory before placement or other
relationship files are allowed to reference these UUIDs.

## Hub CRUD workflow

Initial browser ruling (2026-08-23): Content > Hubs is a shared searchable
catalog and create/edit form. Its only authored field is:

```text
Hub Name:
```

Saving creates or updates a versioned `internal/game/content/hubs.json` record
with an editor-generated immutable UUID and the display name. Duplicate names
are valid. The initial catalog starts empty; the editor does not import the
runtime's hard-coded hubs.

A Hub with no assigned Locations is valid. A selected Hub's Locations tab
lists its children, creates and assigns a Location, assigns an existing
orphaned Location, and unassigns an unplaced Location. Coordinates do not
belong on this Content form; Place > Locations authors them separately.

## Location, Room, and NPC CRUD workflows

Initial browser ruling (2026-08-24): Locations, Rooms, and NPCs each use a shared
searchable catalog and create/edit form with two required fields:

```text
Name:
Description:
```

Location and Room Description are general place prose. NPC Description is the NPC's
general examine text. Both schemas store an editor-generated immutable UUID;
duplicate display names are valid because UUIDs establish identity. These
definition forms do not assign a Location to a Hub, a Room to a Location, place an NPC, or
author dialogue, presence rules, aliases, or keywords.

A selected Location's Rooms tab mirrors the Hub workflow for Room ownership.
It lists assigned Rooms and their placement status, creates and assigns Rooms,
assigns existing orphaned Rooms, and unassigns unplaced Rooms. It also manages
the optional entry Room from eligible Rooms already placed in that Location.

Save creates or updates the corresponding `locations.json`, `rooms.json`, or
`npcs.json` catalog.
Delete requires confirmation and preserves the prior catalog as a recovery
copy. Orphaned rooms and unassigned NPCs are valid.

### Item-kind implementation

User ruling (2026-08-22): the designer-facing item types are the contract; the
code may represent them with booleans, enums, concrete types, or another
appropriate mechanism as long as behavior maps back to those types.

The designer-facing taxonomy has four base kinds rather than several
overlapping booleans:

```text
takeable
fixed
scenery
terminal
```

The flat-file schemas separate the specialized terminal definition from
ordinary world items. `world_items.json` therefore stores the three-value
`takeable` / `fixed` / `scenery` enum, while `terminals.json` stores the
specialized terminal metadata. Together they preserve the four
designer-facing kinds; this storage boundary does not introduce a new UI
category.

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

In the retired TUI prototype, undeclared choices opened framed screens with one
selectable `Back` entry. In the browser editor, Quests and the Place screens
other than Rooms remain visible placeholders among these workflows.

## Place Locations and Rooms workflow

User ruling (2026-08-24): the editor hierarchy is `Hub → Location → Room`.
Locations are player-standable and may optionally contain Rooms; recursive
Location nesting is not part of version 1.

Content establishes ownership first: Locations are optionally assigned to one
Hub, and Rooms are optionally assigned to one Location. Place > Locations
selects a Hub and uses a 10×10-default sparse grid containing only that Hub's
assigned Locations. Place > Rooms selects a Location and uses a 5×5-default
sparse interior grid containing only that Location's assigned Rooms. Either
grid may expand and is not a world-size limit. Version 1 fixes `z=0` and `w=0`.
Clicking an empty cell or entering non-negative `x` and `y` places the selected
child; placing it again moves it. Unplacing preserves ownership. A child must
be unplaced before it can be unassigned or moved to another parent. Children
may remain orphaned or assigned-but-unplaced, each child has at most one
parent, and each cell has at most one child.

A Location may optionally designate one Room already placed within it as its
entry Room from Content > Locations > Rooms. An unset entry is valid. The
entry must be cleared before that Room is unplaced or unassigned. Definitions remain in
`hubs.json`, `locations.json`, and `rooms.json`; relationships persist in
`location_assignments.json`, `room_assignments.json`,
`location_placements.json`, `room_placements.json`, and
`location_entry_rooms.json`.

## Open behavior gaps

The visual pattern does not yet declare:

- validation error presentation outside the implemented Content CRUD screens
  (the integrity-only validation scope is ruled in `EDITOR.md`);
- the Quest form and runtime mapping;
- the contents of the remaining Place screens and the Quest Content screen;
- whether `Create Quest` authors the existing mission-file/flag/event
  composition or introduces a different runtime model;
- the exact home path derived for a created terminal;
- which deck files, aliases, and dynamic nodes belong in the default terminal
  template;
- default services, ports, route requirement, banner, and process
  table for a created terminal;
- the terminal filesystem schema and loading terminal metadata/filesystems into
  `game.NewWorld`;
- Corporation-network entry points, cross-network routing, and runtime loading;
- runtime loading of Users and terminal-access grants, including migration of
  legacy per-host credentials;
- how a created terminal is placed into the overworld after its filesystem is
  authored;
- whether world items can have aliases or world keywords at creation time;
