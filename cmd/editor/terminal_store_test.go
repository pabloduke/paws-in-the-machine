package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTerminalStoreCRUDDuplicateHostnamesAndRecovery(t *testing.T) {
	dir := t.TempDir()
	store := newTerminalStore(dir)
	first, err := store.Create(terminalInput{HostName: " deck "})
	if err != nil {
		t.Fatal(err)
	}
	if first.Username != "" || first.HostName != "deck" {
		t.Fatalf("terminal was not normalized: %+v", first)
	}
	second, err := store.Create(terminalInput{HostName: "undernet.relay"})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.Create(terminalInput{HostName: "deck"})
	if err != nil || duplicate.ID == first.ID {
		t.Fatalf("unassigned duplicate hostname create = %+v, %v", duplicate, err)
	}
	updated, err := store.Update(first.ID, terminalInput{HostName: "cat-deck"})
	if err != nil || updated.ID != first.ID || updated.HostName != "cat-deck" {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	path := filepath.Join(dir, "terminals.json")
	beforeDelete, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Delete(second.ID); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil || string(backup) != string(beforeDelete) {
		t.Fatalf("recovery copy mismatch: %v", err)
	}
	terminals, err := newTerminalStore(dir).List()
	if err != nil || len(terminals) != 2 || terminals[0].ID != first.ID || terminals[1].ID != duplicate.ID {
		t.Fatalf("reloaded terminals = %+v, %v", terminals, err)
	}
}

func TestTerminalStoreRejectsInvalidAndMalformedWrites(t *testing.T) {
	dir := t.TempDir()
	store := newTerminalStore(dir)
	terminal, err := store.Create(terminalInput{HostName: "deck"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "terminals.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(terminal.ID, terminalInput{HostName: "Bad Host"}); err == nil {
		t.Fatal("invalid update succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("invalid update changed catalog: %v", err)
	}

	brokenDir := t.TempDir()
	brokenPath := filepath.Join(brokenDir, "terminals.json")
	broken := []byte(`{"version":99,"terminals":[]}`)
	if err := os.WriteFile(brokenPath, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newTerminalStore(brokenDir).Create(terminalInput{HostName: "deck"}); err == nil {
		t.Fatal("create succeeded over malformed catalog")
	}
	got, err := os.ReadFile(brokenPath)
	if err != nil || string(got) != string(broken) {
		t.Fatalf("malformed catalog changed: %v", err)
	}
}

func TestTerminalStoreReportsUnknownIDs(t *testing.T) {
	store := newTerminalStore(t.TempDir())
	id := "123e4567-e89b-42d3-a456-426614174000"
	if _, err := store.Get(id); !errors.Is(err, errTerminalNotFound) {
		t.Fatalf("unknown get error = %v", err)
	}
	if _, err := store.Update(id, terminalInput{HostName: "deck"}); !errors.Is(err, errTerminalNotFound) {
		t.Fatalf("unknown update error = %v", err)
	}
	if _, err := store.Delete(id); !errors.Is(err, errTerminalNotFound) {
		t.Fatalf("unknown delete error = %v", err)
	}
}
