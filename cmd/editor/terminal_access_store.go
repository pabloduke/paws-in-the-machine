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

var errTerminalAccessNotFound = errors.New("terminal access grant not found")

type terminalAccessStore struct {
	mu   sync.Mutex
	path string
}

func newTerminalAccessStore(contentDir string) *terminalAccessStore {
	return &terminalAccessStore{path: filepath.Join(contentDir, "terminal_access.json")}
}

func (s *terminalAccessStore) List() ([]gamecontent.TerminalAccess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.TerminalAccess, len(file.Access))
	copy(out, file.Access)
	return out, nil
}

func (s *terminalAccessStore) Grant(userID, terminalID string) (gamecontent.TerminalAccess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.TerminalAccess{}, err
	}
	grant := gamecontent.TerminalAccess{UserID: userID, TerminalID: terminalID}
	for _, existing := range file.Access {
		if existing == grant {
			return existing, nil
		}
	}
	file.Access = append(file.Access, grant)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.TerminalAccess{}, err
	}
	return grant, nil
}

func (s *terminalAccessStore) Revoke(userID, terminalID string) (gamecontent.TerminalAccess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.TerminalAccess{}, err
	}
	for i, grant := range file.Access {
		if grant.UserID != userID || grant.TerminalID != terminalID {
			continue
		}
		file.Access = append(file.Access[:i], file.Access[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.TerminalAccess{}, err
		}
		return grant, nil
	}
	return gamecontent.TerminalAccess{}, errTerminalAccessNotFound
}

func (s *terminalAccessStore) loadLocked() (gamecontent.TerminalAccessFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyTerminalAccess(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.TerminalAccessFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeTerminalAccess(data)
	if err != nil {
		return gamecontent.TerminalAccessFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *terminalAccessStore) writeLocked(file gamecontent.TerminalAccessFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeTerminalAccess(file)
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
			return fmt.Errorf("back up terminal access: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write terminal access: %w", err)
	}
	return nil
}
