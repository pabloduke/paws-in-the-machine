package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAuthEditorHandler(t *testing.T) (http.Handler, string, *terminalStore, *userStore, *terminalAccessStore) {
	t.Helper()
	dir := t.TempDir()
	terminals := newTerminalStore(dir)
	users := newUserStore(dir)
	access := newTerminalAccessStore(dir)
	handler := newEditorHandler(newWorldItemStore(dir), newHubStore(dir), newRoomStore(dir), newNPCStore(dir), terminals, newHostNetworkStore(dir), newNetworkAssignmentStore(dir), users, access, newRoomPlacementStore(dir))
	return handler, dir, terminals, users, access
}

func TestUserCRUDAndValidation(t *testing.T) {
	handler, _, _, users, _ := testAuthEditorHandler(t)
	page := getRequest(t, handler, "/content/users", nil)
	for _, want := range []string{"New User", "Username", "Fictional Password", "Never enter a real password", "Search by username"} {
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), want) {
			t.Fatalf("users page missing %q: %d %s", want, page.Code, page.Body.String())
		}
	}
	rec := postForm(t, handler, "/content/users", url.Values{"username": {"operator"}, "password": {"fictional <password>"}}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != "contentSaved" {
		t.Fatalf("user create failed: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	created, err := users.List()
	if err != nil || len(created) != 1 {
		t.Fatalf("users = %+v, %v", created, err)
	}
	if strings.Contains(rec.Body.String(), "fictional <password>") || !strings.Contains(rec.Body.String(), "fictional &lt;password&gt;") {
		t.Fatal("fictional password was not HTML escaped")
	}
	rec = getRequest(t, handler, "/content/users/list?q=OPER", map[string]string{"HX-Request": "true"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "operator") {
		t.Fatalf("user search failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = postForm(t, handler, "/content/users", url.Values{"username": {"Bad User"}, "password": {" "}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Use lowercase letters") || !strings.Contains(rec.Body.String(), "Fictional Password is required") {
		t.Fatalf("user validation missing: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/users/"+created[0].ID+"/delete", url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Push-Url") != userBasePath {
		t.Fatalf("user delete failed: %d %v", rec.Code, rec.Header())
	}
}

func TestTerminalAccessGrantRevokeAndDeleteGuards(t *testing.T) {
	handler, _, terminals, users, access := testAuthEditorHandler(t)
	terminal, err := terminals.Create(terminalInput{HostName: "workstation"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := users.Create(userInput{Username: "operator", Password: "fictional-one"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := users.Create(userInput{Username: "second", Password: "fictional-two"})
	if err != nil {
		t.Fatal(err)
	}
	rec := getRequest(t, handler, "/content/terminals/"+terminal.ID+"/access", nil)
	for _, want := range []string{"Users with Access", "Grant Access", "operator", "second"} {
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("access page missing %q: %d %s", want, rec.Code, rec.Body.String())
		}
	}
	if strings.Contains(rec.Body.String(), first.Password) || strings.Contains(rec.Body.String(), second.Password) {
		t.Fatal("access screen exposed fictional passwords")
	}
	for _, user := range []gameUserForTest{{first.ID, first.Username}, {second.ID, second.Username}} {
		rec = postForm(t, handler, "/content/terminals/"+terminal.ID+"/grant", url.Values{"user_id": {user.id}}, "http://example.com", true)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Granted access to "+user.username) {
			t.Fatalf("grant failed for %s: %d %s", user.username, rec.Code, rec.Body.String())
		}
	}
	grants, err := access.List()
	if err != nil || len(grants) != 2 {
		t.Fatalf("grants = %+v, %v", grants, err)
	}
	rec = postForm(t, handler, "/content/terminals/"+terminal.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Revoke every user") {
		t.Fatalf("terminal delete was not guarded: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/users/"+first.ID+"/delete", url.Values{}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "Remove this user&#39;s terminal access") {
		t.Fatalf("user delete was not guarded: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/terminals/"+terminal.ID+"/revoke/"+first.ID, url.Values{}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Terminal access revoked") {
		t.Fatalf("revoke failed: %d %s", rec.Code, rec.Body.String())
	}
}

type gameUserForTest struct{ id, username string }

func TestTerminalAccessUsernameUniquenessIsScopedPerTerminal(t *testing.T) {
	handler, _, terminals, users, _ := testAuthEditorHandler(t)
	firstTerminal, _ := terminals.Create(terminalInput{HostName: "first"})
	secondTerminal, _ := terminals.Create(terminalInput{HostName: "second"})
	firstUser, _ := users.Create(userInput{Username: "operator", Password: "one"})
	duplicateUser, _ := users.Create(userInput{Username: "operator", Password: "two"})

	postForm(t, handler, "/content/terminals/"+firstTerminal.ID+"/grant", url.Values{"user_id": {firstUser.ID}}, "http://example.com", true)
	rec := postForm(t, handler, "/content/terminals/"+firstTerminal.ID+"/grant", url.Values{"user_id": {duplicateUser.ID}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "already granted access to this terminal") {
		t.Fatalf("same-terminal duplicate username accepted: %s", rec.Body.String())
	}
	rec = postForm(t, handler, "/content/terminals/"+secondTerminal.ID+"/grant", url.Values{"user_id": {duplicateUser.ID}}, "http://example.com", true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Granted access to operator") {
		t.Fatalf("cross-terminal duplicate username rejected: %d %s", rec.Code, rec.Body.String())
	}

	otherUser, _ := users.Create(userInput{Username: "other", Password: "three"})
	postForm(t, handler, "/content/terminals/"+firstTerminal.ID+"/grant", url.Values{"user_id": {otherUser.ID}}, "http://example.com", true)
	rec = postForm(t, handler, "/content/users/"+otherUser.ID, url.Values{"username": {"operator"}, "password": {"three"}}, "http://example.com", true)
	if !strings.Contains(rec.Body.String(), "already granted access to this terminal") {
		t.Fatalf("assigned user rename collision accepted: %s", rec.Body.String())
	}
}

func TestDanglingTerminalAccessIsReportedAndBlocksWrites(t *testing.T) {
	handler, dir, terminals, _, _ := testAuthEditorHandler(t)
	terminal, err := terminals.Create(terminalInput{HostName: "workstation"})
	if err != nil {
		t.Fatal(err)
	}
	missingUser := "123e4567-e89b-42d3-a456-426614174000"
	data := `{"version":1,"access":[{"user_id":"` + missingUser + `","terminal_id":"` + terminal.ID + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "terminal_access.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := getRequest(t, handler, "/content/users", nil)
	// The Terminal still exists, so the report names it and prints the
	// UUID only for the User that is gone.
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "grants access to a User that no longer exists") {
		t.Fatalf("dangling access was not reported: %d %s", rec.Code, body)
	}
	if !strings.Contains(body, "workstation") || !strings.Contains(body, missingUser) {
		t.Fatalf("report should name the surviving terminal and the dangling UUID: %s", body)
	}
	rec = postForm(t, handler, "/content/terminals", url.Values{"host_name": {"other"}}, "http://example.com", true)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "editor relationships are invalid") {
		t.Fatalf("write was not blocked: %d %s", rec.Code, rec.Body.String())
	}
}
