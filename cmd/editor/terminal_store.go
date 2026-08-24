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

var (
	errTerminalNotFound = errors.New("terminal not found")
)

type terminalInput struct {
	HostName string
}

func (in terminalInput) normalized() terminalInput {
	return terminalInput{HostName: strings.TrimSpace(in.HostName)}
}

type terminalStore struct {
	mu   sync.Mutex
	path string
}

func newTerminalStore(contentDir string) *terminalStore {
	return &terminalStore{path: filepath.Join(contentDir, "terminals.json")}
}

func (s *terminalStore) List() ([]gamecontent.Terminal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.Terminal, len(file.Terminals))
	copy(out, file.Terminals)
	return out, nil
}

func (s *terminalStore) Get(id string) (gamecontent.Terminal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return gamecontent.Terminal{}, err
	}
	for _, terminal := range file.Terminals {
		if terminal.ID == id {
			return terminal, nil
		}
	}
	return gamecontent.Terminal{}, errTerminalNotFound
}

func (s *terminalStore) Create(input terminalInput) (gamecontent.Terminal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.Terminal{}, err
	}
	input = input.normalized()
	id, err := newUUIDv4()
	if err != nil {
		return gamecontent.Terminal{}, err
	}
	terminal := gamecontent.Terminal{ID: id, HostName: input.HostName}
	file.Terminals = append(file.Terminals, terminal)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.Terminal{}, err
	}
	return terminal, nil
}

func (s *terminalStore) Update(id string, input terminalInput) (gamecontent.Terminal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.Terminal{}, err
	}
	input = input.normalized()
	for i := range file.Terminals {
		if file.Terminals[i].ID != id {
			continue
		}
		file.Terminals[i].HostName = input.HostName
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.Terminal{}, err
		}
		return file.Terminals[i], nil
	}
	return gamecontent.Terminal{}, errTerminalNotFound
}

func (s *terminalStore) Delete(id string) (gamecontent.Terminal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.Terminal{}, err
	}
	for i, terminal := range file.Terminals {
		if terminal.ID != id {
			continue
		}
		file.Terminals = append(file.Terminals[:i], file.Terminals[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.Terminal{}, err
		}
		return terminal, nil
	}
	return gamecontent.Terminal{}, errTerminalNotFound
}

func (s *terminalStore) loadLocked() (gamecontent.TerminalsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyTerminals(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.TerminalsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeTerminals(data)
	if err != nil {
		return gamecontent.TerminalsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *terminalStore) writeLocked(file gamecontent.TerminalsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeTerminals(file)
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
			return fmt.Errorf("back up terminals: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write terminals: %w", err)
	}
	return nil
}
