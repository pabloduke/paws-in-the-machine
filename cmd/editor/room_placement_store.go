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

var errRoomPlacementNotFound = errors.New("room placement not found")

type roomPlacementStore struct {
	mu   sync.Mutex
	path string
}

func newRoomPlacementStore(contentDir string) *roomPlacementStore {
	return &roomPlacementStore{path: filepath.Join(contentDir, "placements.json")}
}

func (s *roomPlacementStore) List() ([]gamecontent.RoomPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.RoomPlacement, len(file.Placements))
	copy(out, file.Placements)
	return out, nil
}

func (s *roomPlacementStore) Place(roomID, hubID string, x, y int) (gamecontent.RoomPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.RoomPlacement{}, err
	}
	placement := gamecontent.RoomPlacement{RoomID: roomID, HubID: hubID, X: x, Y: y}
	for i := range file.Placements {
		if file.Placements[i].RoomID == roomID {
			file.Placements[i] = placement
			if err := s.writeLocked(file, old, mode); err != nil {
				return gamecontent.RoomPlacement{}, err
			}
			return placement, nil
		}
	}
	file.Placements = append(file.Placements, placement)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.RoomPlacement{}, err
	}
	return placement, nil
}

func (s *roomPlacementStore) Unassign(roomID string) (gamecontent.RoomPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.RoomPlacement{}, err
	}
	for i, placement := range file.Placements {
		if placement.RoomID == roomID {
			file.Placements = append(file.Placements[:i], file.Placements[i+1:]...)
			if err := s.writeLocked(file, old, mode); err != nil {
				return gamecontent.RoomPlacement{}, err
			}
			return placement, nil
		}
	}
	return gamecontent.RoomPlacement{}, errRoomPlacementNotFound
}

func (s *roomPlacementStore) loadLocked() (gamecontent.RoomPlacementsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyRoomPlacements(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.RoomPlacementsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeRoomPlacements(data)
	if err != nil {
		return gamecontent.RoomPlacementsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *roomPlacementStore) writeLocked(file gamecontent.RoomPlacementsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeRoomPlacements(file)
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
			return fmt.Errorf("back up room placements: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write room placements: %w", err)
	}
	return nil
}
