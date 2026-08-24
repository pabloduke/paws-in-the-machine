package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUserAndTerminalAccessStores(t *testing.T) {
	dir := t.TempDir()
	users := newUserStore(dir)
	access := newTerminalAccessStore(dir)
	user, err := users.Create(userInput{Username: " operator ", Password: " fictional password "})
	if err != nil || user.Username != "operator" || user.Password != " fictional password " {
		t.Fatalf("create user = %+v, %v", user, err)
	}
	updated, err := users.Update(user.ID, userInput{Username: "operator_2", Password: "new fictional password"})
	if err != nil || updated.Username != "operator_2" {
		t.Fatalf("update user = %+v, %v", updated, err)
	}
	terminalID := "123e4567-e89b-42d3-a456-426614174001"
	grant, err := access.Grant(user.ID, terminalID)
	if err != nil || grant.UserID != user.ID || grant.TerminalID != terminalID {
		t.Fatalf("grant = %+v, %v", grant, err)
	}
	if _, err := access.Grant(user.ID, terminalID); err != nil {
		t.Fatalf("idempotent grant failed: %v", err)
	}
	grants, err := access.List()
	if err != nil || len(grants) != 1 {
		t.Fatalf("grants = %+v, %v", grants, err)
	}
	if _, err := access.Revoke(user.ID, terminalID); err != nil {
		t.Fatal(err)
	}
	if _, err := access.Revoke(user.ID, terminalID); !errors.Is(err, errTerminalAccessNotFound) {
		t.Fatalf("missing revoke error = %v", err)
	}
	if _, err := users.Delete(user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "users.json.bak")); err != nil {
		t.Fatalf("user recovery copy missing: %v", err)
	}
}

func TestAuthStoresRefuseMalformedCatalogs(t *testing.T) {
	dir := t.TempDir()
	usersPath := filepath.Join(dir, "users.json")
	if err := os.WriteFile(usersPath, []byte(`{"version":99,"users":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newUserStore(dir).Create(userInput{Username: "operator", Password: "fictional"}); err == nil {
		t.Fatal("created user over malformed catalog")
	}
	accessPath := filepath.Join(dir, "terminal_access.json")
	if err := os.WriteFile(accessPath, []byte(`{"version":99,"access":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newTerminalAccessStore(dir).Grant("123e4567-e89b-42d3-a456-426614174000", "123e4567-e89b-42d3-a456-426614174001"); err == nil {
		t.Fatal("created access over malformed catalog")
	}
}
