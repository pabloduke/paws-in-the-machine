package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

type contentsTestEditor struct {
	handler             http.Handler
	hubs                *hubStore
	locations           *locationStore
	rooms               *roomStore
	items               *worldItemStore
	npcs                *npcStore
	terminals           *terminalStore
	contents            *contentsStore
	networks            *hostNetworkStore
	assignments         *networkAssignmentStore
	users               *userStore
	access              *terminalAccessStore
	locationAssignments *locationAssignmentStore
	roomAssignments     *roomAssignmentStore
	roomPlacements      *roomPlacementStore
	locationPlacements  *locationPlacementStore
	entries             *locationEntryStore
}

func newContentsTestEditor(t *testing.T) contentsTestEditor {
	t.Helper()
	dir := t.TempDir()
	items, hubs, rooms := newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir)
	npcs, terminals := newNPCStore(dir), newTerminalStore(dir)
	networks, assignments := newHostNetworkStore(dir), newNetworkAssignmentStore(dir)
	users, access := newUserStore(dir), newTerminalAccessStore(dir)
	handler := newEditorHandler(items, hubs, rooms, npcs, terminals,
		networks, assignments, users, access, newRoomPlacementStore(dir))
	return contentsTestEditor{
		handler: handler, hubs: hubs, locations: newLocationStore(dir), rooms: rooms,
		items: items, npcs: npcs, terminals: terminals, contents: newContentsStore(dir),
		networks: networks, assignments: assignments, users: users, access: access,
		locationAssignments: newLocationAssignmentStore(dir), roomAssignments: newRoomAssignmentStore(dir),
		roomPlacements: newRoomPlacementStore(dir), locationPlacements: newLocationPlacementStore(dir),
		entries: newLocationEntryStore(dir),
	}
}

// oneRoomWorld builds Hub → Location → Room, all assigned and placed.
func (e contentsTestEditor) oneRoomWorld(t *testing.T) (hubID, locationID, roomID string) {
	t.Helper()
	hub, err := e.hubs.Create("Neighborhood")
	if err != nil {
		t.Fatalf("create hub: %v", err)
	}
	location, err := e.locations.Create(describedInput{Name: "Coffee Shop", Description: "Description"})
	if err != nil {
		t.Fatalf("create location: %v", err)
	}
	room, err := e.rooms.Create(describedInput{Name: "Back Room", Description: "Description"})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := e.locationAssignments.Assign(location.ID, hub.ID); err != nil {
		t.Fatalf("assign location: %v", err)
	}
	if _, err := e.roomAssignments.Assign(room.ID, location.ID); err != nil {
		t.Fatalf("assign room: %v", err)
	}
	if _, err := e.locationPlacements.Place(location.ID, hub.ID, 0, 0); err != nil {
		t.Fatalf("place location: %v", err)
	}
	if _, err := e.roomPlacements.Place(room.ID, location.ID, 0, 0); err != nil {
		t.Fatalf("place room: %v", err)
	}
	return hub.ID, location.ID, room.ID
}

func TestContentsScreenPutsAnItemInARoomAndTakesItBack(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, roomID := e.oneRoomWorld(t)
	item, _ := e.items.Create(worldItemInput{Name: "Post-it", Kind: "takeable", ShortDescription: "a note", FullDescription: "a note"})

	cell := gamecontent.ContainerKindRoom + ":" + roomID
	rec := getRequest(t, e.handler, "/place/world-items?container="+url.QueryEscape(cell), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Neighborhood › Coffee Shop › Back Room", "Post-it", "Put Here"} {
		if !strings.Contains(body, want) {
			t.Fatalf("contents screen missing %q: %s", want, body)
		}
	}

	rec = postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {cell}, "entity_id": {item.ID}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Put Post-it here.") {
		t.Fatalf("add failed: %d %s", rec.Code, rec.Body.String())
	}
	records, _ := e.contents.List()
	if len(records) != 1 || records[0].EntityID != item.ID || records[0].ParentID != roomID {
		t.Fatalf("contents = %+v", records)
	}
	if records[0].ParentKind != gamecontent.ContainerKindRoom || records[0].EntityKind != gamecontent.ContentKindWorldItem {
		t.Fatalf("kinds not recorded: %+v", records[0])
	}

	rec = postForm(t, e.handler, "/place/world-items/remove",
		url.Values{"container": {cell}, "entity_id": {item.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "still exists in the catalog") {
		t.Fatalf("remove failed: %s", rec.Body.String())
	}
	if records, _ = e.contents.List(); len(records) != 0 {
		t.Fatalf("contents after remove = %+v", records)
	}
	if _, err := e.items.Get(item.ID); err != nil {
		t.Fatalf("removing from a cell deleted the item: %v", err)
	}
}

func TestContentsMoveRatherThanDuplicate(t *testing.T) {
	e := newContentsTestEditor(t)
	_, locationID, roomID := e.oneRoomWorld(t)
	npc, _ := e.npcs.Create(describedInput{Name: "Barista", Description: "Description"})

	roomCell := gamecontent.ContainerKindRoom + ":" + roomID
	locationCell := gamecontent.ContainerKindLocation + ":" + locationID
	postForm(t, e.handler, "/place/npcs/add", url.Values{"container": {roomCell}, "entity_id": {npc.ID}}, "http://example.com", true)
	rec := postForm(t, e.handler, "/place/npcs/add", url.Values{"container": {locationCell}, "entity_id": {npc.ID}}, "http://example.com", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("move failed: %d", rec.Code)
	}
	records, _ := e.contents.List()
	if len(records) != 1 {
		t.Fatalf("moving an NPC duplicated it: %+v", records)
	}
	if records[0].ParentID != locationID || records[0].ParentKind != gamecontent.ContainerKindLocation {
		t.Fatalf("NPC did not move: %+v", records[0])
	}
}

func TestContentsRefuseUnknownCellOrEntity(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, roomID := e.oneRoomWorld(t)
	item, _ := e.items.Create(worldItemInput{Name: "Mug", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})
	cell := gamecontent.ContainerKindRoom + ":" + roomID

	rec := postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {"room:not-a-uuid"}, "entity_id": {item.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "no longer exists") {
		t.Fatalf("unknown cell accepted: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {cell}, "entity_id": {"6f1d5a2c-0000-4000-8000-000000000000"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Choose an existing World Item.") {
		t.Fatalf("unknown entity accepted: %s", rec.Body.String())
	}
	if records, _ := e.contents.List(); len(records) != 0 {
		t.Fatalf("rejected writes still persisted: %+v", records)
	}
}

func TestDeleteRefusedWhileSomethingHoldsIt(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, roomID := e.oneRoomWorld(t)
	cell := gamecontent.ContainerKindRoom + ":" + roomID

	item, _ := e.items.Create(worldItemInput{Name: "Mug", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})
	npc, _ := e.npcs.Create(describedInput{Name: "Barista", Description: "Description"})
	terminal, _ := e.terminals.Create(terminalInput{HostName: "till"})
	postForm(t, e.handler, "/place/world-items/add", url.Values{"container": {cell}, "entity_id": {item.ID}}, "http://example.com", true)
	postForm(t, e.handler, "/place/npcs/add", url.Values{"container": {cell}, "entity_id": {npc.ID}}, "http://example.com", true)
	postForm(t, e.handler, "/place/terminals/add", url.Values{"container": {cell}, "entity_id": {terminal.ID}}, "http://example.com", true)

	checks := []struct{ path, message string }{
		{"/content/world-items/" + item.ID + "/delete", "Remove this World Item from Back Room"},
		{"/content/npcs/" + npc.ID + "/delete", "Remove this NPC from Back Room"},
		{"/content/terminals/" + terminal.ID + "/delete", "Remove this Terminal from Back Room"},
		{"/content/rooms/" + roomID + "/delete", "Take everything out of this Room"},
	}
	for _, check := range checks {
		rec := postForm(t, e.handler, check.path, url.Values{}, "http://example.com", true)
		if !strings.Contains(rec.Body.String(), check.message) {
			t.Fatalf("delete %s not guarded, want %q: %s", check.path, check.message, rec.Body.String())
		}
	}
	if _, err := e.items.Get(item.ID); err != nil {
		t.Fatalf("guarded item was deleted anyway: %v", err)
	}
	if _, err := e.rooms.Get(roomID); err != nil {
		t.Fatalf("guarded room was deleted anyway: %v", err)
	}
}

func TestOverviewShowsTheTreeAndReportsNoProblems(t *testing.T) {
	e := newContentsTestEditor(t)
	_, locationID, roomID := e.oneRoomWorld(t)
	if _, err := e.entries.Set(locationID, roomID); err != nil {
		t.Fatalf("set entry: %v", err)
	}
	item, _ := e.items.Create(worldItemInput{Name: "Post-it", Kind: "takeable", ShortDescription: "s", FullDescription: "f"})
	postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {gamecontent.ContainerKindRoom + ":" + roomID}, "entity_id": {item.ID}}, "http://example.com", true)

	rec := getRequest(t, e.handler, "/overview", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"No problems found", "Neighborhood", "Coffee Shop", "Back Room", "Post-it", "entry",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("overview missing %q: %s", want, body)
		}
	}
}

func TestOverviewReportsEveryProblemNotJustTheFirst(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, _ := e.hubs.Create("Hub")
	otherHub, _ := e.hubs.Create("Other Hub")
	first, _ := e.locations.Create(describedInput{Name: "First", Description: "Description"})
	second, _ := e.locations.Create(describedInput{Name: "Second", Description: "Description"})
	// Two independent placement/ownership mismatches.
	e.locationAssignments.Assign(first.ID, hub.ID)
	e.locationPlacements.Place(first.ID, otherHub.ID, 0, 0)
	e.locationAssignments.Assign(second.ID, hub.ID)
	e.locationPlacements.Place(second.ID, otherHub.ID, 1, 0)

	rec := getRequest(t, e.handler, "/overview", nil)
	body := rec.Body.String()
	if !strings.Contains(body, "Problems (2)") {
		t.Fatalf("overview did not list both problems: %s", body)
	}
	for _, want := range []string{"First", "Second"} {
		if !strings.Contains(body, want) {
			t.Fatalf("problem report missing %q", want)
		}
	}
}

func TestOverviewListsWhatIsStillWaiting(t *testing.T) {
	e := newContentsTestEditor(t)
	e.hubs.Create("Hub")
	e.locations.Create(describedInput{Name: "Orphan Location", Description: "Description"})
	e.rooms.Create(describedInput{Name: "Orphan Room", Description: "Description"})
	e.items.Create(worldItemInput{Name: "Loose Mug", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})

	rec := getRequest(t, e.handler, "/overview", nil)
	body := rec.Body.String()
	for _, want := range []string{
		"Still waiting for you", "Locations in no Hub", "Orphan Location",
		"Rooms in no Location", "Orphan Room", "World Items in no cell", "Loose Mug",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("overview advisories missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "Problems (") {
		t.Fatal("unassigned entities were reported as problems; they are a valid resting state")
	}
}

func TestContentsScreenExplainsAnEmptyWorld(t *testing.T) {
	e := newContentsTestEditor(t)
	rec := getRequest(t, e.handler, "/place/npcs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "There is nowhere to put NPCs yet.") {
		t.Fatalf("empty-world guidance missing: %s", rec.Body.String())
	}
}

func TestContentsScreenWritesRequireSameOrigin(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, roomID := e.oneRoomWorld(t)
	item, _ := e.items.Create(worldItemInput{Name: "Mug", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})
	rec := postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {gamecontent.ContainerKindRoom + ":" + roomID}, "entity_id": {item.ID}}, "", true)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin write status = %d, want 403", rec.Code)
	}
}

// Tab navigation and the cell picker target #workspace, so an HTMX GET
// must return the whole screen. Returning only the body dropped the
// navigation and the #contents-workspace wrapper, which silently broke
// every form on the screen because their hx-target no longer existed.
func TestContentsHTMXNavigationReturnsTheWholeWorkspace(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, roomID := e.oneRoomWorld(t)
	item, _ := e.items.Create(worldItemInput{Name: "Mug", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})
	cell := gamecontent.ContainerKindRoom + ":" + roomID

	for _, path := range []string{
		"/place/world-items",
		"/place/world-items?container=" + url.QueryEscape(cell),
		"/place/npcs",
		"/place/terminals",
	} {
		rec := getRequest(t, e.handler, path, map[string]string{"HX-Request": "true"})
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
		body := rec.Body.String()
		for _, want := range []string{
			`id="contents-workspace"`,
			`class="header-tabs"`,
			`class="detail-tabs"`,
			`aria-current="page"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("HTMX GET %s dropped %q from the workspace", path, want)
			}
		}
		if strings.Contains(body, "<!doctype html>") {
			t.Fatalf("HTMX GET %s returned a full document instead of the workspace", path)
		}
	}

	// A form submission still swaps only the body, into that wrapper.
	rec := postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {cell}, "entity_id": {item.ID}}, "http://example.com", true)
	body := rec.Body.String()
	if strings.Contains(body, `class="header-tabs"`) {
		t.Fatal("a form submission re-rendered the navigation instead of just the body")
	}
	if !strings.Contains(body, "Put Mug here.") {
		t.Fatalf("add did not confirm: %s", body)
	}
}
