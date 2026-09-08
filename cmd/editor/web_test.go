package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testEditorHandler(t *testing.T) (http.Handler, *worldItemStore) {
	t.Helper()
	dir := t.TempDir()
	store := newWorldItemStore(dir)
	return newEditorHandler(store, newHubStore(dir), newRoomStore(dir), newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir)), store
}

func testHubEditorHandler(t *testing.T) (http.Handler, *hubStore) {
	t.Helper()
	dir := t.TempDir()
	store := newHubStore(dir)
	return newEditorHandler(newWorldItemStore(dir), store, newRoomStore(dir), newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir)), store
}

func testRemainingEditorHandler(t *testing.T) (http.Handler, *roomStore, *npcStore, *terminalStore) {
	t.Helper()
	dir := t.TempDir()
	rooms := newRoomStore(dir)
	npcs := newNPCStore(dir)
	terminals := newTerminalStore(dir)
	return newEditorHandler(newWorldItemStore(dir), newHubStore(dir), rooms, npcs, terminals, newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir)), rooms, npcs, terminals
}

func testNetworkEditorHandler(t *testing.T) (http.Handler, *hostNetworkStore, *terminalStore, *networkAssignmentStore) {
	t.Helper()
	dir := t.TempDir()
	networks := newHostNetworkStore(dir)
	terminals := newTerminalStore(dir)
	assignments := newNetworkAssignmentStore(dir)
	return newEditorHandler(newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir), newNPCStore(dir), terminals, networks, assignments, newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir)), networks, terminals, assignments
}

func getRequest(t *testing.T, handler http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func postForm(t *testing.T, handler http.Handler, path string, values url.Values, origin string, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func validItemForm(name string) url.Values {
	return url.Values{
		"name":              {name},
		"kind":              {"takeable"},
		"short_description": {"short description"},
		"full_description":  {"full description"},
	}
}

func TestRootRedirectsToHubs(t *testing.T) {
	handler, _ := testEditorHandler(t)
	rec := getRequest(t, handler, "/", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("root status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/content/hubs" {
		t.Fatalf("root redirect = %q, want /content/hubs", got)
	}
}

func TestFullPageHasLibraryNavigationAndWorldItemWorkspace(t *testing.T) {
	handler, _ := testEditorHandler(t)
	rec := getRequest(t, handler, "/content/world-items", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<!doctype html>", "PAWS_IN_THE_SHELL", `href="/content/world-items"`,
		`href="/library"`, `aria-label="content details"`,
		`hx-target="#workspace"`, `aria-current="page"`, "New Item",
		"Item Name", "Takeable", "Fixed", "Scenery", "Short Description",
		"Full Description", "/static/htmx.min.js", "/static/editor.js",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("world-item page missing %q", want)
		}
	}
	if strings.Contains(body, "Saving is not implemented") || strings.Contains(body, "disabled title") {
		t.Fatal("persistent form still claims Save is unavailable")
	}
}

func TestLegacyCreateAndEditURLsRedirectToContent(t *testing.T) {
	handler, _ := testEditorHandler(t)
	for _, path := range []string{"/create/world-items", "/edit/world-items"} {
		rec := getRequest(t, handler, path, nil)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/content/world-items" {
			t.Fatalf("%s did not redirect to Content: %d %q", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}

func TestUndeclaredAndAssignmentScreensStayPlaceholders(t *testing.T) {
	handler, _ := testEditorHandler(t)
	for _, path := range []string{
		"/content/quests",
	} {
		t.Run(path, func(t *testing.T) {
			rec := getRequest(t, handler, path, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "(Placeholder)") {
				t.Fatal("undeclared screen did not expose its placeholder status")
			}
		})
	}
}

func TestRemainingCRUDPagesExposeDeclaredForms(t *testing.T) {
	handler, _, _, _ := testRemainingEditorHandler(t)
	for path, wants := range map[string][]string{
		"/content/locations": {"New Location", "Location Name", "Description", "Search by name"},
		"/content/rooms":     {"New Room", "Room Name", "Description", "Search by name"},
		"/content/npcs":      {"New NPC", "NPC Name", "Description", "Search by name"},
		"/content/terminals": {"New Terminal", "Machine Hostname", "Search by machine hostname"},
	} {
		t.Run(path, func(t *testing.T) {
			rec := getRequest(t, handler, path, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			for _, want := range wants {
				if !strings.Contains(rec.Body.String(), want) {
					t.Errorf("page missing %q", want)
				}
			}
			if strings.Contains(rec.Body.String(), "(Placeholder)") || strings.Contains(rec.Body.String(), "persistence is not implemented") {
				t.Fatal("implemented CRUD screen still reports placeholder status")
			}
		})
	}
}

func TestCreateWorldItemPersistsAndKeepsItSelected(t *testing.T) {
	handler, store := testEditorHandler(t)
	rec := postForm(t, handler, "/content/world-items", validItemForm("new item"), "http://example.com", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Trigger"); got != "contentSaved" {
		t.Fatalf("HX-Trigger = %q, want contentSaved", got)
	}
	if got := rec.Header().Get("HX-Push-Url"); !strings.HasPrefix(got, "/content/world-items/") {
		t.Fatalf("HX-Push-Url = %q, want selected item URL", got)
	}
	body := rec.Body.String()
	for _, want := range []string{"Saved new item.", "new item", "Edit"} {
		if !strings.Contains(body, want) {
			t.Errorf("saved workspace missing %q", want)
		}
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "new item" {
		t.Fatalf("created item was not persisted: %+v", items)
	}
}

func TestPlainHTMLCreateRedirectsToSavedItem(t *testing.T) {
	handler, _ := testEditorHandler(t)
	rec := postForm(t, handler, "/content/world-items", validItemForm("plain item"), "http://example.com", false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if location := rec.Header().Get("Location"); !strings.HasPrefix(location, "/content/world-items/") || !strings.HasSuffix(location, "?saved=1") {
		t.Fatalf("unexpected post-save redirect %q", location)
	}
}

func TestCreateValidationPreservesValuesAndDoesNotWrite(t *testing.T) {
	handler, store := testEditorHandler(t)
	values := url.Values{
		"name":              {""},
		"kind":              {"invalid"},
		"short_description": {""},
		"full_description":  {""},
	}
	rec := postForm(t, handler, "/content/world-items", values, "http://example.com", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Item Name is required.", "Choose Takeable, Fixed, or Scenery.",
		"Short Description is required.", "Full Description is required.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("validation response missing %q", want)
		}
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("invalid form persisted items: %+v", items)
	}
}

func TestSelectAndUpdateWorldItem(t *testing.T) {
	handler, store := testEditorHandler(t)
	created, err := store.Create(testWorldItemInput("old name"))
	if err != nil {
		t.Fatal(err)
	}
	rec := getRequest(t, handler, "/content/world-items/"+created.ID, map[string]string{"HX-Request": "true"})
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("selection did not return an editor fragment: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "old name") {
		t.Fatal("selected item values were not loaded")
	}

	values := validItemForm("new name")
	values.Set("kind", "scenery")
	rec = postForm(t, handler, "/content/world-items/"+created.ID, values, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Saved new name.") {
		t.Fatalf("update failed: %d %s", rec.Code, rec.Body.String())
	}
	updated, err := store.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || updated.Name != "new name" || updated.Kind != "scenery" {
		t.Fatalf("unexpected persisted update: %+v", updated)
	}
}

func TestDeleteWorldItemRequiresConfirmationUIAndPersists(t *testing.T) {
	handler, store := testEditorHandler(t)
	created, err := store.Create(testWorldItemInput("delete me"))
	if err != nil {
		t.Fatal(err)
	}
	selected := getRequest(t, handler, "/content/world-items/"+created.ID, nil)
	for _, want := range []string{
		">Delete</button>",
		`hx-confirm="Delete delete me? This cannot be undone from the editor."`,
		"/content/world-items/" + created.ID + "/delete",
	} {
		if !strings.Contains(selected.Body.String(), want) {
			t.Errorf("selected item is missing delete UI %q", want)
		}
	}

	rec := postForm(t, handler, "/content/world-items/"+created.ID+"/delete", url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("HX-Trigger") != "contentDeleted" || rec.Header().Get("HX-Push-Url") != "/content/world-items" {
		t.Fatalf("delete did not reset HTMX state: %+v", rec.Header())
	}
	if !strings.Contains(rec.Body.String(), "Deleted delete me.") || !strings.Contains(rec.Body.String(), "New World Item") {
		t.Fatalf("delete response did not return to create mode: %s", rec.Body.String())
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("deleted item remained in catalog: %+v", items)
	}
}

func TestUnknownOrCrossOriginDeleteDoesNotChangeCatalog(t *testing.T) {
	handler, store := testEditorHandler(t)
	created, err := store.Create(testWorldItemInput("keep me"))
	if err != nil {
		t.Fatal(err)
	}
	rec := postForm(t, handler, "/content/world-items/"+created.ID+"/delete", url.Values{}, "https://unrelated.example", true)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin delete status = %d, want 403", rec.Code)
	}
	if _, err := store.Get(created.ID); err != nil {
		t.Fatalf("cross-origin delete removed item: %v", err)
	}
	rec = postForm(t, handler, "/content/world-items/123e4567-e89b-42d3-a456-426614174000/delete", url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown delete status = %d, want 404", rec.Code)
	}
}

func TestWorldItemSearchFiltersNamesWithoutReplacingEditor(t *testing.T) {
	handler, store := testEditorHandler(t)
	for _, name := range []string{"Copper Mug", "Data Shard"} {
		if _, err := store.Create(testWorldItemInput(name)); err != nil {
			t.Fatal(err)
		}
	}
	rec := getRequest(t, handler, "/content/world-items/list?q=SHARD", map[string]string{"HX-Request": "true"})
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Data Shard") || strings.Contains(body, "Copper Mug") {
		t.Fatalf("unexpected search result: %d %s", rec.Code, body)
	}
	if strings.Contains(body, "id=\"item-editor\"") {
		t.Fatal("catalog search response replaced the item editor")
	}
}

func TestWorldItemWritesRequireSameOrigin(t *testing.T) {
	handler, store := testEditorHandler(t)
	for _, origin := range []string{"", "https://unrelated.example"} {
		rec := postForm(t, handler, "/content/world-items", validItemForm("blocked"), origin, true)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("origin %q status = %d, want 403", origin, rec.Code)
		}
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatal("cross-origin request changed content")
	}
}

func TestMalformedCatalogIsReportedAndNotReplaced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "world_items.json")
	broken := []byte(`{"version":99,"world_items":[]}`)
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	handler := newEditorHandler(newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir), newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir))
	rec := getRequest(t, handler, "/content/world-items", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "storage is unavailable") {
		t.Fatalf("malformed catalog error was not rendered: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "<button type=\"submit\" disabled") {
		t.Fatal("Save remained enabled while storage was unavailable")
	}
}

func TestWorldItemHTMLIsEscaped(t *testing.T) {
	handler, store := testEditorHandler(t)
	if _, err := store.Create(testWorldItemInput("<script>alert(1)</script>")); err != nil {
		t.Fatal(err)
	}
	rec := getRequest(t, handler, "/content/world-items", nil)
	body := rec.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") || !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("authored item name was not HTML escaped")
	}
}

func TestHubPageHasSharedCatalogAndEditor(t *testing.T) {
	handler, _ := testHubEditorHandler(t)
	rec := getRequest(t, handler, "/content/hubs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	for _, want := range []string{"New Hub", "Hub Name", `action="/content/hubs"`, `data-dirty-form`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("hub page missing %q", want)
		}
	}
	if strings.Contains(rec.Body.String(), "(Placeholder)") {
		t.Fatal("hub CRUD screen still renders as a placeholder")
	}
}

func TestCreateSelectUpdateAndSearchHub(t *testing.T) {
	handler, store := testHubEditorHandler(t)
	rec := postForm(t, handler, "/content/hubs", url.Values{"name": {"Central"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentSaved" {
		t.Fatalf("create failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	hubs, err := store.List()
	if err != nil || len(hubs) != 1 {
		t.Fatalf("created hub was not persisted: %+v, %v", hubs, err)
	}
	created := hubs[0]
	if got := rec.Header().Get("HX-Push-Url"); got != "/content/hubs/"+created.ID {
		t.Fatalf("HX-Push-Url = %q", got)
	}

	selected := getRequest(t, handler, "/content/hubs/"+created.ID, map[string]string{"HX-Request": "true"})
	if selected.Code != http.StatusOK || !strings.Contains(selected.Body.String(), "Central") || strings.Contains(selected.Body.String(), "<!doctype html>") {
		t.Fatalf("selection did not return the populated editor: %d %s", selected.Code, selected.Body.String())
	}

	rec = postForm(t, handler, "/content/hubs/"+created.ID, url.Values{"name": {"North Hub"}}, "http://example.com", true)
	updated, err := store.Get(created.ID)
	if rec.Code != http.StatusOK || err != nil || updated.ID != created.ID || updated.Name != "North Hub" {
		t.Fatalf("update failed: %d %+v %v %s", rec.Code, updated, err, rec.Body.String())
	}

	if _, err := store.Create("South Hub"); err != nil {
		t.Fatal(err)
	}
	rec = getRequest(t, handler, "/content/hubs/list?q=NORTH", map[string]string{"HX-Request": "true"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "North Hub") || strings.Contains(rec.Body.String(), "South Hub") {
		t.Fatalf("unexpected hub search result: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHubValidationPlainRedirectAndDelete(t *testing.T) {
	handler, store := testHubEditorHandler(t)
	rec := postForm(t, handler, "/content/hubs", url.Values{"name": {"   "}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Hub Name is required.") {
		t.Fatalf("blank name validation failed: %d %s", rec.Code, rec.Body.String())
	}
	if hubs, err := store.List(); err != nil || len(hubs) != 0 {
		t.Fatalf("invalid form changed catalog: %+v %v", hubs, err)
	}

	rec = postForm(t, handler, "/content/hubs", url.Values{"name": {"Plain Hub"}}, "http://example.com", false)
	if rec.Code != http.StatusSeeOther || !strings.HasSuffix(rec.Header().Get("Location"), "?saved=1") {
		t.Fatalf("plain create did not redirect: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	hubs, err := store.List()
	if err != nil || len(hubs) != 1 {
		t.Fatalf("plain create did not persist: %+v %v", hubs, err)
	}
	selected := getRequest(t, handler, "/content/hubs/"+hubs[0].ID+"?details=1", nil)
	for _, want := range []string{">Delete</button>", `hx-confirm="Delete Plain Hub? This cannot be undone from the editor."`} {
		if !strings.Contains(selected.Body.String(), want) {
			t.Errorf("selected hub missing delete UI %q", want)
		}
	}
	rec = postForm(t, handler, "/content/hubs/"+hubs[0].ID+"/delete", url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentDeleted" || rec.Header().Get("HX-Push-Url") != "/content/hubs" {
		t.Fatalf("delete failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	if remaining, err := store.List(); err != nil || len(remaining) != 0 {
		t.Fatalf("deleted hub remained: %+v %v", remaining, err)
	}
}

func TestHubWritesRequireSameOriginAndMalformedCatalogIsPreserved(t *testing.T) {
	handler, store := testHubEditorHandler(t)
	for _, origin := range []string{"", "https://unrelated.example"} {
		rec := postForm(t, handler, "/content/hubs", url.Values{"name": {"blocked"}}, origin, true)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("origin %q status = %d, want 403", origin, rec.Code)
		}
	}
	if hubs, err := store.List(); err != nil || len(hubs) != 0 {
		t.Fatalf("cross-origin write changed catalog: %+v %v", hubs, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "hubs.json")
	broken := []byte(`{"version":99,"hubs":[]}`)
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	handler = newEditorHandler(newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir), newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir))
	rec := getRequest(t, handler, "/content/hubs", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "storage is unavailable") || !strings.Contains(rec.Body.String(), `<button type="submit" disabled`) {
		t.Fatalf("malformed hub catalog was not safely reported: %d %s", rec.Code, rec.Body.String())
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(broken) {
		t.Fatalf("malformed catalog changed: %q %v", got, err)
	}
}

func TestRoomCRUDSearchValidationAndDelete(t *testing.T) {
	handler, rooms, _, _ := testRemainingEditorHandler(t)
	values := url.Values{"name": {"Studio"}, "description": {"A room description."}}
	rec := postForm(t, handler, "/content/rooms", values, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentSaved" {
		t.Fatalf("room create failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	created, err := rooms.List()
	if err != nil || len(created) != 1 {
		t.Fatalf("room not persisted: %+v %v", created, err)
	}
	id := created[0].ID
	selected := getRequest(t, handler, "/content/rooms/"+id, map[string]string{"HX-Request": "true"})
	if selected.Code != http.StatusOK || !strings.Contains(selected.Body.String(), "Studio") || strings.Contains(selected.Body.String(), "<!doctype html>") {
		t.Fatalf("room selection failed: %d %s", selected.Code, selected.Body.String())
	}
	rec = postForm(t, handler, "/content/rooms/"+id, url.Values{"name": {"Workshop"}, "description": {"Updated."}}, "http://example.com", true)
	updated, err := rooms.Get(id)
	if rec.Code != http.StatusOK || err != nil || updated.ID != id || updated.Name != "Workshop" {
		t.Fatalf("room update failed: %d %+v %v", rec.Code, updated, err)
	}
	if _, err := rooms.Create(describedInput{Name: "Atrium", Description: "Other."}); err != nil {
		t.Fatal(err)
	}
	rec = getRequest(t, handler, "/content/rooms/list?q=WORK", map[string]string{"HX-Request": "true"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Workshop") || strings.Contains(rec.Body.String(), "Atrium") {
		t.Fatalf("room search failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = postForm(t, handler, "/content/rooms", url.Values{"name": {""}, "description": {""}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Room Name is required.") || !strings.Contains(rec.Body.String(), "Description is required.") {
		t.Fatalf("room validation missing: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/rooms/"+id+"/delete", url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentDeleted" || rec.Header().Get("HX-Push-Url") != "/content/rooms" {
		t.Fatalf("room delete failed: %d %v", rec.Code, rec.Header())
	}
}

func TestNPCCreatePlainRedirectEscapingAndDeleteUI(t *testing.T) {
	handler, _, npcs, _ := testRemainingEditorHandler(t)
	values := url.Values{"name": {"<script>npc</script>"}, "description": {"NPC description."}}
	rec := postForm(t, handler, "/content/npcs", values, "http://example.com", false)
	if rec.Code != http.StatusSeeOther || !strings.HasSuffix(rec.Header().Get("Location"), "?saved=1") {
		t.Fatalf("NPC create redirect failed: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	created, err := npcs.List()
	if err != nil || len(created) != 1 {
		t.Fatalf("NPC not persisted: %+v %v", created, err)
	}
	rec = getRequest(t, handler, "/content/npcs/"+created[0].ID, nil)
	body := rec.Body.String()
	if strings.Contains(body, "<script>npc</script>") || !strings.Contains(body, "&lt;script&gt;npc&lt;/script&gt;") {
		t.Fatal("NPC name was not HTML escaped")
	}
	if !strings.Contains(body, `hx-confirm="Delete &lt;script&gt;npc&lt;/script&gt;? This cannot be undone from the editor."`) {
		t.Fatal("selected NPC is missing confirmed delete UI")
	}
}

func TestTerminalCRUDValidationSearchAndDelete(t *testing.T) {
	handler, _, _, terminals := testRemainingEditorHandler(t)
	rec := postForm(t, handler, "/content/terminals", url.Values{"host_name": {"deck"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentSaved" {
		t.Fatalf("terminal create failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	created, err := terminals.List()
	if err != nil || len(created) != 1 {
		t.Fatalf("terminal not persisted: %+v %v", created, err)
	}
	id := created[0].ID
	rec = postForm(t, handler, "/content/terminals/"+id, url.Values{"host_name": {"undernet.relay"}}, "http://example.com", true)
	updated, err := terminals.Get(id)
	if rec.Code != http.StatusOK || err != nil || updated.ID != id || updated.HostName != "undernet.relay" {
		t.Fatalf("terminal update failed: %d %+v %v", rec.Code, updated, err)
	}
	if _, err := terminals.Create(terminalInput{HostName: "deck"}); err != nil {
		t.Fatal(err)
	}
	rec = getRequest(t, handler, "/content/terminals/list?q=UNDERNET", map[string]string{"HX-Request": "true"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "undernet.relay") || strings.Contains(rec.Body.String(), ">deck<") {
		t.Fatalf("terminal search failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = postForm(t, handler, "/content/terminals", url.Values{"host_name": {"Bad Host"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Use lowercase letters") {
		t.Fatalf("terminal format validation missing: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/terminals", url.Values{"host_name": {"deck"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Saved deck.") {
		t.Fatalf("unassigned duplicate hostname was rejected: %d %s", rec.Code, rec.Body.String())
	}
	rec = postForm(t, handler, "/content/terminals/"+id+"/delete", url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Push-Url") != "/content/terminals" {
		t.Fatalf("terminal delete failed: %d %v", rec.Code, rec.Header())
	}
}

func TestRemainingCRUDWritesRequireSameOrigin(t *testing.T) {
	handler, rooms, npcs, terminals := testRemainingEditorHandler(t)
	requests := []struct {
		path   string
		values url.Values
	}{
		{"/content/rooms", url.Values{"name": {"Room"}, "description": {"Description"}}},
		{"/content/npcs", url.Values{"name": {"NPC"}, "description": {"Description"}}},
		{"/content/terminals", url.Values{"host_name": {"deck"}}},
	}
	for _, request := range requests {
		rec := postForm(t, handler, request.path, request.values, "https://unrelated.example", true)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s status = %d, want 403", request.path, rec.Code)
		}
	}
	if got, _ := rooms.List(); len(got) != 0 {
		t.Fatal("cross-origin room write succeeded")
	}
	if got, _ := npcs.List(); len(got) != 0 {
		t.Fatal("cross-origin NPC write succeeded")
	}
	if got, _ := terminals.List(); len(got) != 0 {
		t.Fatal("cross-origin terminal write succeeded")
	}
}

func TestRemainingCRUDMalformedStorageDisablesSave(t *testing.T) {
	for _, entity := range []string{"rooms", "npcs", "terminals"} {
		t.Run(entity, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, entity+".json")
			broken := []byte(`{"version":99,"` + entity + `":[]}`)
			if err := os.WriteFile(path, broken, 0o644); err != nil {
				t.Fatal(err)
			}
			handler := newEditorHandler(newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir), newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir))
			rec := getRequest(t, handler, "/content/"+entity, nil)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "storage is unavailable") || !strings.Contains(rec.Body.String(), `<button type="submit" disabled`) {
				t.Fatalf("malformed %s catalog was not safely reported: %d %s", entity, rec.Code, rec.Body.String())
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != string(broken) {
				t.Fatalf("malformed %s catalog changed: %v", entity, err)
			}
		})
	}
}

func TestHostNetworkCRUDAndNestedTerminalCreation(t *testing.T) {
	handler, networks, terminals, assignments := testNetworkEditorHandler(t)
	page := getRequest(t, handler, "/content/corporations", nil)
	for _, want := range []string{"Corporations", "New Corporation", "Corporation Name"} {
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), want) {
			t.Fatalf("host-network page missing %q: %d %s", want, page.Code, page.Body.String())
		}
	}
	rec := postForm(t, handler, "/content/corporations", url.Values{"name": {"Office Network"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentSaved" {
		t.Fatalf("network create failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	createdNetworks, err := networks.List()
	if err != nil || len(createdNetworks) != 1 {
		t.Fatalf("network not persisted: %+v %v", createdNetworks, err)
	}
	networkID := createdNetworks[0].ID
	rec = getRequest(t, handler, "/content/corporations/"+networkID+"/terminals", map[string]string{"HX-Request": "true"})
	for _, want := range []string{"Assigned Terminals", "Create Terminal in Office Network", "Create and Assign"} {
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("network terminal view missing %q: %d %s", want, rec.Code, rec.Body.String())
		}
	}
	rec = postForm(t, handler, "/content/corporations/"+networkID+"/terminals", url.Values{"host_name": {"workstation-01"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Created and assigned workstation-01.") {
		t.Fatalf("nested terminal create failed: %d %s", rec.Code, rec.Body.String())
	}
	createdTerminals, err := terminals.List()
	if err != nil || len(createdTerminals) != 1 {
		t.Fatalf("terminal not created: %+v %v", createdTerminals, err)
	}
	assignment, err := assignments.GetByTerminal(createdTerminals[0].ID)
	if err != nil || assignment.HostNetworkID != networkID {
		t.Fatalf("terminal not assigned: %+v %v", assignment, err)
	}
}

func TestNetworkAssignmentHostnameScopeUnassignAndDeleteGuards(t *testing.T) {
	handler, networks, terminals, assignments := testNetworkEditorHandler(t)
	firstNetwork, err := networks.Create("First")
	if err != nil {
		t.Fatal(err)
	}
	secondNetwork, err := networks.Create("Second")
	if err != nil {
		t.Fatal(err)
	}
	first, err := terminals.Create(terminalInput{HostName: "server"})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := terminals.Create(terminalInput{HostName: "server"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := assignments.Assign(first.ID, firstNetwork.ID); err != nil {
		t.Fatal(err)
	}

	rec := postForm(t, handler, "/content/corporations/"+firstNetwork.ID+"/assign", url.Values{"terminal_id": {duplicate.ID}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "hostname is already used") {
		t.Fatalf("same-network duplicate was not rejected: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := assignments.GetByTerminal(duplicate.ID); !errors.Is(err, errNetworkAssignmentNotFound) {
		t.Fatalf("rejected duplicate was assigned: %v", err)
	}
	rec = postForm(t, handler, "/content/corporations/"+secondNetwork.ID+"/assign", url.Values{"terminal_id": {duplicate.ID}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Assigned server.") {
		t.Fatalf("cross-network duplicate was rejected: %d %s", rec.Code, rec.Body.String())
	}
	other, err := terminals.Create(terminalInput{HostName: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := assignments.Assign(other.ID, secondNetwork.ID); err != nil {
		t.Fatal(err)
	}
	rec = postForm(t, handler, "/content/terminals/"+other.ID, url.Values{"host_name": {"server"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Machine Hostname must be unique within its assigned network.") {
		t.Fatalf("assigned terminal rename collision was not rejected: %s", rec.Body.String())
	}
	unchanged, err := terminals.Get(other.ID)
	if err != nil || unchanged.HostName != "other" {
		t.Fatalf("rejected rename changed terminal: %+v %v", unchanged, err)
	}

	rec = postForm(t, handler, "/content/terminals/"+first.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unassign this terminal") {
		t.Fatalf("assigned terminal delete was not blocked: %s", rec.Body.String())
	}
	if rec.Header().Get("HX-Trigger") != "" || rec.Header().Get("HX-Push-Url") != "" {
		t.Fatalf("blocked terminal delete reported success: %v", rec.Header())
	}
	rec = postForm(t, handler, "/content/corporations/"+firstNetwork.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Unassign every terminal") {
		t.Fatalf("nonempty network delete was not blocked: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/corporations/"+firstNetwork.ID+"/unassign/"+first.ID, url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Unassigned server.") {
		t.Fatalf("unassign failed: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := assignments.GetByTerminal(first.ID); !errors.Is(err, errNetworkAssignmentNotFound) {
		t.Fatalf("assignment remained after unassign: %v", err)
	}
	rec = postForm(t, handler, "/content/corporations/"+firstNetwork.ID+"/delete", url.Values{}, "http://example.com", true)
	if rec.Header().Get("HX-Trigger") != "contentDeleted" {
		t.Fatalf("empty network delete failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestNetworkRelationshipValidationRejectsDanglingReferences(t *testing.T) {
	handler, networks, _, assignments := testNetworkEditorHandler(t)
	network, err := networks.Create("Network")
	if err != nil {
		t.Fatal(err)
	}
	missingTerminal := "123e4567-e89b-42d3-a456-426614174999"
	if _, err := assignments.Assign(missingTerminal, network.ID); err != nil {
		t.Fatal(err)
	}
	rec := getRequest(t, handler, "/content/corporations/"+network.ID+"/terminals", nil)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "is assigned a Terminal that no longer exists") || !strings.Contains(body, "storage is unavailable") {
		t.Fatalf("dangling relationship was not reported: %d %s", rec.Code, body)
	}
	if !strings.Contains(body, "Network") || !strings.Contains(body, missingTerminal) {
		t.Fatalf("report should name the surviving corporation and the dangling UUID: %s", body)
	}
	rec = postForm(t, handler, "/content/corporations", url.Values{"name": {"Blocked"}}, "http://example.com", true)
	if rec.Code != http.StatusConflict {
		t.Fatalf("write over invalid relationships status = %d, want 409", rec.Code)
	}
}

func TestHostNetworkWritesRequireSameOriginAndMalformedStorageIsPreserved(t *testing.T) {
	handler, networks, _, _ := testNetworkEditorHandler(t)
	rec := postForm(t, handler, "/content/corporations", url.Values{"name": {"Blocked"}}, "https://unrelated.example", true)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin network write status = %d, want 403", rec.Code)
	}
	if got, err := networks.List(); err != nil || len(got) != 0 {
		t.Fatalf("cross-origin write changed networks: %+v %v", got, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "host_networks.json")
	broken := []byte(`{"version":99,"host_networks":[]}`)
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	handler = newEditorHandler(newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir), newNPCStore(dir), newTerminalStore(dir), newHostNetworkStore(dir), newNetworkAssignmentStore(dir), newUserStore(dir), newTerminalAccessStore(dir), newRoomPlacementStore(dir))
	rec = getRequest(t, handler, "/content/corporations", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "storage is unavailable") || !strings.Contains(rec.Body.String(), `<button type="submit" disabled`) {
		t.Fatalf("malformed host-network catalog was not safely reported: %d %s", rec.Code, rec.Body.String())
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(broken) {
		t.Fatalf("malformed host-network catalog changed: %v", err)
	}
}

func TestStaticAssetsAreEmbedded(t *testing.T) {
	handler, _ := testEditorHandler(t)
	for path, want := range map[string]string{
		"/static/editor.css": ".crud-layout",
		"/static/editor.js":  "Discard unsaved content changes?",
	} {
		rec := getRequest(t, handler, path, nil)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("asset %s missing %q: %d", path, want, rec.Code)
		}
	}
}

func TestUnknownWorldItemAndUnsupportedMethods(t *testing.T) {
	handler, _ := testEditorHandler(t)
	rec := getRequest(t, handler, "/content/world-items/123e4567-e89b-42d3-a456-426614174000", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown item status = %d, want 404", rec.Code)
	}
	req := httptest.NewRequest(http.MethodDelete, "/content/world-items", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE status = %d, want 405", rec.Code)
	}
}
