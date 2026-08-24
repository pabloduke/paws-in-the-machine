package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRoomPlacementStorePlaceMoveUnassignAndRecovery(t *testing.T) {
	dir := t.TempDir()
	store := newRoomPlacementStore(dir)
	roomID := "123e4567-e89b-42d3-a456-426614174000"
	hubID := "123e4567-e89b-42d3-a456-426614174001"
	if _, err := store.Place(roomID, hubID, 2, 3); err != nil {
		t.Fatal(err)
	}
	moved, err := store.Place(roomID, hubID, 11, 4)
	if err != nil || moved.X != 11 || moved.Y != 4 {
		t.Fatalf("move = %+v, %v", moved, err)
	}
	if _, err := store.Unassign(roomID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Unassign(roomID); !errors.Is(err, errRoomPlacementNotFound) {
		t.Fatalf("missing unassign error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "placements.json.bak")); err != nil {
		t.Fatalf("recovery copy missing: %v", err)
	}
}
