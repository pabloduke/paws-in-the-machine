# Editor API v1

The browser and API use the same stores, contextual commands, validation and
mutation lock. API access is intended for the same trusted authoring environment
as the editor. JSON writes with an Origin must be same-origin; cross-site browser
requests are rejected. Command-line clients may omit Origin. No CORS is enabled.

Read `GET /api/v1/worlds/main` to obtain `catalogs`, `hierarchy`, and `revision`.
The response ETag is the revision. Every write requires that exact value in
`If-Match` and `Content-Type: application/json`. A stale revision returns 409;
a missing one returns 428. Re-read and review conflicts rather than blindly retry.
The world name must match the loaded editor world, otherwise 409; API operations
never implicitly switch worlds. An in-flight request remains bound to its original world across a switch;
reloaded handlers for the same directory share one mutation lock.
Single-directory editors expose their world as `main`.

## Resources

- `GET /api/v1/worlds/{world}/validation`: readiness and diagnostics.
- `GET /api/v1/worlds/{world}/entities/{kind}[/{id}]`: list or read definitions.
- `POST /api/v1/worlds/{world}/entities/{kind}`: create an unplaced definition.
- `PUT /api/v1/worlds/{world}/entities/{kind}/{id}`: replace editable fields.
- `DELETE /api/v1/worlds/{world}/entities/{kind}/{id}`: delete after removing references; send `{}`.
- `POST /api/v1/worlds/{world}/hubs`: create a Hub with `name`.
- `POST /api/v1/worlds/{world}/{hubs|locations|rooms}/{id}/actions`: contextual commands below.
- `POST /api/v1/worlds/{world}/play-settings`: set `cell` to `location:UUID`, `room:UUID`, or empty to clear.

Kinds are `hub`, `location`, `room`, `world_item`, `npc`, `terminal`.
Definition fields: Hub `name`; Location/Room/NPC `name, description`;
world item `name, kind, short_description, full_description`; terminal `host_name`.
Item kind is `takeable`, `fixed`, or `scenery`. All request values are strings,
including coordinate integers. Responses to writes include the updated snapshot,
revision, newly created IDs, and an editor URL where applicable. Validation errors
use `{"error":{"code":"validation","message":"…"}}` and status 422.

## Contextual commands

| action | Context | Fields |
|---|---|---|
| save-details | Hub/Location/Room | name, description (except Hub) |
| delete-self | Hub/Location/Room | remove ownership, placement and content references first |
| create-child | Hub/Location | child_name, child_description; optional x,y,z to place immediately |
| assign-child | Hub/Location | child_id (currently unassigned) |
| place-child | Hub/Location | child_id (assigned, unplaced), x,y,z |
| move-child | Hub/Location | child_id, source_x,source_y,x,y,z (same level drag) |
| position | Location/Room | parent_id,x,y,z |
| unplace | Location/Room | parent_id |
| assign-parent | Location/Room | parent_id; child must be unplaced |
| unassign-parent | Location/Room | child must be unplaced |
| entry | Location | entry_id, or empty to clear |
| arrival | Hub | arrival_id, or empty to clear |
| save-thing | Location/Room | thing_kind; optional thing_id to edit held content; fields below |
| add-thing | Location/Room | thing_kind,thing_id |
| remove-thing | Location/Room | thing_kind,thing_id |
| connect-vertical | lower Location/Room | upper_id |
| disconnect-vertical | either endpoint | lower_id |

`save-thing` fields: items use `thing_name,item_kind,short_description,full_description`;
NPCs use `thing_name,thing_description`; terminals use `host_name`.
It creates and places content through one validated command. As with browser
forms, disk failures can leave a saved definition in Library; errors explain this.
Writes are not batch transactions. Review the resulting snapshot after any I/O error.

Example creation in a Room (substitute the actual world revision and Room UUID):

```json
{"action":"save-thing","thing_kind":"npc","thing_name":"Test NPC","thing_description":"(Placeholder)"}
```

## Height

Placement version 2 accepts signed z and requires w=0; version 1 still loads at
z=0. Omitted z on a new command means zero. x/y remain nonnegative. Two cells may
share x/y at different levels. Vertical connections are stored in
`vertical_connections.json` version 1 and join aligned, adjacent levels within
one parent. Remove connections before relocating their endpoints.


## Quests

`GET /api/v1/worlds/{world}/quests` lists definitions; append `/{id}` to read
one. `POST` on the collection creates, `PUT /{id}` replaces, and `DELETE /{id}`
removes. Quest bodies use structured JSON (unlike legacy string-field entity
commands). IDs for new quests and steps are generated when omitted. Preserve
step IDs when editing/reordering so existing progress remains associated with
the same objective. IDs cannot contain URL delimiters.

```json
{
  "name": "Pick Up the Cup",
  "description": "Pick up the Paper Cup at Megasoft.",
  "enabled": true,
  "auto_start": true,
  "completion_text": "Quest complete: Pick Up the Cup.",
  "steps": [
    {
      "text": "Pick up the Paper Cup at Megasoft.",
      "kind": "carry",
      "target": {"kind": "world_item", "id": "a09e5e48-ae2c-40c8-a048-6c856d6dd3af"}
    }
  ]
}
```

All writes require the snapshot's ETag in `If-Match`; missing/stale revisions
return 428/409. Creation returns 201 and the saved definition with generated
IDs. Successful writes return the new world ETag. Invalid enabled objectives
return 422 without writing; incomplete disabled drafts are accepted. Unknown
quest IDs return 404. Existing world-scoping and same-origin policies apply.
Quests participate in snapshots, revisions, world copies, and validation.
Preview is a read-only editor form operation, not a mutation API.

### Adjoining paths

Use the existing `POST /api/v1/worlds/{world}/locations/{id}/actions` or
`/rooms/{id}/actions` endpoint with the world ETag in `If-Match`:

```json
{"action":"set-passage","other_id":"<neighbor UUID>","blocked":"true","expected_blocked":"false"}
```

This blocks both directions of a horizontal adjacency. To reopen, send
`blocked:"false"` and `expected_blocked:"true"`. Both cells must share kind,
parent, and floor and be adjoining on one compass axis. The expected state
protects stale forms; existing API ETag validation also applies. No neighboring
cell means no configurable passage. Entrance selection still uses the Location
`entry` action with `entry_id`; an empty ID removes the reciprocal enter/out pair.
