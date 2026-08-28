package main

import (
	"net/http"
	"strings"
	"testing"
)

// Every tab is an ordinary link enhanced by HTMX with
// hx-target="#workspace". A handler that answers such a GET with only
// its inner body strips the navigation out of the page and leaves the
// user with no way forward but the browser's Back button, so this pins
// the whole set at once.
func TestEveryTabURLReturnsAWholeWorkspaceOverHTMX(t *testing.T) {
	e := newContentsTestEditor(t)
	e.oneRoomWorld(t)
	for _, path := range []string{
		"/overview",
		"/content/world-items", "/content/locations", "/content/rooms", "/content/hubs",
		"/content/npcs", "/content/corporations", "/content/terminals", "/content/users",
		"/content/quests",
		"/place/locations", "/place/rooms",
		"/place/world-items", "/place/npcs", "/place/terminals",
	} {
		rec := getRequest(t, e.handler, path, map[string]string{"HX-Request": "true"})
		if rec.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", path, rec.Code)
			continue
		}
		body := rec.Body.String()
		for _, want := range []string{`class="header-tabs"`, `class="detail-tabs"`, `aria-current="page"`} {
			if !strings.Contains(body, want) {
				t.Errorf("HTMX GET %s dropped %q, so the navigation would disappear", path, want)
			}
		}
		if strings.Contains(body, "<!doctype html>") {
			t.Errorf("HTMX GET %s returned a whole document into #workspace", path)
		}
	}
}
