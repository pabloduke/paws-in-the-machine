# Ambient world draft — 2026-09-07

User-authorized first pass: 16 items, 7 descriptive NPCs, and 4 terminals across
the eight existing Locations and Rooms. All additions were created and placed
through the editor API. They are ordinary editable content, intended for the
user to reshape. This authorization permits ambient drafts, not new missions
or major world facts. No buildings, Rooms, or floors were added.

Existing definitions, prose, items, terminal assignments, geometry and ownership
were compared with the pre-population API snapshot and preserved. After that
comparison, Lowtown's missing arrival was set to its only placed Location,
Megarise 7, so the world can launch. The existing start remains Megasoft.

## Review in the editor

Open the indicated cell, then click any entry in its Contents section to edit it.
The UUIDs below identify this exact draft if names are changed later.

### Worstum HQ

Editor route: `/content/locations/bbc294fa-3812-4c8c-95bc-8782579ec33b`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Umbrella Stand | `3b53f2f8-9ba7-4fc6-b677-e45e2ce8f886` |
| world_item | Folded Receipt | `37137b5e-2f5f-44ba-9b0c-449b390b6d98` |
| npc | Waiting Visitor | `20f5f8e9-5290-48d6-8fbd-5f3dd0063cae` |
| terminal | worstum-visitor-kiosk | `2fd66456-8a6e-4109-865b-feff2c258f24` |

### Megasoft

Editor route: `/content/locations/a90406ac-9345-4cea-818f-eb0f5f180563`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Entrance Planter | `83b8485d-cf4b-42bc-9034-22a2fd8ef27d` |
| world_item | Paper Cup | `a09e5e48-ae2c-40c8-a048-6c856d6dd3af` |
| npc | Arriving Employee | `c547f8c8-e3a8-4b1b-ae44-37fb265a5bfd` |

### Microslop

Editor route: `/content/locations/e56ad7ab-7e4f-4b0a-b7fa-68222dd19f5c`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Scuffed Bench | `39945df4-6c37-45f3-b6bb-ec8434334b65` |
| world_item | Rubber Band | `bf93615a-a8b4-4770-91f6-eeb600faf2a9` |
| npc | Waiting Courier | `1e338f53-2576-403e-a100-8ccc4423b6e4` |
| terminal | microslop-directory | `b9369711-b0b0-4f99-95bb-91c1aae800d3` |

### Megarise 7

Editor route: `/content/locations/30c01e94-b68b-41ea-819f-3dc514c8ac38`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Worn Doormat | `adf9356b-6c00-41bb-89ff-b1ed2cbe3344` |
| world_item | Bottle Cap | `67e22b6c-d9b1-4e36-a60b-f80cedee1807` |
| npc | Returning Resident | `8252a986-e78b-4656-831b-47f7ef38b36e` |

### Megasoft Lobby

Editor route: `/content/rooms/3024886f-3c29-43a7-9dca-52e5c2aab0f3`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Reception Desk | `c039bcc7-a72c-40e5-893c-3f921f785e0b` |
| world_item | Lobby Seating | `760128be-716b-4bb7-b657-c9b04d2711ca` |
| npc | Lobby Receptionist | `481cb64d-e464-4341-9c14-b61bcac65ec8` |
| terminal | megasoft-check-in | `b85d521b-0886-441d-9206-6e3d23d2b91e` |

### Megasoft Cafeteria

Editor route: `/content/rooms/65167489-98fa-4efe-b478-2543f41f06b2`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Tray Return | `5ea54e5c-941c-448e-ac70-be6e8bc0e2a3` |
| world_item | Paper Napkin | `f4e73766-108c-4d5e-b847-22e66e42c2fb` |
| npc | Lunch Break Employee | `65fffa28-9367-4c5d-8f17-52bf3c091ae6` |
| terminal | megasoft-cafeteria-register | `1b247e99-0dcb-4148-a9ac-e340df55802b` |

### Megasoft Hallway to Cafeteria

Editor route: `/content/rooms/a0c39783-9a9b-40f1-89ff-0d1f6e7cb56c`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Waste Bin | `2581ddb0-9b02-4560-bc63-9a6330677cf0` |
| world_item | Crinkled Wrapper | `ad8219fe-b10e-4964-8256-8991059b489a` |
| npc | Passing Employee | `0a82e387-51f6-49b5-8b0c-b1cc3719c4a0` |

### Paw's Apartment

Editor route: `/content/rooms/80111cf3-704f-475a-90b3-ed607a25cd4c`

| Kind | Draft name | ID |
|---|---|---|
| world_item | Folded Blanket | `4bcf6e4d-8a03-4d7e-9661-ee5efc9b4d2f` |
| world_item | Cardboard Box | `77fc434d-7f2a-42fc-ba14-9fba2a1d8c32` |

## Remaining authoring gaps

- Worstum HQ and Microslop are separated from the starting Location by empty
  grid cells. Their contents are editable, but normal walking cannot reach them.
- Megarise 7 has no designated entry Room, so Paw's Apartment is not reachable
  from the outside. Choose its intended entrance and floor arrangement.
- Hightown is empty and therefore excluded from playtests.
- The existing ashley-laptop, mariah-laptop, laura-laptop, and archives terminals
  remain unplaced. No intended placement was inferred from their names.
- The existing Bookshelf description contains test text; it was preserved.
- NPCs currently provide descriptions, without dialogue or schedules. Terminals
  are visible/examinable but do not yet supply usable filesystems or networks.

Validation reports the world ready, with the warnings above. The draft includes
no puzzle clues, mission flags, access credentials, or new NPC logic.
