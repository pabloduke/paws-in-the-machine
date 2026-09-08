package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestEditorAuthoredPlayThrough(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, location, room := e.oneRoomWorld(t)
	if _, err := e.entries.Set(location, room); err != nil {
		t.Fatal(err)
	}
	item, err := e.items.Create(worldItemInput{Name: "Test Item", Kind: "takeable", ShortDescription: "short text", FullDescription: "full text"})
	if err != nil {
		t.Fatal(err)
	}
	rec := postForm(t, e.handler, "/place/world-items/add", url.Values{"container": {"room:" + room}, "entity_id": {item.ID}}, "http://example.com", false)
	// Use the same public cell-contents route as the editor form.
	if rec.Code >= 400 {
		t.Fatalf("place item: %d %s", rec.Code, rec.Body.String())
	}
	// Explicitly verify the relation instead of relying on the response code.
	placed, _ := e.contents.List()
	if len(placed) == 0 {
		t.Fatal("contents form did not place the item")
	}
	for _, post := range []struct{ path, cell string }{
		{"/overview/play-settings", "location:" + location},
		{"/content/hubs/" + hub + "/arrival", location},
	} {
		rec := postForm(t, e.handler, post.path, url.Values{"cell": {post.cell}}, "http://example.com", false)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("%s: %d %s", post.path, rec.Code, rec.Body.String())
		}
	}
	dir := filepath.Dir(e.rooms.path)
	r := game.LoadAuthored(dir)
	if !r.Ready() {
		t.Fatal(r.DiagnosticText())
	}
	overview := getRequest(t, e.handler, "/overview", nil).Body.String()
	if !strings.Contains(overview, "Ready to play") || !strings.Contains(overview, `value="location:`+location+`" selected`) {
		t.Fatal("overview lacks ready state or selected start")
	}
	hubPage := getRequest(t, e.handler, "/content/hubs/"+hub, nil).Body.String()
	if !strings.Contains(hubPage, `value="`+location+`" selected`) {
		t.Fatal("hub lacks selected arrival")
	}
	eng := engine.New(r.World)
	eng.Execute("enter")
	if r.World.Room().ID != "room:"+room {
		t.Fatal("could not enter authored interior")
	}
	if got := eng.Execute("examine Test Item"); got != "full text" {
		t.Fatal(got)
	}
	eng.Execute("take Test Item")
	if !r.World.Carried(r.World.FindID("world_item:" + item.ID)) {
		t.Fatal("could not take editor item")
	}
	eng.Execute("drop Test Item")
	eng.Execute("out")
	if r.World.Room().ID != "location:"+location {
		t.Fatal("could not return outside")
	}
	// Changing the start to an interior protects its exterior ancestry too.
	postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {"room:" + room}}, "http://example.com", false)
	for _, post := range []struct {
		path string
		form url.Values
	}{
		{"/place/locations/unassign", url.Values{"parent_id": {hub}, "unassign_entity_id": {location}}},
		{"/place/rooms/unassign", url.Values{"parent_id": {location}, "unassign_entity_id": {room}}},
		{"/content/rooms/" + room + "/delete", nil},
		{"/content/locations/" + location + "/delete", nil},
		{"/content/hubs/" + hub + "/delete", nil},
	} {
		rec := postForm(t, e.handler, post.path, post.form, "http://example.com", false)
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "starting point or Hub arrival") {
			t.Fatalf("reference guard: %d %s", rec.Code, rec.Body.String())
		}
	}
	before, err := os.ReadFile(filepath.Join(dir, "play_settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	rec = postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {"room:" + item.ID}}, "http://example.com", false)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid starting cell: %d", rec.Code)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "play_settings.json"))
	if string(before) != string(after) {
		t.Fatal("invalid settings changed disk")
	}
	// Clear settings explicitly and verify the original unplacement path works.
	postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {""}}, "http://example.com", false)
	postForm(t, e.handler, "/content/hubs/"+hub+"/arrival", url.Values{"cell": {""}}, "http://example.com", false)
	rec = postForm(t, e.handler, "/place/locations/unassign", url.Values{"parent_id": {hub}, "unassign_entity_id": {location}}, "http://example.com", false)
	if rec.Code >= 400 {
		t.Fatalf("clear did not release reference: %s", rec.Body.String())
	}
	placements, _ := e.locationPlacements.List()
	if len(placements) != 0 {
		t.Fatal("location remained placed after clearing settings")
	}
}

func TestPlaySettingsRecoveryCopyAndWorldCopy(t *testing.T) {
	registry, _, dir := newWorldsTestRegistry(t)
	store := newPlaySettingsStore(dir)
	id := "00000000-0000-4000-8000-000000000001"
	if err := store.Update(func(s *content.PlaySettings) error { s.Start = &content.CellRef{Kind: "location", ID: id}; return nil }); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(store.path)
	if err := store.Update(func(s *content.PlaySettings) error { s.Start = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	backup, _ := os.ReadFile(store.path + ".bak")
	if string(backup) != string(original) {
		t.Fatal("recovery copy not preserved")
	}
	if err := registry.Copy("main", "copy"); err != nil {
		t.Fatal(err)
	}
	current, _ := os.ReadFile(store.path)
	copied, _ := os.ReadFile(filepath.Join(registry.root, "copy", "play_settings.json"))
	if string(copied) != string(current) {
		t.Fatal("world copy omitted play settings")
	}
	if _, err := os.Stat(filepath.Join(registry.root, "copy", "play_settings.json.bak")); !os.IsNotExist(err) {
		t.Fatal("world copy included recovery file")
	}
}

func TestPlayReferenceErrorRendersForHTMX(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, location, _ := e.oneRoomWorld(t)
	postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {"location:" + location}}, "http://example.com", false)
	for _, post := range []struct {
		path     string
		form     url.Values
		fragment string
	}{
		{"/place/locations/unassign", url.Values{"parent_id": {hub}, "unassign_entity_id": {location}}, "grid"},
		{"/content/hubs/" + hub + "/delete", nil, "hub-editor"},
		{"/content/locations/" + location + "/delete", nil, "locations-editor"},
	} {
		rec := postForm(t, e.handler, post.path, post.form, "http://example.com", true)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "starting point or Hub arrival") || !strings.Contains(rec.Body.String(), post.fragment) {
			t.Fatalf("HTMX guard did not render: %d %s", rec.Code, rec.Body.String())
		}
	}
}
