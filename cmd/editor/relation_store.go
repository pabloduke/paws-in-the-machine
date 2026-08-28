package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// errRelationNotFound is returned when a keyed relation record is absent.
// Callers that need a caller-facing distinction wrap it.
var errRelationNotFound = errors.New("relation not found")

// relationStore is one versioned JSON catalog of relation records —
// containment, placement, entry rooms, cell contents. Every relation
// file in the editor has the same shape: a list of records, each
// identified by one key, written atomically with a single recovery
// copy. The record type and its key are the only things that differ,
// so they are the only things a caller supplies.
//
// This mirrors describedStore, which does the same job for definition
// catalogs.
type relationStore[T any] struct {
	mu     sync.Mutex
	path   string
	label  string
	key    func(T) string
	decode func([]byte) ([]T, error)
	encode func([]T) ([]byte, error)
}

func (s *relationStore[T]) List() ([]T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]T, len(records))
	copy(out, records)
	return out, nil
}

func (s *relationStore[T]) Get(key string) (T, error) {
	records, err := s.List()
	if err != nil {
		var zero T
		return zero, err
	}
	for _, record := range records {
		if s.key(record) == key {
			return record, nil
		}
	}
	var zero T
	return zero, errRelationNotFound
}

// Put inserts the record, or replaces the existing record with the same
// key. Relations are upserts by nature: assigning an already-assigned
// child moves it rather than creating a second edge.
func (s *relationStore[T]) Put(record T) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, old, mode, err := s.loadLocked()
	if err != nil {
		var zero T
		return zero, err
	}
	key := s.key(record)
	replaced := false
	for i := range records {
		if s.key(records[i]) == key {
			records[i], replaced = record, true
			break
		}
	}
	if !replaced {
		records = append(records, record)
	}
	if err := s.writeLocked(records, old, mode); err != nil {
		var zero T
		return zero, err
	}
	return record, nil
}

func (s *relationStore[T]) Delete(key string) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, old, mode, err := s.loadLocked()
	if err != nil {
		var zero T
		return zero, err
	}
	for i, record := range records {
		if s.key(record) != key {
			continue
		}
		records = append(records[:i], records[i+1:]...)
		if err := s.writeLocked(records, old, mode); err != nil {
			var zero T
			return zero, err
		}
		return record, nil
	}
	var zero T
	return zero, errRelationNotFound
}

func (s *relationStore[T]) loadLocked() ([]T, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []T{}, nil, 0o644, nil
	}
	if err != nil {
		return nil, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	records, err := s.decode(data)
	if err != nil {
		return nil, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return records, data, mode, nil
}

func (s *relationStore[T]) writeLocked(records []T, old []byte, mode os.FileMode) error {
	data, err := s.encode(records)
	if err != nil {
		return err
	}
	if bytes.Equal(data, old) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create content directory: %w", err)
	}
	if old != nil {
		if err := writeAtomic(s.path+".bak", old, mode); err != nil {
			return fmt.Errorf("back up %s: %w", s.label, err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", s.label, err)
	}
	return nil
}
