package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func testRoomPlacementHandler(t *testing.T) (http.Handler, *hubStore, *roomStore, *roomPlacementStore) {
	t.Helper()
	dir := t.TempDir()
	hubs, rooms, placements := newHubStore(dir), newRoomStore(dir), newRoomPlacementStore(dir)
	handler := newEditorHandler(newWorldItemStore(dir), hubs, rooms, newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), placements)
	return handler, hubs, rooms, placements
}

func TestRoomPlacementGridPlaceMoveCollisionAndUnassign(t *testing.T) {
	handler, hubs, rooms, placements := testRoomPlacementHandler(t)
	hub, _ := hubs.Create("Hub")
	first, _ := rooms.Create(describedInput{Name: "First Room", Description: "Description"})
	second, _ := rooms.Create(describedInput{Name: "Second Room", Description: "Description"})

	page := getRequest(t, handler, "/place/rooms?hub_id="+hub.ID, nil)
	for _, want := range []string{"Assign Rooms", "First Room", "Second Room", "value=\"9,9\"", "Enter coordinates or select an empty grid cell"} {
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), want) {
			t.Fatalf("placement page missing %q: %d %s", want, page.Code, page.Body.String())
		}
	}
	rec := postForm(t, handler, "/place/rooms/place", url.Values{"hub_id": {hub.ID}, "room_id": {first.ID}, "cell": {"2,3"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Placed First Room at 2, 3.") {
		t.Fatalf("place failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = postForm(t, handler, "/place/rooms/place", url.Values{"hub_id": {hub.ID}, "room_id": {first.ID}, "x": {"11"}, "y": {"4"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "First Room at 11,4") || !strings.Contains(rec.Body.String(), "repeat(12") {
		t.Fatalf("move/expansion failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = postForm(t, handler, "/place/rooms/place", url.Values{"hub_id": {hub.ID}, "room_id": {second.ID}, "x": {"11"}, "y": {"4"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "already occupied") {
		t.Fatalf("collision accepted: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/place/rooms/unassign", url.Values{"hub_id": {hub.ID}, "unassign_room_id": {first.ID}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Unassigned First Room.") {
		t.Fatalf("unassign failed: %d %s", rec.Code, rec.Body.String())
	}
	got, err := placements.List()
	if err != nil || len(got) != 0 {
		t.Fatalf("placements after unassign = %+v, %v", got, err)
	}
}

func TestPlacedRoomAndHubDeletionAreGuarded(t *testing.T) {
	handler, hubs, rooms, placements := testRoomPlacementHandler(t)
	hub, _ := hubs.Create("Hub")
	room, _ := rooms.Create(describedInput{Name: "Room", Description: "Description"})
	if _, err := placements.Place(room.ID, hub.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	rec := postForm(t, handler, "/content/rooms/"+room.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unassign this Room") || rec.Header().Get("HX-Trigger") != "" {
		t.Fatalf("room delete not guarded: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/hubs/"+hub.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unassign every Room") || rec.Header().Get("HX-Trigger") != "" {
		t.Fatalf("hub delete not guarded: %s", rec.Body.String())
	}
}
