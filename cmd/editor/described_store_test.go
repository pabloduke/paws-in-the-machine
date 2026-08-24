package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRoomStoreCRUDAndRecovery(t *testing.T) {
	dir := t.TempDir()
	store := newRoomStore(dir)
	first, err := store.Create(describedInput{Name: "  Room  ", Description: " line one\r\nline two "})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(describedInput{Name: "Room", Description: "Another description"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Room" || first.Description != "line one\nline two" || first.ID == second.ID {
		t.Fatalf("unexpected rooms: %+v %+v", first, second)
	}
	updated, err := store.Update(first.ID, describedInput{Name: "Updated", Description: "Updated description"})
	if err != nil || updated.ID != first.ID || updated.Name != "Updated" {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	path := filepath.Join(dir, "rooms.json")
	beforeDelete, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := store.Delete(first.ID)
	if err != nil || deleted.ID != first.ID {
		t.Fatalf("delete = %+v, %v", deleted, err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil || string(backup) != string(beforeDelete) {
		t.Fatalf("recovery copy mismatch: %v", err)
	}
	reloaded, err := newRoomStore(dir).List()
	if err != nil || len(reloaded) != 1 || reloaded[0].ID != second.ID {
		t.Fatalf("reloaded rooms = %+v, %v", reloaded, err)
	}
}

func TestNPCStoreCRUDAndInvalidWriteProtection(t *testing.T) {
	dir := t.TempDir()
	store := newNPCStore(dir)
	npc, err := store.Create(describedInput{Name: "NPC", Description: "Description"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "npcs.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(npc.ID, describedInput{Name: "", Description: "Description"}); err == nil {
		t.Fatal("invalid update succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("invalid update changed catalog: %v", err)
	}
	unknown := "123e4567-e89b-42d3-a456-426614174999"
	if _, err := store.Get(unknown); !errors.Is(err, errDescribedEntityNotFound) {
		t.Fatalf("unknown get error = %v", err)
	}
}

func TestDescribedStoreRefusesMalformedCatalog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rooms.json")
	broken := []byte(`{"version":99,"rooms":[]}`)
	if err := os.WriteFile(path, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newRoomStore(dir).Create(describedInput{Name: "Room", Description: "Description"}); err == nil {
		t.Fatal("create succeeded over malformed catalog")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(broken) {
		t.Fatalf("malformed catalog changed: %v", err)
	}
}
