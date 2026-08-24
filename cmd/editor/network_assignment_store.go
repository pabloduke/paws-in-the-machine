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

var errNetworkAssignmentNotFound = errors.New("network assignment not found")

type networkAssignmentStore struct {
	mu   sync.Mutex
	path string
}

func newNetworkAssignmentStore(contentDir string) *networkAssignmentStore {
	return &networkAssignmentStore{path: filepath.Join(contentDir, "network_assignments.json")}
}

func (s *networkAssignmentStore) List() ([]gamecontent.NetworkAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.NetworkAssignment, len(file.Assignments))
	copy(out, file.Assignments)
	return out, nil
}

func (s *networkAssignmentStore) GetByTerminal(terminalID string) (gamecontent.NetworkAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return gamecontent.NetworkAssignment{}, err
	}
	for _, assignment := range file.Assignments {
		if assignment.TerminalID == terminalID {
			return assignment, nil
		}
	}
	return gamecontent.NetworkAssignment{}, errNetworkAssignmentNotFound
}

func (s *networkAssignmentStore) Assign(terminalID, networkID string) (gamecontent.NetworkAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.NetworkAssignment{}, err
	}
	assignment := gamecontent.NetworkAssignment{TerminalID: terminalID, HostNetworkID: networkID}
	for i := range file.Assignments {
		if file.Assignments[i].TerminalID != terminalID {
			continue
		}
		file.Assignments[i] = assignment
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.NetworkAssignment{}, err
		}
		return assignment, nil
	}
	file.Assignments = append(file.Assignments, assignment)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.NetworkAssignment{}, err
	}
	return assignment, nil
}

func (s *networkAssignmentStore) Unassign(terminalID string) (gamecontent.NetworkAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.NetworkAssignment{}, err
	}
	for i, assignment := range file.Assignments {
		if assignment.TerminalID != terminalID {
			continue
		}
		file.Assignments = append(file.Assignments[:i], file.Assignments[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.NetworkAssignment{}, err
		}
		return assignment, nil
	}
	return gamecontent.NetworkAssignment{}, errNetworkAssignmentNotFound
}

func (s *networkAssignmentStore) loadLocked() (gamecontent.NetworkAssignmentsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyNetworkAssignments(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.NetworkAssignmentsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeNetworkAssignments(data)
	if err != nil {
		return gamecontent.NetworkAssignmentsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *networkAssignmentStore) writeLocked(file gamecontent.NetworkAssignmentsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeNetworkAssignments(file)
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
			return fmt.Errorf("back up network assignments: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write network assignments: %w", err)
	}
	return nil
}
