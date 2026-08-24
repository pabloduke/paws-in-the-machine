package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHostNetworkStoreCRUDAndRecovery(t *testing.T) {
	dir := t.TempDir()
	store := newHostNetworkStore(dir)
	network, err := store.Create(" Network ")
	if err != nil || network.Name != "Network" {
		t.Fatalf("create = %+v, %v", network, err)
	}
	updated, err := store.Update(network.ID, "Updated")
	if err != nil || updated.ID != network.ID || updated.Name != "Updated" {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	path := filepath.Join(dir, "host_networks.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Delete(network.ID); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil || string(backup) != string(before) {
		t.Fatalf("recovery copy mismatch: %v", err)
	}
	if _, err := store.Get(network.ID); !errors.Is(err, errHostNetworkNotFound) {
		t.Fatalf("unknown get error = %v", err)
	}
}

func TestNetworkAssignmentStoreAssignMoveUnassignAndRecover(t *testing.T) {
	dir := t.TempDir()
	store := newNetworkAssignmentStore(dir)
	terminalID := "123e4567-e89b-42d3-a456-426614174010"
	firstNetwork := "123e4567-e89b-42d3-a456-426614174011"
	secondNetwork := "123e4567-e89b-42d3-a456-426614174012"
	if _, err := store.Assign(terminalID, firstNetwork); err != nil {
		t.Fatal(err)
	}
	moved, err := store.Assign(terminalID, secondNetwork)
	if err != nil || moved.HostNetworkID != secondNetwork {
		t.Fatalf("move = %+v, %v", moved, err)
	}
	assignments, err := store.List()
	if err != nil || len(assignments) != 1 {
		t.Fatalf("assignments = %+v, %v", assignments, err)
	}
	path := filepath.Join(dir, "network_assignments.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Unassign(terminalID); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil || string(backup) != string(before) {
		t.Fatalf("recovery copy mismatch: %v", err)
	}
	if _, err := store.GetByTerminal(terminalID); !errors.Is(err, errNetworkAssignmentNotFound) {
		t.Fatalf("unknown assignment error = %v", err)
	}
}
