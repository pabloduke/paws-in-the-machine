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

var errLocationAssignmentNotFound = errors.New("location assignment not found")
var errRoomAssignmentNotFound = errors.New("room assignment not found")

type locationAssignmentStore struct {
	mu   sync.Mutex
	path string
}

func newLocationAssignmentStore(contentDir string) *locationAssignmentStore {
	return &locationAssignmentStore{path: filepath.Join(contentDir, "location_assignments.json")}
}

func (s *locationAssignmentStore) List() ([]gamecontent.LocationAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	return append([]gamecontent.LocationAssignment(nil), file.Assignments...), err
}

func (s *locationAssignmentStore) Get(locationID string) (gamecontent.LocationAssignment, error) {
	assignments, err := s.List()
	if err != nil {
		return gamecontent.LocationAssignment{}, err
	}
	for _, assignment := range assignments {
		if assignment.LocationID == locationID {
			return assignment, nil
		}
	}
	return gamecontent.LocationAssignment{}, errLocationAssignmentNotFound
}

func (s *locationAssignmentStore) Assign(locationID, hubID string) (gamecontent.LocationAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.LocationAssignment{}, err
	}
	assignment := gamecontent.LocationAssignment{LocationID: locationID, HubID: hubID}
	for i := range file.Assignments {
		if file.Assignments[i].LocationID == locationID {
			file.Assignments[i] = assignment
			err = s.writeLocked(file, old, mode)
			return assignment, err
		}
	}
	file.Assignments = append(file.Assignments, assignment)
	err = s.writeLocked(file, old, mode)
	return assignment, err
}

func (s *locationAssignmentStore) Unassign(locationID string) (gamecontent.LocationAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.LocationAssignment{}, err
	}
	for i, assignment := range file.Assignments {
		if assignment.LocationID == locationID {
			file.Assignments = append(file.Assignments[:i], file.Assignments[i+1:]...)
			err = s.writeLocked(file, old, mode)
			return assignment, err
		}
	}
	return gamecontent.LocationAssignment{}, errLocationAssignmentNotFound
}

func (s *locationAssignmentStore) loadLocked() (gamecontent.LocationAssignmentsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyLocationAssignments(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.LocationAssignmentsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeLocationAssignments(data)
	if err != nil {
		return gamecontent.LocationAssignmentsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *locationAssignmentStore) writeLocked(file gamecontent.LocationAssignmentsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeLocationAssignments(file)
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
			return fmt.Errorf("back up location assignments: %w", err)
		}
	}
	return writeAtomic(s.path, data, mode)
}

type roomAssignmentStore struct {
	mu   sync.Mutex
	path string
}

func newRoomAssignmentStore(contentDir string) *roomAssignmentStore {
	return &roomAssignmentStore{path: filepath.Join(contentDir, "room_assignments.json")}
}

func (s *roomAssignmentStore) List() ([]gamecontent.RoomAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	return append([]gamecontent.RoomAssignment(nil), file.Assignments...), err
}

func (s *roomAssignmentStore) Get(roomID string) (gamecontent.RoomAssignment, error) {
	assignments, err := s.List()
	if err != nil {
		return gamecontent.RoomAssignment{}, err
	}
	for _, assignment := range assignments {
		if assignment.RoomID == roomID {
			return assignment, nil
		}
	}
	return gamecontent.RoomAssignment{}, errRoomAssignmentNotFound
}

func (s *roomAssignmentStore) Assign(roomID, locationID string) (gamecontent.RoomAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.RoomAssignment{}, err
	}
	assignment := gamecontent.RoomAssignment{RoomID: roomID, LocationID: locationID}
	for i := range file.Assignments {
		if file.Assignments[i].RoomID == roomID {
			file.Assignments[i] = assignment
			err = s.writeLocked(file, old, mode)
			return assignment, err
		}
	}
	file.Assignments = append(file.Assignments, assignment)
	err = s.writeLocked(file, old, mode)
	return assignment, err
}

func (s *roomAssignmentStore) Unassign(roomID string) (gamecontent.RoomAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.RoomAssignment{}, err
	}
	for i, assignment := range file.Assignments {
		if assignment.RoomID == roomID {
			file.Assignments = append(file.Assignments[:i], file.Assignments[i+1:]...)
			err = s.writeLocked(file, old, mode)
			return assignment, err
		}
	}
	return gamecontent.RoomAssignment{}, errRoomAssignmentNotFound
}

func (s *roomAssignmentStore) loadLocked() (gamecontent.RoomAssignmentsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyRoomAssignments(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.RoomAssignmentsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeRoomAssignments(data)
	if err != nil {
		return gamecontent.RoomAssignmentsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *roomAssignmentStore) writeLocked(file gamecontent.RoomAssignmentsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeRoomAssignments(file)
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
			return fmt.Errorf("back up room assignments: %w", err)
		}
	}
	return writeAtomic(s.path, data, mode)
}
