package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHubStoreCreatesAndReloadsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	store := newHubStore(dir)
	first, err := store.Create("  hub  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create("hub")
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "hub" || first.ID == second.ID {
		t.Fatalf("unexpected created hubs: %+v %+v", first, second)
	}
	if len(first.ID) != 36 || first.ID[14] != '4' || !strings.Contains("89ab", strings.ToLower(first.ID[19:20])) {
		t.Fatalf("generated ID is not UUIDv4: %q", first.ID)
	}
	reloaded := newHubStore(dir)
	hubs, err := reloaded.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(hubs) != 2 || hubs[0].ID != first.ID || hubs[1].ID != second.ID {
		t.Fatalf("stored hub order changed: %+v", hubs)
	}
}

func TestHubStoreUpdatesWithoutChangingID(t *testing.T) {
	store := newHubStore(t.TempDir())
	created, err := store.Create("old")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(created.ID, " new ")
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || updated.Name != "new" {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func TestHubStoreRejectsInvalidWriteWithoutChangingFile(t *testing.T) {
	dir := t.TempDir()
	store := newHubStore(dir)
	created, err := store.Create("valid")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hubs.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(created.ID, " "); err == nil {
		t.Fatal("invalid update succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("invalid update changed hubs.json")
	}
}

func TestHubStoreRefusesMalformedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hubs.json")
	broken := []byte(`{"version":99,"hubs":[]}`)
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	store := newHubStore(dir)
	if _, err := store.Create("hub"); err == nil {
		t.Fatal("create succeeded over malformed hubs")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(broken) {
		t.Fatal("malformed hubs file was overwritten")
	}
}

func TestHubStoreDeletesAndPreservesRecoveryCopy(t *testing.T) {
	dir := t.TempDir()
	store := newHubStore(dir)
	first, err := store.Create("first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create("second")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hubs.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := store.Delete(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != first.ID {
		t.Fatalf("deleted hub = %+v, want %s", deleted, first.ID)
	}
	hubs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(hubs) != 1 || hubs[0].ID != second.ID {
		t.Fatalf("unexpected hubs after delete: %+v", hubs)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != string(before) {
		t.Fatal("hub recovery copy does not contain pre-delete catalog")
	}
}

func TestHubStoreReportsUnknownIDs(t *testing.T) {
	store := newHubStore(t.TempDir())
	id := "123e4567-e89b-42d3-a456-426614174000"
	if _, err := store.Get(id); err != errHubNotFound {
		t.Fatalf("unknown get error = %v", err)
	}
	if _, err := store.Update(id, "hub"); err != errHubNotFound {
		t.Fatalf("unknown update error = %v", err)
	}
	if _, err := store.Delete(id); err != errHubNotFound {
		t.Fatalf("unknown delete error = %v", err)
	}
}
