package main

import (
	"encoding/json"
	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestQuestAPIAndForms(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, room := e.oneRoomWorld(t)
	root := "/api/v1/worlds/main"
	route := root + "/quests"
	revision := func() string { return apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag") }
	q := content.Quest{Name: "Visit", Enabled: true, AutoStart: true, Steps: []content.QuestStep{{Text: "Visit room", Kind: "visit", Target: content.CellRef{Kind: "room", ID: room}}}}
	raw, _ := json.Marshal(q)
	rev := revision()
	if apiRequest(t, e.handler, "POST", route, "", string(raw)).Code != 428 {
		t.Fatal("missing revision accepted")
	}
	created := apiRequest(t, e.handler, "POST", route, rev, string(raw))
	if created.Code != 201 {
		t.Fatal(created.Body.String())
	}
	if created.Header().Get("ETag") == rev {
		t.Fatal("quest did not change world revision")
	}
	json.Unmarshal(created.Body.Bytes(), &q)
	if q.ID == "" || q.Steps[0].ID == "" {
		t.Fatal("missing stable IDs")
	}
	if apiRequest(t, e.handler, "POST", route, rev, string(raw)).Code != 409 {
		t.Fatal("stale revision accepted")
	}
	page := getRequest(t, e.handler, "/content/quests/"+q.ID, nil)
	if !strings.Contains(page.Body.String(), "Preview conditions") || strings.Contains(page.Body.String(), "editor template error") {
		t.Fatal(page.Body.String())
	}
	related := getRequest(t, e.handler, "/content/rooms/"+room, nil)
	if !strings.Contains(related.Body.String(), "Related quests") {
		t.Fatal("missing related quest")
	}
	preview := postForm(t, e.handler, "/content/quests/"+q.ID, url.Values{"revision": {revision()}, "action": {"preview"}, "satisfied": {q.Steps[0].ID}}, "http://example.com", false)
	if !strings.Contains(preview.Body.String(), "Visit — completed") {
		t.Fatal(preview.Body.String())
	}
	if revision() != created.Header().Get("ETag") {
		t.Fatal("preview changed authored content")
	}
	oldID := q.Steps[0].ID
	q.Name = "Edited"
	raw, _ = json.Marshal(q)
	edited := apiRequest(t, e.handler, "PUT", route+"/"+q.ID, revision(), string(raw))
	if edited.Code != 200 || !strings.Contains(edited.Body.String(), oldID) {
		t.Fatal(edited.Body.String())
	}
	if apiRequest(t, e.handler, "DELETE", root+"/entities/room/"+room, revision(), `{}`).Code != 422 {
		t.Fatal("reference deletion not blocked")
	}
	bad := q
	bad.Steps = append([]content.QuestStep(nil), q.Steps...)
	bad.Steps[0].Target.ID = "missing"
	raw, _ = json.Marshal(bad)
	rev = revision()
	if apiRequest(t, e.handler, "PUT", route+"/"+q.ID, rev, string(raw)).Code != 422 || revision() != rev {
		t.Fatal("bad target changed data")
	}
	bad.Enabled = false
	raw, _ = json.Marshal(bad)
	if apiRequest(t, e.handler, "PUT", route+"/"+q.ID, revision(), string(raw)).Code != 200 {
		t.Fatal("incomplete draft rejected")
	}
	request := httptest.NewRequest("DELETE", route+"/"+q.ID, nil)
	request.Header.Set("If-Match", revision())
	request.Header.Set("Origin", "https://elsewhere.example")
	response := httptest.NewRecorder()
	e.handler.ServeHTTP(response, request)
	if response.Code != 403 {
		t.Fatal("cross origin accepted")
	}
	if apiRequest(t, e.handler, "DELETE", route+"/"+q.ID, revision(), "").Code != 200 {
		t.Fatal("delete failed")
	}
	if apiRequest(t, e.handler, "GET", route+"/"+q.ID, "", "").Code != 404 {
		t.Fatal("quest not deleted")
	}
	if apiRequest(t, e.handler, "GET", "/api/v1/worlds/other/quests", "", "").Code != 409 {
		t.Fatal("world isolation failed")
	}
}
func TestQuestStepFormsAndStaleWrites(t *testing.T) {
	e := newContentsTestEditor(t)
	_, _, room := e.oneRoomWorld(t)
	root := "/api/v1/worlds/main"
	revision := func() string { return apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag") }
	created := postForm(t, e.handler, "/content/quests", url.Values{"revision": {revision()}, "action": {"save"}, "name": {"Test"}, "auto_start": {"on"}}, "http://example.com", false)
	path := created.Header().Get("Location")
	if path == "" {
		t.Fatal(created.Body.String())
	}
	for _, txt := range []string{"First", "Second"} {
		response := postForm(t, e.handler, path, url.Values{"revision": {revision()}, "action": {"save-step"}, "text": {txt}, "kind": {"visit"}, "target": {"room:" + room}}, "http://example.com", false)
		if response.Code != 303 {
			t.Fatal(response.Body.String())
		}
	}
	get := apiRequest(t, e.handler, "GET", root+"/quests/"+strings.TrimPrefix(path, "/content/quests/"), "", "")
	var q content.Quest
	json.Unmarshal(get.Body.Bytes(), &q)
	response := postForm(t, e.handler, path, url.Values{"revision": {revision()}, "action": {"up"}, "step_id": {q.Steps[1].ID}}, "http://example.com", false)
	if response.Code != 303 {
		t.Fatal(response.Body.String())
	}
	get = apiRequest(t, e.handler, "GET", root+"/quests/"+q.ID, "", "")
	json.Unmarshal(get.Body.Bytes(), &q)
	if q.Steps[0].Text != "Second" {
		t.Fatal("reordering failed")
	}
	rev := revision()
	response = postForm(t, e.handler, path, url.Values{"revision": {"stale"}, "action": {"delete"}}, "http://example.com", false)
	if !strings.Contains(response.Body.String(), "World changed") || revision() != rev {
		t.Fatal("stale form mutated")
	}
}

func TestQuestCarryReferenceProtection(t *testing.T) {
	e := newContentsTestEditor(t)
	e.oneRoomWorld(t)
	root := "/api/v1/worlds/main"
	revision := func() string { return apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag") }
	item := apiRequest(t, e.handler, "POST", root+"/entities/world_item", revision(), `{"name":"Test cup","kind":"takeable","short_description":"(Placeholder)","full_description":"(Placeholder)"}`)
	if item.Code != 200 {
		t.Fatal(item.Body.String())
	}
	list := apiRequest(t, e.handler, "GET", root+"/entities/world_item", "", "")
	var items []content.WorldItem
	json.Unmarshal(list.Body.Bytes(), &items)
	id := ""
	for _, v := range items {
		if v.Name == "Test cup" {
			id = v.ID
		}
	}
	if id == "" {
		t.Fatal("item missing")
	}
	q := content.Quest{Name: "Carry", Enabled: true, AutoStart: true, Steps: []content.QuestStep{{Text: "Carry cup", Kind: "carry", Target: content.CellRef{Kind: "world_item", ID: id}}}}
	raw, _ := json.Marshal(q)
	r := apiRequest(t, e.handler, "POST", root+"/quests", revision(), string(raw))
	if r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	r = apiRequest(t, e.handler, "PUT", root+"/entities/world_item/"+id, revision(), `{"name":"Test cup","kind":"fixed","short_description":"(Placeholder)","full_description":"(Placeholder)"}`)
	if r.Code != 422 || !strings.Contains(r.Body.String(), "remain takeable") {
		t.Fatal(r.Body.String())
	}
	r = apiRequest(t, e.handler, "DELETE", root+"/entities/world_item/"+id, revision(), `{}`)
	if r.Code != 422 || !strings.Contains(r.Body.String(), "Referenced by quest") {
		t.Fatal(r.Body.String())
	}
	r = postForm(t, e.handler, "/content/world-items/"+id+"/delete", url.Values{}, "http://example.com", false)
	if !strings.Contains(r.Body.String(), "Referenced by quest") {
		t.Fatal("form deleted target", r.Body.String())
	}
}

func TestQuestValidationFormsRemainEditable(t *testing.T) {
	e := newContentsTestEditor(t)
	e.oneRoomWorld(t)
	root := "/api/v1/worlds/main"
	revision := func() string { return apiRequest(t, e.handler, "GET", root, "", "").Header().Get("ETag") }
	badDelete := postForm(t, e.handler, "/content/quests", url.Values{"revision": {revision()}, "action": {"delete"}}, "http://example.com", false)
	if badDelete.Code != 200 || !strings.Contains(badDelete.Body.String(), "file does not exist") {
		t.Fatal("invalid collection deletion not rejected", badDelete.Body.String())
	}
	r := postForm(t, e.handler, "/content/quests", url.Values{"revision": {revision()}, "action": {"save"}, "name": {"Incomplete"}, "enabled": {"on"}}, "http://example.com", false)
	if !strings.Contains(r.Body.String(), `action="/content/quests/"`) || !strings.Contains(r.Body.String(), "At least one objective") {
		t.Fatal("failed creation must keep collection URL", r.Body.String())
	}
	r = postForm(t, e.handler, "/content/quests/", url.Values{"revision": {revision()}, "action": {"save"}, "name": {"Incomplete"}}, "http://example.com", false)
	if r.Code != 303 {
		t.Fatal("could not recover as draft", r.Body.String())
	}
}
