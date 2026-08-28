package main

import (
	"os"
	"path/filepath"
	"testing"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const testItemUUID = "11111111-1111-4111-8111-111111111111"
const testRoomUUID = "22222222-2222-4222-8222-222222222222"
const testLocationUUID = "33333333-3333-4333-8333-333333333333"

func TestContentsStoreUpsertsMovesAndRemoves(t *testing.T) {
	dir := t.TempDir()
	store := newContentsStore(dir)

	if records, err := store.List(); err != nil || len(records) != 0 {
		t.Fatalf("a missing catalog should read as empty: %+v %v", records, err)
	}
	if _, err := store.Remove(gamecontent.ContentKindWorldItem, testItemUUID); err != errContentNotFound {
		t.Fatalf("removing nothing = %v, want errContentNotFound", err)
	}

	if _, err := store.PlaceIn(gamecontent.ContentKindWorldItem, testItemUUID, gamecontent.ContainerKindRoom, testRoomUUID); err != nil {
		t.Fatalf("place: %v", err)
	}
	// Placing the same entity elsewhere moves it rather than adding a
	// second record: an entity is in at most one cell.
	if _, err := store.PlaceIn(gamecontent.ContentKindWorldItem, testItemUUID, gamecontent.ContainerKindLocation, testLocationUUID); err != nil {
		t.Fatalf("move: %v", err)
	}
	records, err := store.List()
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %+v, %v", records, err)
	}
	if records[0].ParentID != testLocationUUID || records[0].ParentKind != gamecontent.ContainerKindLocation {
		t.Fatalf("entity did not move: %+v", records[0])
	}

	record, held, err := store.ContainerOf(gamecontent.ContentKindWorldItem, testItemUUID)
	if err != nil || !held || record.ParentID != testLocationUUID {
		t.Fatalf("ContainerOf = %+v %v %v", record, held, err)
	}
	// An NPC that shares the item's UUID is a different entity.
	if _, held, err = store.ContainerOf(gamecontent.ContentKindNPC, testItemUUID); err != nil || held {
		t.Fatalf("kinds share a key space: %v %v", held, err)
	}

	if _, err := store.Remove(gamecontent.ContentKindWorldItem, testItemUUID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if records, _ = store.List(); len(records) != 0 {
		t.Fatalf("records after remove = %+v", records)
	}
}

func TestContentsStoreWritesAtomicallyAndKeepsOneRecoveryCopy(t *testing.T) {
	dir := t.TempDir()
	store := newContentsStore(dir)
	path := filepath.Join(dir, "contents.json")

	if _, err := store.PlaceIn(gamecontent.ContentKindNPC, testItemUUID, gamecontent.ContainerKindRoom, testRoomUUID); err != nil {
		t.Fatalf("place: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("catalog was not created: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("a first write should not leave a recovery copy")
	}

	// Rewriting identical content must not touch the file at all, so
	// unchanged content stays byte-identical for Git review.
	if _, err := store.PlaceIn(gamecontent.ContentKindNPC, testItemUUID, gamecontent.ContainerKindRoom, testRoomUUID); err != nil {
		t.Fatalf("idempotent place: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("an unchanged write still rewrote the catalog")
	}

	if _, err := store.PlaceIn(gamecontent.ContentKindNPC, testItemUUID, gamecontent.ContainerKindLocation, testLocationUUID); err != nil {
		t.Fatalf("move: %v", err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("recovery copy missing: %v", err)
	}
	if string(backup) != string(first) {
		t.Fatal("recovery copy does not hold the pre-write bytes")
	}
}
