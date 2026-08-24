package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

func testWorldItemInput(name string) worldItemInput {
	return worldItemInput{
		Name:             name,
		Kind:             gamecontent.WorldItemTakeable,
		ShortDescription: "short description",
		FullDescription:  "full description",
	}
}

func TestStoreCreatesAndReloadsWorldItems(t *testing.T) {
	dir := t.TempDir()
	store := newWorldItemStore(dir)
	first, err := store.Create(testWorldItemInput("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(testWorldItemInput("first"))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatal("created items did not receive unique UUIDs")
	}
	if len(first.ID) != 36 || first.ID[14] != '4' || !strings.Contains("89ab", strings.ToLower(first.ID[19:20])) {
		t.Fatalf("generated ID is not an RFC 4122 UUIDv4: %q", first.ID)
	}

	reloaded := newWorldItemStore(dir)
	items, err := reloaded.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != first.ID || items[1].ID != second.ID {
		t.Fatalf("stored item order changed: %+v", items)
	}
}

func TestStorePersistsEveryWorldItemKind(t *testing.T) {
	store := newWorldItemStore(t.TempDir())
	for _, kind := range []gamecontent.WorldItemKind{
		gamecontent.WorldItemTakeable,
		gamecontent.WorldItemFixed,
		gamecontent.WorldItemScenery,
	} {
		input := testWorldItemInput(string(kind))
		input.Kind = kind
		if _, err := store.Create(input); err != nil {
			t.Fatalf("create %s: %v", kind, err)
		}
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
}

func TestStoreNormalizesAndUpdatesWithoutChangingID(t *testing.T) {
	dir := t.TempDir()
	store := newWorldItemStore(dir)
	created, err := store.Create(worldItemInput{
		Name:             "  item  ",
		Kind:             gamecontent.WorldItemFixed,
		ShortDescription: "  short\r\nline  ",
		FullDescription:  "  full  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "item" || created.ShortDescription != "short\nline" {
		t.Fatalf("input was not normalized: %+v", created)
	}

	updatedInput := testWorldItemInput("renamed")
	updatedInput.Kind = gamecontent.WorldItemScenery
	updated, err := store.Update(created.ID, updatedInput)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || updated.Name != "renamed" || updated.Kind != gamecontent.WorldItemScenery {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func TestStoreRejectsInvalidInputWithoutChangingFile(t *testing.T) {
	dir := t.TempDir()
	store := newWorldItemStore(dir)
	created, err := store.Create(testWorldItemInput("valid"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "world_items.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	invalid := testWorldItemInput(" ")
	if _, err := store.Update(created.ID, invalid); err == nil {
		t.Fatal("invalid update succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("invalid update changed the content file")
	}
}

func TestStoreRefusesMalformedFileWithoutOverwritingIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "world_items.json")
	broken := []byte(`{"version":99,"world_items":[]}`)
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	store := newWorldItemStore(dir)
	if _, err := store.Create(testWorldItemInput("item")); err == nil {
		t.Fatal("create succeeded over malformed content")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(broken) {
		t.Fatal("malformed file was overwritten")
	}
}

func TestStoreKeepsRecoveryCopyOnReplacement(t *testing.T) {
	dir := t.TempDir()
	store := newWorldItemStore(dir)
	if _, err := store.Create(testWorldItemInput("first")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "world_items.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(testWorldItemInput("second")); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != string(before) {
		t.Fatal("recovery copy does not contain the previous file")
	}
}

func TestStoreReportsUnknownUpdate(t *testing.T) {
	store := newWorldItemStore(t.TempDir())
	if _, err := store.Update("123e4567-e89b-42d3-a456-426614174000", testWorldItemInput("item")); err != errWorldItemNotFound {
		t.Fatalf("unknown update error = %v, want %v", err, errWorldItemNotFound)
	}
}

func TestStoreDeletesItemAndPreservesRecoveryCopy(t *testing.T) {
	dir := t.TempDir()
	store := newWorldItemStore(dir)
	first, err := store.Create(testWorldItemInput("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(testWorldItemInput("second"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "world_items.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := store.Delete(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != first.ID {
		t.Fatalf("deleted item = %+v, want %s", deleted, first.ID)
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("unexpected catalog after delete: %+v", items)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != string(before) {
		t.Fatal("delete recovery copy does not contain the pre-delete catalog")
	}
}

func TestStoreReportsUnknownDelete(t *testing.T) {
	store := newWorldItemStore(t.TempDir())
	if _, err := store.Delete("123e4567-e89b-42d3-a456-426614174000"); err != errWorldItemNotFound {
		t.Fatalf("unknown delete error = %v, want %v", err, errWorldItemNotFound)
	}
}
