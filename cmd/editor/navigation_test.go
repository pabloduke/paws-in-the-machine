package main

import (
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func selectionGet(h http.Handler, target string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "http://example.com"+target, nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestNavigationRequiresParentsAndKeepsSelection(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, room := e.oneRoomWorld(t)
	for path, message := range map[string]string{"/navigate/location": "Hub required.", "/navigate/room": "Location required.", "/navigate/floor": "Location required."} {
		w := selectionGet(e.handler, path, nil)
		if !strings.Contains(w.Body.String(), message) {
			t.Fatal(path, w.Body.String())
		}
	}
	w := selectionGet(e.handler, entityURL("hub", hub), nil)
	cookie := w.Result().Cookies()[0]
	w = selectionGet(e.handler, "/navigate/location", cookie)
	if w.Header().Get("Location") != entityURL("hub", hub) {
		t.Fatal("Hub not retained")
	}
	w = selectionGet(e.handler, "/navigate/room", cookie)
	if !strings.Contains(w.Body.String(), "Location required.") {
		t.Fatal("Hub substituted for Location")
	}
	w = selectionGet(e.handler, entityURL("location", loc)+"?z=-2", cookie)
	cookie = w.Result().Cookies()[0]
	w = selectionGet(e.handler, "/navigate/floor", cookie)
	if w.Header().Get("Location") != entityURL("location", loc)+"?z=-2#children-heading" {
		t.Fatal("floor lost", w.Header())
	}
	w = selectionGet(e.handler, entityURL("room", room), cookie)
	cookie = w.Result().Cookies()[0]
	w = selectionGet(e.handler, "/content/hubs", cookie)
	if !strings.Contains(w.Body.String(), "Selected Hub") || !strings.Contains(w.Body.String(), `name="arrival_id"`) {
		t.Fatal("Hub controls missing")
	}
	other, _ := e.hubs.Create("Other Hub")
	w = selectionGet(e.handler, entityURL("hub", other.ID), cookie)
	cookie = w.Result().Cookies()[0]
	w = selectionGet(e.handler, "/navigate/room", cookie)
	if !strings.Contains(w.Body.String(), "Location required.") {
		t.Fatal("changing Hub kept stale Location")
	}
	// A second browser never inherits the first browser's selected parent.
	w = selectionGet(e.handler, "/navigate/location", nil)
	if !strings.Contains(w.Body.String(), "Hub required.") {
		t.Fatal("shared selection leaked")
	}
}
func TestWorldSwitchClearsBrowserSelection(t *testing.T) {
	registry, _, mainDir := newWorldsTestRegistry(t)
	hub, _ := newHubStore(mainDir).Create("Test Hub")
	w := selectionGet(registry, entityURL("hub", hub.ID), nil)
	cookie := w.Result().Cookies()[0]
	if err := registry.Create("other"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Load("other"); err != nil {
		t.Fatal(err)
	}
	w = selectionGet(registry, "/navigate/location", cookie)
	if !strings.Contains(w.Body.String(), "Hub required.") {
		t.Fatal("selection survived a world switch")
	}
}
func TestWorldRenameAndRecoverableDeletion(t *testing.T) {
	registry, root, _ := newWorldsTestRegistry(t)
	if err := registry.Create("draft"); err != nil {
		t.Fatal(err)
	}
	hub, _ := newHubStore(filepath.Join(root, "draft")).Create("Preserved Hub")
	if err := registry.manageInactive("draft", "renamed", false); err != nil {
		t.Fatal(err)
	}
	if _, err := newHubStore(filepath.Join(root, "renamed")).Get(hub.ID); err != nil {
		t.Fatal(err)
	}
	if err := registry.Load("renamed"); err != nil {
		t.Fatal(err)
	}
	if err := registry.manageInactive("renamed", "", true); err == nil {
		t.Fatal("deleted loaded world")
	}
	if err := registry.Load(mainWorld); err != nil {
		t.Fatal(err)
	}
	if err := registry.manageInactive("renamed", "", true); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".deleted-worlds", "renamed-*", "hubs.json"))
	if len(matches) != 1 {
		t.Fatal("archive missing")
	}
	worlds, err := registry.List()
	if err != nil || len(worlds) != 1 {
		t.Fatal(worlds, err)
	}
	if err := registry.manageInactive(mainWorld, "", true); err == nil {
		t.Fatal("main deleted")
	}
}

func TestWorkflowFormsAreNotNested(t *testing.T) {
	e := newContentsTestEditor(t)
	hub, loc, room := e.oneRoomWorld(t)
	for _, target := range []string{entityURL("hub", hub), entityURL("location", loc) + "?x=0&y=1", entityURL("room", room) + "?new=world_item"} {
		body := selectionGet(e.handler, target, nil).Body.String()
		tokens := html.NewTokenizer(strings.NewReader(body))
		inside := false
		for {
			typ := tokens.Next()
			if typ == html.ErrorToken {
				if tokens.Err() != io.EOF {
					t.Fatal(tokens.Err())
				}
				break
			}
			token := tokens.Token()
			if token.Data != "form" {
				continue
			}
			if typ == html.StartTagToken {
				if inside {
					t.Fatalf("nested form in %s", target)
				}
				inside = true
			}
			if typ == html.EndTagToken {
				if !inside {
					t.Fatalf("unmatched form closure in %s", target)
				}
				inside = false
			}
		}
		if inside {
			t.Fatal("unclosed form", target)
		}
	}
}
func TestCatalogAssignmentPanelAndStaleGuard(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, room := e.oneRoomWorld(t)
	npc, err := e.npcs.Create(describedInput{Name: "Test NPC", Description: "(Placeholder)"})
	if err != nil {
		t.Fatal(err)
	}
	body := selectionGet(e.handler, "/content/npcs/"+npc.ID, nil).Body.String()
	if !strings.Contains(body, `action="/editor-assignment"`) {
		t.Fatal("NPC assignment panel missing")
	}
	values := url.Values{"kind": {"npc"}, "id": {npc.ID}, "current": {""}, "container": {"room:" + room}}
	w := postForm(t, e.handler, "/editor-assignment", values, "http://example.com", false)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Body.String())
	}
	w = postForm(t, e.handler, "/editor-assignment", values, "http://example.com", true)
	if !strings.Contains(w.Body.String(), "Assignment changed") {
		t.Fatal("stale assignment accepted")
	}
	values.Set("current", "room:"+room)
	values.Set("container", "")
	w = postForm(t, e.handler, "/editor-assignment", values, "http://example.com", false)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Body.String())
	}
	_, held, err := e.contents.ContainerOf("npc", npc.ID)
	if err != nil || held {
		t.Fatal("entity remained assigned", err)
	}
}
