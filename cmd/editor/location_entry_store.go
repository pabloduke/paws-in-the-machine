package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errLocationEntryNotFound = errors.New("location entry not found")

type locationEntryStore struct {
	mu   sync.Mutex
	path string
}

func newLocationEntryStore(contentDir string) *locationEntryStore {
	return &locationEntryStore{path: filepath.Join(contentDir, "location_entry_rooms.json")}
}

func (s *locationEntryStore) List() ([]gamecontent.LocationEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return append([]gamecontent.LocationEntry(nil), file.Entries...), nil
}

func (s *locationEntryStore) Set(locationID, roomID string) (gamecontent.LocationEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.LocationEntry{}, err
	}
	entry := gamecontent.LocationEntry{LocationID: locationID, RoomID: roomID}
	for i := range file.Entries {
		if file.Entries[i].LocationID == locationID {
			file.Entries[i] = entry
			if err := s.writeLocked(file, old, mode); err != nil {
				return gamecontent.LocationEntry{}, err
			}
			return entry, nil
		}
	}
	file.Entries = append(file.Entries, entry)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.LocationEntry{}, err
	}
	return entry, nil
}

func (s *locationEntryStore) Clear(locationID string) (gamecontent.LocationEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.LocationEntry{}, err
	}
	for i, entry := range file.Entries {
		if entry.LocationID == locationID {
			file.Entries = append(file.Entries[:i], file.Entries[i+1:]...)
			if err := s.writeLocked(file, old, mode); err != nil {
				return gamecontent.LocationEntry{}, err
			}
			return entry, nil
		}
	}
	return gamecontent.LocationEntry{}, errLocationEntryNotFound
}

func (s *locationEntryStore) loadLocked() (gamecontent.LocationEntriesFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyLocationEntries(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.LocationEntriesFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeLocationEntries(data)
	if err != nil {
		return gamecontent.LocationEntriesFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *locationEntryStore) writeLocked(file gamecontent.LocationEntriesFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeLocationEntries(file)
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
			return fmt.Errorf("back up location entries: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write location entries: %w", err)
	}
	return nil
}
