package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type spatialTestEditor struct {
	handler             http.Handler
	hubs                *hubStore
	locations           *locationStore
	rooms               *roomStore
	locationAssignments *locationAssignmentStore
	roomAssignments     *roomAssignmentStore
	locationPlacements  *locationPlacementStore
	roomPlacements      *roomPlacementStore
	entries             *locationEntryStore
}

func newSpatialTestEditor(t *testing.T) spatialTestEditor {
	t.Helper()
	dir := t.TempDir()
	hubs, rooms := newHubStore(dir), newRoomStore(dir)
	handler := newEditorHandler(newWorldItemStore(dir), hubs, rooms, newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir))
	return spatialTestEditor{
		handler: handler, hubs: hubs, locations: newLocationStore(dir), rooms: rooms,
		locationAssignments: newLocationAssignmentStore(dir), roomAssignments: newRoomAssignmentStore(dir),
		locationPlacements: newLocationPlacementStore(dir), roomPlacements: newRoomPlacementStore(dir), entries: newLocationEntryStore(dir),
	}
}

func TestNestedHubLocationAndLocationRoomCreation(t *testing.T) {
	e := newSpatialTestEditor(t)
	hub, _ := e.hubs.Create("Hub")
	rec := getRequest(t, e.handler, "/content/hubs/"+hub.ID, nil)
	for _, want := range []string{"Details", "Locations"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("hub details missing %q", want)
		}
	}
	rec = postForm(t, e.handler, "/content/hubs/"+hub.ID+"/locations", url.Values{"name": {"Building"}, "description": {"Description"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Created and assigned Building.") || !strings.Contains(rec.Body.String(), "Unplaced") {
		t.Fatalf("nested location create failed: %d %s", rec.Code, rec.Body.String())
	}
	locations, _ := e.locations.List()
	if len(locations) != 1 {
		t.Fatalf("locations = %+v", locations)
	}
	locationID := locations[0].ID
	assignment, err := e.locationAssignments.Get(locationID)
	if err != nil || assignment.HubID != hub.ID {
		t.Fatalf("location assignment = %+v, %v", assignment, err)
	}

	rec = getRequest(t, e.handler, "/content/locations/"+locationID, nil)
	for _, want := range []string{"Details", "Rooms"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("location details missing %q", want)
		}
	}
	rec = postForm(t, e.handler, "/content/locations/"+locationID+"/rooms", url.Values{"name": {"Interior"}, "description": {"Description"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Created and assigned Interior.") || !strings.Contains(rec.Body.String(), "Unplaced") {
		t.Fatalf("nested room create failed: %d %s", rec.Code, rec.Body.String())
	}
	rooms, _ := e.rooms.List()
	assignmentRoom, err := e.roomAssignments.Get(rooms[0].ID)
	if err != nil || assignmentRoom.LocationID != locationID {
		t.Fatalf("room assignment = %+v, %v", assignmentRoom, err)
	}
}

func TestAssignExistingChildAndRequireUnplacementBeforeUnassign(t *testing.T) {
	e := newSpatialTestEditor(t)
	hub, _ := e.hubs.Create("Hub")
	location, _ := e.locations.Create(describedInput{Name: "Location", Description: "Description"})
	rec := postForm(t, e.handler, "/content/hubs/"+hub.ID+"/assign", url.Values{"child_id": {location.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Assigned Location.") {
		t.Fatalf("assign existing failed: %s", rec.Body.String())
	}
	if _, err := e.locationPlacements.Place(location.ID, hub.ID, 1, 1); err != nil {
		t.Fatal(err)
	}
	rec = postForm(t, e.handler, "/content/hubs/"+hub.ID+"/unassign/"+location.ID, url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Remove Location from the grid") {
		t.Fatalf("placed child unassigned: %s", rec.Body.String())
	}
	if _, err := e.locationPlacements.Unassign(location.ID); err != nil {
		t.Fatal(err)
	}
	rec = postForm(t, e.handler, "/content/hubs/"+hub.ID+"/unassign/"+location.ID, url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unassigned Location.") {
		t.Fatalf("unassign failed: %s", rec.Body.String())
	}
}

func TestLocationPlacementGridUsesOnlyAssignedLocations(t *testing.T) {
	e := newSpatialTestEditor(t)
	hub, _ := e.hubs.Create("Hub")
	first, _ := e.locations.Create(describedInput{Name: "First Location", Description: "Description"})
	second, _ := e.locations.Create(describedInput{Name: "Second Location", Description: "Description"})
	orphan, _ := e.locations.Create(describedInput{Name: "Orphan", Description: "Description"})
	_, _ = e.locationAssignments.Assign(first.ID, hub.ID)
	_, _ = e.locationAssignments.Assign(second.ID, hub.ID)

	page := getRequest(t, e.handler, "/place/locations?parent_id="+hub.ID, nil)
	for _, want := range []string{"Assign Locations", "First Location", "Second Location", "value=\"9,9\""} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("placement page missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), orphan.Name) {
		t.Fatal("orphan appeared in placement picker")
	}
	rec := postForm(t, e.handler, "/place/locations/place", url.Values{"parent_id": {hub.ID}, "entity_id": {first.ID}, "cell": {"2,3"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Placed First Location at 2, 3.") {
		t.Fatalf("place failed: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/place/locations/place", url.Values{"parent_id": {hub.ID}, "entity_id": {first.ID}, "x": {"11"}, "y": {"4"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "First Location at 11,4") || !strings.Contains(rec.Body.String(), "repeat(12") {
		t.Fatalf("move failed: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/place/locations/place", url.Values{"parent_id": {hub.ID}, "entity_id": {second.ID}, "x": {"11"}, "y": {"4"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "already occupied") {
		t.Fatalf("collision accepted: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/place/locations/unassign", url.Values{"parent_id": {hub.ID}, "unassign_entity_id": {first.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unplaced First Location.") {
		t.Fatalf("unplace failed: %s", rec.Body.String())
	}
	if assignment, err := e.locationAssignments.Get(first.ID); err != nil || assignment.HubID != hub.ID {
		t.Fatalf("unplacing removed ownership: %+v %v", assignment, err)
	}
}

func TestRoomPlacementAndEntryRoomLiveInLocationTab(t *testing.T) {
	e := newSpatialTestEditor(t)
	location, _ := e.locations.Create(describedInput{Name: "Location", Description: "Description"})
	other, _ := e.locations.Create(describedInput{Name: "Other", Description: "Description"})
	room, _ := e.rooms.Create(describedInput{Name: "Room", Description: "Description"})
	_, _ = e.roomAssignments.Assign(room.ID, location.ID)
	page := getRequest(t, e.handler, "/place/rooms?parent_id="+location.ID, nil)
	for _, want := range []string{"Assign Rooms", "Room", "value=\"4,4\""} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("room page missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Optional Entry Room") {
		t.Fatal("entry selector remained on Place screen")
	}
	rec := postForm(t, e.handler, "/place/rooms/place", url.Values{"parent_id": {location.ID}, "entity_id": {room.ID}, "cell": {"1,2"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Placed Room at 1, 2.") {
		t.Fatalf("room placement failed: %s", rec.Body.String())
	}
	rec = getRequest(t, e.handler, "/content/locations/"+location.ID+"/rooms", nil)
	if !strings.Contains(rec.Body.String(), "Optional Entry Room") {
		t.Fatalf("entry selector missing from Location Rooms tab: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/content/locations/"+location.ID+"/entry", url.Values{"entry_room_id": {room.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Entry Room saved.") {
		t.Fatalf("entry save failed: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/place/rooms/unassign", url.Values{"parent_id": {location.ID}, "unassign_entity_id": {room.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Clear this Room as the Location entry") {
		t.Fatalf("entry room unplaced: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/content/locations/"+location.ID+"/entry-clear", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Entry Room cleared.") {
		t.Fatalf("entry clear failed: %s", rec.Body.String())
	}
	rec = postForm(t, e.handler, "/place/rooms/place", url.Values{"parent_id": {other.ID}, "entity_id": {room.ID}, "cell": {"0,0"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Assign this Room to the selected Location") {
		t.Fatalf("foreign ownership accepted: %s", rec.Body.String())
	}
}

func TestSpatialDeletionGuardsUseOwnership(t *testing.T) {
	e := newSpatialTestEditor(t)
	hub, _ := e.hubs.Create("Hub")
	location, _ := e.locations.Create(describedInput{Name: "Location", Description: "Description"})
	room, _ := e.rooms.Create(describedInput{Name: "Room", Description: "Description"})
	_, _ = e.locationAssignments.Assign(location.ID, hub.ID)
	_, _ = e.roomAssignments.Assign(room.ID, location.ID)
	checks := []struct{ path, message string }{
		{"/content/rooms/" + room.ID + "/delete", "Unassign this Room"},
		{"/content/locations/" + location.ID + "/delete", "Unassign this Location"},
		{"/content/hubs/" + hub.ID + "/delete", "Unassign every Location"},
	}
	for _, check := range checks {
		rec := postForm(t, e.handler, check.path, url.Values{}, "http://example.com", true)
		if !strings.Contains(rec.Body.String(), check.message) {
			t.Fatalf("delete %s not guarded: %s", check.path, rec.Body.String())
		}
	}
	_, _ = e.locationAssignments.Unassign(location.ID)
	rec := postForm(t, e.handler, "/content/locations/"+location.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unassign every Room") {
		t.Fatalf("owned room guard missing: %s", rec.Body.String())
	}
}

func TestSpatialValidationRejectsPlacementOwnershipMismatchAndUnplacedEntry(t *testing.T) {
	e := newSpatialTestEditor(t)
	firstHub, _ := e.hubs.Create("First")
	secondHub, _ := e.hubs.Create("Second")
	location, _ := e.locations.Create(describedInput{Name: "Location", Description: "Description"})
	_, _ = e.locationAssignments.Assign(location.ID, firstHub.ID)
	_, _ = e.locationPlacements.Place(location.ID, secondHub.ID, 0, 0)
	rec := getRequest(t, e.handler, "/place/locations", nil)
	if !strings.Contains(rec.Body.String(), "but assigned to a different Hub") {
		t.Fatalf("mismatch not reported: %s", rec.Body.String())
	}

	e = newSpatialTestEditor(t)
	location, _ = e.locations.Create(describedInput{Name: "Location", Description: "Description"})
	room, _ := e.rooms.Create(describedInput{Name: "Room", Description: "Description"})
	_, _ = e.roomAssignments.Assign(room.ID, location.ID)
	_, _ = e.entries.Set(location.ID, room.ID)
	rec = getRequest(t, e.handler, "/place/rooms", nil)
	if !strings.Contains(rec.Body.String(), "but that Room is not placed in it") {
		t.Fatalf("unplaced entry not reported: %s", rec.Body.String())
	}
}
