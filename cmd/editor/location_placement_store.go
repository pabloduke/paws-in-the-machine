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

var errLocationPlacementNotFound = errors.New("location placement not found")

type locationPlacementStore struct {
	mu   sync.Mutex
	path string
}

func newLocationPlacementStore(contentDir string) *locationPlacementStore {
	return &locationPlacementStore{path: filepath.Join(contentDir, "location_placements.json")}
}

func (s *locationPlacementStore) List() ([]gamecontent.LocationPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := append([]gamecontent.LocationPlacement(nil), file.Placements...)
	return out, nil
}

func (s *locationPlacementStore) Place(locationID, hubID string, x, y int) (gamecontent.LocationPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.LocationPlacement{}, err
	}
	placement := gamecontent.LocationPlacement{LocationID: locationID, HubID: hubID, X: x, Y: y}
	for i := range file.Placements {
		if file.Placements[i].LocationID == locationID {
			file.Placements[i] = placement
			if err := s.writeLocked(file, old, mode); err != nil {
				return gamecontent.LocationPlacement{}, err
			}
			return placement, nil
		}
	}
	file.Placements = append(file.Placements, placement)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.LocationPlacement{}, err
	}
	return placement, nil
}

func (s *locationPlacementStore) Unassign(locationID string) (gamecontent.LocationPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.LocationPlacement{}, err
	}
	for i, placement := range file.Placements {
		if placement.LocationID == locationID {
			file.Placements = append(file.Placements[:i], file.Placements[i+1:]...)
			if err := s.writeLocked(file, old, mode); err != nil {
				return gamecontent.LocationPlacement{}, err
			}
			return placement, nil
		}
	}
	return gamecontent.LocationPlacement{}, errLocationPlacementNotFound
}

func (s *locationPlacementStore) loadLocked() (gamecontent.LocationPlacementsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyLocationPlacements(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.LocationPlacementsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeLocationPlacements(data)
	if err != nil {
		return gamecontent.LocationPlacementsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *locationPlacementStore) writeLocked(file gamecontent.LocationPlacementsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeLocationPlacements(file)
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
			return fmt.Errorf("back up location placements: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write location placements: %w", err)
	}
	return nil
}
