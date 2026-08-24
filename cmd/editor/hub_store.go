package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errHubNotFound = errors.New("hub not found")

type hubStore struct {
	mu   sync.Mutex
	path string
}

func newHubStore(contentDir string) *hubStore {
	return &hubStore{path: filepath.Join(contentDir, "hubs.json")}
}

func (s *hubStore) List() ([]gamecontent.Hub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.Hub, len(file.Hubs))
	copy(out, file.Hubs)
	return out, nil
}

func (s *hubStore) Get(id string) (gamecontent.Hub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return gamecontent.Hub{}, err
	}
	for _, hub := range file.Hubs {
		if hub.ID == id {
			return hub, nil
		}
	}
	return gamecontent.Hub{}, errHubNotFound
}

func (s *hubStore) Create(name string) (gamecontent.Hub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.Hub{}, err
	}
	id, err := newUUIDv4()
	if err != nil {
		return gamecontent.Hub{}, err
	}
	hub := gamecontent.Hub{ID: id, Name: strings.TrimSpace(name)}
	file.Hubs = append(file.Hubs, hub)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.Hub{}, err
	}
	return hub, nil
}

func (s *hubStore) Update(id, name string) (gamecontent.Hub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.Hub{}, err
	}
	for i := range file.Hubs {
		if file.Hubs[i].ID != id {
			continue
		}
		file.Hubs[i].Name = strings.TrimSpace(name)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.Hub{}, err
		}
		return file.Hubs[i], nil
	}
	return gamecontent.Hub{}, errHubNotFound
}

func (s *hubStore) Delete(id string) (gamecontent.Hub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.Hub{}, err
	}
	for i, hub := range file.Hubs {
		if hub.ID != id {
			continue
		}
		file.Hubs = append(file.Hubs[:i], file.Hubs[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.Hub{}, err
		}
		return hub, nil
	}
	return gamecontent.Hub{}, errHubNotFound
}

func (s *hubStore) loadLocked() (gamecontent.HubsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyHubs(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.HubsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeHubs(data)
	if err != nil {
		return gamecontent.HubsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *hubStore) writeLocked(file gamecontent.HubsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeHubs(file)
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
			return fmt.Errorf("back up hubs: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write hubs: %w", err)
	}
	return nil
}
