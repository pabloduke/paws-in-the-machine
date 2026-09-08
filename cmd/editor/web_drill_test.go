package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func drillSubmit(t *testing.T, e contentsTestEditor, target, action string, values url.Values) string {
	t.Helper()
	if values == nil {
		values = url.Values{}
	}
	values.Set("action", action)
	rec := postForm(t, e.handler, target+"/work", values, "http://example.com", false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("%s %s: %d %s", target, action, rec.Code, rec.Body.String())
	}
	return strings.Split(rec.Header().Get("Location"), "?")[0]
}
func drillCheck(t *testing.T, e contentsTestEditor, target string, wants ...string) string {
	t.Helper()
	rec := getRequest(t, e.handler, target, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: %d", target, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("%s missing %q", target, want)
		}
	}
	return body
}

func TestDrillAuthoringToGame(t *testing.T) {
	e := newContentsTestEditor(t)
	rec := postForm(t, e.handler, "/content/hubs", url.Values{"name": {"Test Hub"}, "drill": {"1"}}, "http://example.com", false)
	if rec.Code != http.StatusSeeOther {
		t.Fatal(rec.Code, rec.Body.String())
	}
	hubURL := strings.Split(rec.Header().Get("Location"), "?")[0]
	hubID := path.Base(hubURL)
	drillCheck(t, e, hubURL, "Locations in Test Hub", "Assigned but unplaced", `?x=0&amp;y=0&amp;z=0#cell-editor`)
	locationURL := drillSubmit(t, e, hubURL, "create-child", url.Values{"child_name": {"Test Location"}, "child_description": {"(Placeholder)"}, "x": {"1"}, "y": {"2"}})
	locationID := path.Base(locationURL)
	drillCheck(t, e, hubURL, `href="`+locationURL+`"`, "Test Location")
	roomURL := drillSubmit(t, e, locationURL, "create-child", url.Values{"child_name": {"Test Room"}, "child_description": {"(Placeholder)"}, "x": {"0"}, "y": {"0"}})
	roomID := path.Base(roomURL)
	body := drillCheck(t, e, roomURL, `aria-label="Breadcrumb"`, "Contents of Test Room", "World Items", "NPCs", "Terminals", `href="`+hubURL+`"`, `href="`+locationURL+`"`)
	if strings.Contains(body, `<select name="container"`) {
		t.Fatal("room contents require a global cell selection")
	}
	drillSubmit(t, e, roomURL, "save-thing", url.Values{"thing_kind": {"world_item"}, "thing_name": {"Test Item"}, "item_kind": {"takeable"}, "short_description": {"short text"}, "full_description": {"full text"}})
	items, _ := e.items.List()
	if len(items) != 1 {
		t.Fatal(items)
	}
	itemID := items[0].ID
	drillCheck(t, e, roomURL+"?thing=world_item:"+itemID, "Edit World Item in Test Room", `value="Test Item"`, "full text")
	drillSubmit(t, e, roomURL, "save-thing", url.Values{"thing_kind": {"world_item"}, "thing_id": {itemID}, "thing_name": {"Test Item"}, "item_kind": {"takeable"}, "short_description": {"short text"}, "full_description": {"updated text"}})
	drillSubmit(t, e, roomURL, "save-thing", url.Values{"thing_kind": {"npc"}, "thing_name": {"Test NPC"}, "thing_description": {"(Placeholder)"}})
	drillSubmit(t, e, locationURL, "save-thing", url.Values{"thing_kind": {"terminal"}, "host_name": {"test-host"}})
	drillSubmit(t, e, hubURL, "arrival", url.Values{"arrival_id": {locationID}})
	drillSubmit(t, e, locationURL, "entry", url.Values{"entry_id": {roomID}})
	postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {"location:" + locationID}}, "http://example.com", false)
	r := game.LoadAuthored(filepath.Dir(e.rooms.path))
	if !r.Ready() {
		t.Fatal(r.DiagnosticText())
	}
	eng := engine.New(r.World)
	eng.Execute("enter")
	if r.World.Room().ID != "room:"+roomID {
		t.Fatal("interior was not playable")
	}
	if got := eng.Execute("examine Test Item"); got != "updated text" {
		t.Fatal(got)
	}
	eng.Execute("take Test Item")
	if !r.World.Carried(r.World.FindID("world_item:" + itemID)) {
		t.Fatal("authored item not portable")
	}
	eng.Execute("out")
	if r.World.Room().ID != "location:"+locationID {
		t.Fatal("interior return broken")
	}
	placements, _ := e.locationPlacements.List()
	if len(placements) != 1 || placements[0].HubID != hubID || placements[0].X != 1 || placements[0].Y != 2 {
		t.Fatal(placements)
	}
}

func TestDrillUnplacedLibraryAndExplicitPlacement(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, _ := e.oneRoomWorld(t)
	hubURL, locURL := entityURL("hub", hub), entityURL("location", loc)
	unplacedURL := drillSubmit(t, e, hubURL, "create-child", url.Values{"child_name": {"Unplaced Location"}, "child_description": {"(Placeholder)"}})
	id := path.Base(unplacedURL)
	drillCheck(t, e, hubURL, "Assigned but unplaced", `href="`+unplacedURL+`"`)
	drillCheck(t, e, hubURL+"?x=1&y=0", "Cell 1, 0", "Place existing here", "Unplaced Location")
	drillSubmit(t, e, hubURL, "place-child", url.Values{"child_id": {id}, "x": {"1"}, "y": {"0"}})
	drillSubmit(t, e, unplacedURL, "position", url.Values{"parent_id": {hub}, "x": {"2"}, "y": {"0"}})
	drillSubmit(t, e, unplacedURL, "unplace", url.Values{"parent_id": {hub}})
	drillSubmit(t, e, unplacedURL, "unassign-parent", nil)
	drillCheck(t, e, "/library", "Unassigned Locations and Rooms", `href="`+unplacedURL+`"`)
	drillCheck(t, e, unplacedURL, "Unassigned location", "Assign parent")
	drillSubmit(t, e, unplacedURL, "assign-parent", url.Values{"parent_id": {hub}})
	drillCheck(t, e, unplacedURL, `href="`+hubURL+`"`)
	orphan, err := e.rooms.Create(describedInput{Name: "Unassigned Room", Description: "(Placeholder)"})
	if err != nil {
		t.Fatal(err)
	}
	drillCheck(t, e, entityURL("room", orphan.ID), "Unassigned room", "Library")
	drillSubmit(t, e, locURL, "assign-child", url.Values{"child_id": {orphan.ID}})
	drillCheck(t, e, locURL, "Unassigned Room")
}

func TestDrillErrorsPreserveFormsAndRelationships(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, room := e.oneRoomWorld(t)
	hubURL, locURL, roomURL := entityURL("hub", hub), entityURL("location", loc), entityURL("room", room)
	for _, tt := range []struct {
		target string
		form   url.Values
		want   string
	}{
		{hubURL, url.Values{"action": {"create-child"}, "child_name": {"Retained Name"}, "child_description": {"Retained Description"}, "x": {"0"}, "y": {"0"}}, "That cell is occupied"},
		{hubURL, url.Values{"action": {"create-child"}, "child_name": {"Retained Name"}, "child_description": {"Retained Description"}, "x": {"-1"}, "y": {"0"}}, "non-negative"},
		{roomURL, url.Values{"action": {"save-details"}, "name": {"Retained Name"}, "description": {""}}, "Name and Description are required"},
		{roomURL, url.Values{"action": {"save-thing"}, "thing_kind": {"world_item"}, "thing_name": {"Retained Name"}, "item_kind": {"takeable"}, "short_description": {"short text"}, "full_description": {""}}, "empty full description"},
	} {
		rec := postForm(t, e.handler, tt.target+"/work", tt.form, "http://example.com", true)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tt.want) || !strings.Contains(rec.Body.String(), "Retained Name") || rec.Header().Get("HX-Retarget") != "#workspace" {
			t.Fatalf("error was not preserved: %d %s", rec.Code, rec.Body.String())
		}
	}
	locations, _ := e.locations.List()
	if len(locations) != 1 {
		t.Fatal("invalid creation wrote an entity")
	}
	postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {"room:" + room}}, "http://example.com", false)
	for _, target := range []string{locURL, roomURL} {
		parent := hub
		if target == roomURL {
			parent = loc
		}
		rec := postForm(t, e.handler, target+"/work", url.Values{"action": {"unplace"}, "parent_id": {parent}}, "http://example.com", true)
		if !strings.Contains(rec.Body.String(), "starting point or Hub arrival") {
			t.Fatal("play settings guard bypassed")
		}
	}
	item, err := e.items.Create(testWorldItemInput("Moved Item"))
	if err != nil {
		t.Fatal(err)
	}
	e.contents.PlaceIn("world_item", item.ID, "location", loc)
	rec := postForm(t, e.handler, roomURL+"/work", url.Values{"action": {"remove-thing"}, "thing_kind": {"world_item"}, "thing_id": {item.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "no longer in this cell") {
		t.Fatal("stale contents removal accepted")
	}
	rec = postForm(t, e.handler, roomURL+"/work", url.Values{"action": {"save-thing"}, "thing_kind": {"world_item"}, "thing_id": {item.ID}, "thing_name": {"Unexpected"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "no longer in this cell") {
		t.Fatal("stale contents update accepted")
	}
	after, _ := e.items.Get(item.ID)
	if after.Name != "Moved Item" {
		t.Fatal("stale form overwrote item")
	}
	if rec := getRequest(t, e.handler, "/content/rooms/missing", map[string]string{"HX-Request": "true"}); rec.Code != http.StatusNotFound {
		t.Fatal("missing cell silently selected another")
	}
}

func TestDrillHTMXNavigationAndReturn(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, room := e.oneRoomWorld(t)
	for _, target := range []string{"/content/hubs", "/library", entityURL("hub", hub), entityURL("location", loc), entityURL("room", room)} {
		rec := getRequest(t, e.handler, target, map[string]string{"HX-Request": "true", "HX-Target": "locations-editor"})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `class="header-tabs"`) || strings.Contains(rec.Body.String(), "<!doctype html>") {
			t.Fatal("navigation did not return workspace", target)
		}
		if strings.Contains(target, "/content/") && rec.Header().Get("HX-Retarget") != "#workspace" {
			t.Fatal("legacy catalog target not corrected")
		}
	}
	rec := postForm(t, e.handler, entityURL("room", room)+"/work", url.Values{"action": {"save-details"}, "name": {"Renamed Room"}, "description": {"(Placeholder)"}}, "http://example.com", true)
	var location map[string]string
	if err := json.Unmarshal([]byte(rec.Header().Get("HX-Location")), &location); err != nil {
		t.Fatal(err)
	}
	if location["target"] != "#workspace" || location["path"] != entityURL("room", room)+"?saved=1" || rec.Header().Get("HX-Trigger") != "contentSaved" {
		t.Fatal(location)
	}
	for _, target := range []string{"/place/locations?parent_id=" + hub, "/place/rooms?parent_id=" + loc, "/place/world-items?container=room:" + room, entityURL("room", room) + "?details=1"} {
		if rec := getRequest(t, e.handler, target, nil); rec.Code != http.StatusOK {
			t.Fatal("legacy route failed", target)
		}
	}
	rec = postForm(t, e.handler, entityURL("room", room)+"/work", url.Values{"action": {"save-details"}, "name": {"Blocked"}, "description": {"(Placeholder)"}}, "https://unrelated.example", true)
	if rec.Code != http.StatusForbidden {
		t.Fatal("new route bypassed same-origin protection")
	}
}

func TestDrillWorldSwitchTargetsLoadedWorld(t *testing.T) {
	registry, _, mainDir := newWorldsTestRegistry(t)
	hub, err := newHubStore(mainDir).Create("Original Hub")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Copy("main", "variant"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Load("variant"); err != nil {
		t.Fatal(err)
	}
	target := entityURL("hub", hub.ID)
	rec := postForm(t, registry, target+"/work", url.Values{"action": {"save-details"}, "name": {"Variant Hub"}}, "http://example.com", false)
	if rec.Code != http.StatusSeeOther {
		t.Fatal(rec.Code, rec.Body.String())
	}
	original, _ := newHubStore(mainDir).Get(hub.ID)
	variant, _ := newHubStore(filepath.Join(registry.root, "variant")).Get(hub.ID)
	if original.Name != "Original Hub" || variant.Name != "Variant Hub" {
		t.Fatal("contextual save crossed world boundaries")
	}
	rec = getRequest(t, registry, target, nil)
	if !strings.Contains(rec.Body.String(), "Variant Hub") || !strings.Contains(rec.Body.String(), "variant") {
		t.Fatal("loaded-world navigation lost context")
	}
}

func TestDrillEntryAndOccupancyGuards(t *testing.T) {
	e := newContentsTestEditor(t)
	_, loc, room := e.oneRoomWorld(t)
	e.entries.Set(loc, room)
	target := entityURL("room", room)
	rec := postForm(t, e.handler, target+"/work", url.Values{"action": {"unplace"}, "parent_id": {loc}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Clear this Room as the Location entry") {
		t.Fatal("entry safeguard missing")
	}
	other, _ := e.rooms.Create(describedInput{Name: "Other Room", Description: "(Placeholder)"})
	e.roomAssignments.Assign(other.ID, loc)
	e.roomPlacements.Place(other.ID, loc, 1, 0)
	rec = postForm(t, e.handler, target+"/work", url.Values{"action": {"position"}, "parent_id": {loc}, "x": {"1"}, "y": {"0"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "That cell is occupied") {
		t.Fatal("occupancy safeguard missing")
	}
	placements, _ := e.roomPlacements.List()
	for _, p := range placements {
		if p.RoomID == room && (p.X != 0 || p.Y != 0) {
			t.Fatal("blocked reposition changed placement")
		}
	}
}

func TestDrillDragMovesAndGuards(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, room := e.oneRoomWorld(t)
	for _, tc := range []struct{ kind, parent, child string }{{"hub", hub, loc}, {"location", loc, room}} {
		target := entityURL(tc.kind, tc.parent)
		drillCheck(t, e, target, `draggable="true"`, `data-move-url="`+target+`/work"`)
		move := url.Values{"child_id": {tc.child}, "source_x": {"0"}, "source_y": {"0"}, "x": {"2"}, "y": {"3"}}
		if got := drillSubmit(t, e, target, "move-child", move); got != target {
			t.Fatalf("move left parent grid: %s", got)
		}
		drillCheck(t, e, target, `data-child-id="`+tc.child+`" data-x="2" data-y="3"`)
		// Replaying an old drag must not undo a newer placement.
		rec := postForm(t, e.handler, target+"/work", move, "http://example.com", true)
		if !strings.Contains(rec.Body.String(), "has moved") {
			t.Fatal(rec.Body.String())
		}
		drillSubmit(t, e, target, "create-child", url.Values{"child_name": {"Other"}, "child_description": {"(Placeholder)"}, "x": {"4"}, "y": {"3"}})
		move.Set("source_x", "2")
		move.Set("source_y", "3")
		move.Set("x", "4")
		rec = postForm(t, e.handler, target+"/work", move, "http://example.com", false)
		if !strings.Contains(rec.Body.String(), "cell is occupied") {
			t.Fatal(rec.Body.String())
		}
		move.Set("x", "-1")
		rec = postForm(t, e.handler, target+"/work", move, "http://example.com", false)
		if rec.Code == http.StatusSeeOther {
			t.Fatal("accepted negative coordinate")
		}
		move.Set("child_id", hub)
		rec = postForm(t, e.handler, target+"/work", move, "http://example.com", false)
		if !strings.Contains(rec.Body.String(), "does not belong") {
			t.Fatal(rec.Body.String())
		}
		drillCheck(t, e, target, `data-child-id="`+tc.child+`" data-x="2" data-y="3"`)
		body := drillCheck(t, e, target+"?x=1&y=1", `data-focus-create`, "Create here")
		if strings.Index(body, `id="cell-editor"`) > strings.Index(body, `class="report drill-grid"`) {
			t.Fatal("selected cell form hidden below grid")
		}
	}
}
