package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func apiRequest(t *testing.T, h http.Handler, method, path, revision, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Match", revision)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAPIRevisionWorldAndSharedCreation(t *testing.T) {
	e := newContentsTestEditor(t)
	_, loc, room := e.oneRoomWorld(t)
	root := "/api/v1/worlds/main"
	read := apiRequest(t, e.handler, "GET", root, "", "")
	rev := read.Header().Get("ETag")
	if read.Code != 200 || rev == "" {
		t.Fatal(read.Body.String())
	}
	route := root + "/rooms/" + room + "/actions"
	body := `{"action":"save-thing","thing_kind":"npc","thing_name":"Test NPC","thing_description":"(Placeholder)"}`
	w := apiRequest(t, e.handler, "POST", route, rev, body)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var result struct{ Created []string }
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result.Created) != 1 {
		t.Fatal(w.Body.String())
	}
	drillCheck(t, e, entityURL("room", room), "Test NPC")
	if apiRequest(t, e.handler, "POST", route, rev, body).Code != 409 {
		t.Fatal("accepted stale revision")
	}
	if apiRequest(t, e.handler, "POST", route, "", body).Code != 428 {
		t.Fatal("accepted missing revision")
	}
	if apiRequest(t, e.handler, "GET", "/api/v1/worlds/other", "", "").Code != 409 {
		t.Fatal("wrong world accepted")
	}
	rev = w.Header().Get("ETag")
	bad := apiRequest(t, e.handler, "POST", root+"/locations/"+loc+"/actions", rev, `{"action":"create-child","child_name":"Invalid","child_description":"(Placeholder)","x":"0","y":"0"}`)
	if bad.Code != 422 {
		t.Fatal(bad.Body.String())
	}
	if apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag") != rev {
		t.Fatal("failed validation changed world")
	}
	r := httptest.NewRequest("POST", "http://example.com"+route, strings.NewReader(body))
	r.Header.Set("Origin", "https://elsewhere.example")
	r.Header.Set("If-Match", rev)
	r.Header.Set("Content-Type", "application/json")
	out := httptest.NewRecorder()
	e.handler.ServeHTTP(out, r)
	if out.Code != 403 {
		t.Fatal("cross-origin mutation accepted")
	}
}
func TestFloorsConnectionsAndInteriorNavigation(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, room := e.oneRoomWorld(t)
	lower := entityURL("room", room)
	parent := entityURL("location", loc)
	upper := drillSubmit(t, e, parent, "create-child", url.Values{"child_name": {"Upper"}, "child_description": {"(Placeholder)"}, "x": {"0"}, "y": {"0"}, "z": {"1"}})
	upperID := upper[strings.LastIndex(upper, "/")+1:]
	basement := drillSubmit(t, e, parent, "create-child", url.Values{"child_name": {"Basement"}, "child_description": {"(Placeholder)"}, "x": {"0"}, "y": {"0"}, "z": {"-1"}})
	drillCheck(t, e, parent+"?z=1", "Upper", "Level 1")
	drillSubmit(t, e, parent, "entry", url.Values{"entry_id": {room}})
	drillSubmit(t, e, entityURL("hub", hub), "arrival", url.Values{"arrival_id": {loc}})
	postForm(t, e.handler, "/overview/play-settings", url.Values{"cell": {"location:" + loc}}, "http://example.com", false)
	c, err := game.ReadCatalogs(filepath.Dir(e.rooms.path))
	if err != nil {
		t.Fatal(err)
	}
	assembled := game.AssembleAuthored(c)
	if !assembled.Ready() {
		t.Fatal(assembled.DiagnosticText())
	}
	eng := engine.New(assembled.World)
	eng.Execute("enter")
	if assembled.World.Room().ID != "authored:room:"+room { // use suffix: runtime IDs are implementation-owned
		if !strings.HasSuffix(assembled.World.Room().ID, room) {
			t.Fatal("enter failed")
		}
	}
	before := assembled.World.Room().ID
	eng.Execute("up")
	if assembled.World.Room().ID != before {
		t.Fatal("stacked rooms auto-connected")
	}
	drillSubmit(t, e, lower, "connect-vertical", url.Values{"upper_id": {upperID}})
	drillSubmit(t, e, basement, "connect-vertical", url.Values{"upper_id": {room}})
	assembled = game.LoadAuthored(filepath.Dir(e.rooms.path))
	if !assembled.Ready() {
		t.Fatal(assembled.DiagnosticText())
	}
	eng = engine.New(assembled.World)
	for _, step := range []struct{ cmd, id string }{{"enter", room}, {"up", upperID}, {"down", room}, {"out", loc}} {
		eng.Execute(step.cmd)
		if !strings.HasSuffix(assembled.World.Room().ID, step.id) {
			t.Fatalf("%s failed", step.cmd)
		}
	}
	rec := postForm(t, e.handler, upper+"/work", url.Values{"action": {"position"}, "parent_id": {loc}, "x": {"1"}, "y": {"0"}, "z": {"1"}}, "http://example.com", false)
	if !strings.Contains(rec.Body.String(), "Remove this cell&#39;s vertical connections") {
		t.Fatal("connected room moved", rec.Body.String())
	}
}

func TestAPIEntityCRUDAndReferenceProtection(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, room := e.oneRoomWorld(t)
	root := "/api/v1/worlds/main"
	revision := func() string { return apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag") }
	w := apiRequest(t, e.handler, "POST", root+"/entities/npc", revision(), `{"name":"Draft NPC","description":"(Placeholder)"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var created struct{ Created []string }
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created.Created[0]
	read := apiRequest(t, e.handler, "GET", root+"/entities/npc/"+id, "", "")
	if !strings.Contains(read.Body.String(), "Draft NPC") {
		t.Fatal(read.Body.String())
	}
	w = apiRequest(t, e.handler, "PUT", root+"/entities/npc/"+id, revision(), `{"name":"Edited NPC","description":"Edited draft"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	drillSubmit(t, e, entityURL("room", room), "add-thing", url.Values{"thing_kind": {"npc"}, "thing_id": {id}})
	w = apiRequest(t, e.handler, "DELETE", root+"/entities/npc/"+id, revision(), `{}`)
	if w.Code != 422 {
		t.Fatal("deleted placed NPC", w.Body.String())
	}
	drillSubmit(t, e, entityURL("room", room), "remove-thing", url.Values{"thing_kind": {"npc"}, "thing_id": {id}})
	w = apiRequest(t, e.handler, "DELETE", root+"/entities/npc/"+id, revision(), `{}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if apiRequest(t, e.handler, "GET", root+"/entities/npc/"+id, "", "").Code != 404 {
		t.Fatal("deleted NPC still returned")
	}
}

func TestLegacyPlacementRetainsSelectedLevel(t *testing.T) {
	e := newContentsTestEditor(t)
	_, loc, room := e.oneRoomWorld(t)
	rec := postForm(t, e.handler, "/place/rooms/place", url.Values{"parent_id": {loc}, "entity_id": {room}, "x": {"0"}, "y": {"0"}, "z": {"-2"}}, "http://example.com", true)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	placements, err := e.roomPlacements.List()
	if err != nil || len(placements) != 1 || placements[0].Z != -2 {
		t.Fatal(placements, err)
	}
	body := drillCheck(t, e, "/place/rooms?parent_id="+loc+"&z=-2", `name="z" value="-2"`)
	if !strings.Contains(body, room) {
		t.Fatal("elevated room missing in legacy grid")
	}
}

func TestAPIWorldSwitchKeepsBoundHandler(t *testing.T) {
	root := t.TempDir()
	mainDir := t.TempDir()
	registry, err := newWorldRegistry(root, mainDir)
	if err != nil {
		t.Fatal(err)
	}
	original := registry.Current().handler
	if err = registry.Create("other"); err != nil {
		t.Fatal(err)
	}
	if err = registry.Load("other"); err != nil {
		t.Fatal(err)
	}
	if apiRequest(t, registry, "GET", "/api/v1/worlds/main", "", "").Code != 409 {
		t.Fatal("main request followed world switch")
	}
	if apiRequest(t, registry, "GET", "/api/v1/worlds/other", "", "").Code != 200 {
		t.Fatal("named world unavailable")
	}
	// An already-routed request finishes against the old world, without consulting
	// the registry's newly selected name or redirecting its write.
	rev := apiRequest(t, original, "GET", "/api/v1/worlds/main", "", "").Header().Get("ETag")
	w := apiRequest(t, original, "POST", "/api/v1/worlds/main/hubs", rev, `{"name":"Original world Hub"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	other := apiRequest(t, registry, "GET", "/api/v1/worlds/other", "", "")
	if strings.Contains(other.Body.String(), "Original world Hub") {
		t.Fatal("write crossed worlds")
	}
}
