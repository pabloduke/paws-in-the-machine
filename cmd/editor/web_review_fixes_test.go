package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

// An unassigned Location is a valid resting state, and so is a Room
// assigned to one. Neither may vanish from the editor: a designer must
// be able to build an interior before deciding where the building goes.
func TestOrphanLocationKeepsItsRoomsVisibleAndAuthorable(t *testing.T) {
	e := newContentsTestEditor(t)
	location, _ := e.locations.Create(describedInput{Name: "Orphan Tower", Description: "Description"})
	room, _ := e.rooms.Create(describedInput{Name: "Tower Lobby", Description: "Description"})
	if _, err := e.roomAssignments.Assign(room.ID, location.ID); err != nil {
		t.Fatalf("assign room: %v", err)
	}
	if _, err := e.roomPlacements.Place(room.ID, location.ID, 0, 0); err != nil {
		t.Fatalf("place room: %v", err)
	}

	overview := getRequest(t, e.handler, "/overview", nil).Body.String()
	for _, want := range []string{"Orphan Tower", "Tower Lobby"} {
		if !strings.Contains(overview, want) {
			t.Errorf("Overview does not draw %q, but it exists", want)
		}
	}
	if strings.Contains(overview, "Rooms in no Location") && strings.Contains(overview, "Tower Lobby") {
		// The Room has a Location; only its Location lacks a Hub.
		if strings.Contains(overview, "Rooms in no Location</a> (1)") {
			t.Error("a Room owned by an orphan Location was miscounted as having no Location")
		}
	}

	picker := getRequest(t, e.handler, "/place/world-items", nil).Body.String()
	for _, want := range []string{"Orphan Tower", "Tower Lobby"} {
		if !strings.Contains(picker, want) {
			t.Errorf("cell picker omits %q, so nothing can be placed in it", want)
		}
	}

	// Contents already inside such a cell must still be listed.
	item, _ := e.items.Create(worldItemInput{Name: "Reception Desk", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})
	cell := gamecontent.ContainerKindRoom + ":" + room.ID
	rec := postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {cell}, "entity_id": {item.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Put Reception Desk here.") {
		t.Fatalf("could not place into an orphan subtree: %s", rec.Body.String())
	}
	held := getRequest(t, e.handler, "/place/world-items?container="+url.QueryEscape(cell), nil).Body.String()
	if !strings.Contains(held, "Reception Desk") {
		t.Error("contents of a cell under an orphan Location are not listed")
	}
	if !strings.Contains(getRequest(t, e.handler, "/overview", nil).Body.String(), "Reception Desk") {
		t.Error("Overview omits contents held under an orphan Location")
	}
}

// The remove form names the cell it was rendered for. If the entity has
// moved since, the stale submission must not delete its new placement.
func TestRemoveIsScopedToTheCellTheFormShowed(t *testing.T) {
	e := newContentsTestEditor(t)
	_, locationID, roomID := e.oneRoomWorld(t)
	item, _ := e.items.Create(worldItemInput{Name: "Mug", Kind: "fixed", ShortDescription: "s", FullDescription: "f"})
	roomCell := gamecontent.ContainerKindRoom + ":" + roomID
	locationCell := gamecontent.ContainerKindLocation + ":" + locationID

	postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {roomCell}, "entity_id": {item.ID}}, "http://example.com", true)
	postForm(t, e.handler, "/place/world-items/add",
		url.Values{"container": {locationCell}, "entity_id": {item.ID}}, "http://example.com", true)

	// A stale screen still showing the Room submits its remove button.
	rec := postForm(t, e.handler, "/place/world-items/remove",
		url.Values{"container": {roomCell}, "entity_id": {item.ID}}, "http://example.com", true)
	records, err := e.contents.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(records) != 1 || records[0].ParentID != locationID {
		t.Fatalf("a stale remove destroyed the current placement: %+v", records)
	}
	body := rec.Body.String()
	if strings.Contains(body, "still exists in the catalog") {
		t.Error("a stale remove reported success")
	}
	if !strings.Contains(body, "no longer") && !strings.Contains(body, "moved") {
		t.Errorf("a stale remove gave no actionable explanation: %s", body)
	}
}

// The write gate and the Overview must agree about whether the authored
// graph is valid; contents were missing from the gate.
func TestWriteGateRefusesWhileContentsAreBroken(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, roomID := e.oneRoomWorld(t)
	missingEntity := "99999999-9999-4999-8999-999999999999"
	if _, err := e.contents.PlaceIn(gamecontent.ContentKindWorldItem, missingEntity,
		gamecontent.ContainerKindRoom, roomID); err != nil {
		t.Fatalf("seed broken contents: %v", err)
	}

	before, _ := e.hubs.List()
	rec := postForm(t, e.handler, "/content/hubs", url.Values{"name": {"New Hub"}}, "http://example.com", true)
	if rec.Code != http.StatusConflict {
		t.Fatalf("protected POST status = %d, want 409 while contents are broken", rec.Code)
	}
	after, _ := e.hubs.List()
	if len(after) != len(before) {
		t.Fatalf("a refused write still mutated storage: %d -> %d", len(before), len(after))
	}
}

// The Overview promises every problem at once. The network and access
// validators stop at their first bad record, so more than one problem
// per catalog must still be reported.
func TestOverviewCollectsEveryNetworkAndAccessProblem(t *testing.T) {
	e := newContentsTestEditor(t)
	dir := t.TempDir()
	_ = dir
	networks, terminals := e.networks, e.terminals
	users, access, assignments := e.users, e.access, e.assignments

	firstTerminal, _ := terminals.Create(terminalInput{HostName: "one"})
	secondTerminal, _ := terminals.Create(terminalInput{HostName: "two"})
	// Two independent dangling corporation references.
	assignments.Assign(firstTerminal.ID, "11111111-1111-4111-8111-111111111111")
	assignments.Assign(secondTerminal.ID, "22222222-2222-4222-8222-222222222222")
	_ = networks

	firstUser, _ := users.Create(userInput{Username: "alice", Password: "fictional"})
	secondUser, _ := users.Create(userInput{Username: "bob", Password: "fictional"})
	// Two independent dangling terminal references.
	access.Grant(firstUser.ID, "33333333-3333-4333-8333-333333333333")
	access.Grant(secondUser.ID, "44444444-4444-4444-8444-444444444444")

	body := getRequest(t, e.handler, "/overview", nil).Body.String()
	if !strings.Contains(body, "Problems (4)") {
		t.Fatalf("Overview did not collect all four problems: %s", problemsSection(body))
	}
	for _, want := range []string{"one", "two", "alice", "bob"} {
		if !strings.Contains(body, want) {
			t.Errorf("problem report does not name %q", want)
		}
	}
}

func problemsSection(body string) string {
	start := strings.Index(body, "problems-title")
	if start < 0 {
		return "(no problems section rendered)"
	}
	end := start + 1200
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}
