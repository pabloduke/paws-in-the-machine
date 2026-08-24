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

var errHostNetworkNotFound = errors.New("host network not found")

type hostNetworkStore struct {
	mu   sync.Mutex
	path string
}

func newHostNetworkStore(contentDir string) *hostNetworkStore {
	return &hostNetworkStore{path: filepath.Join(contentDir, "host_networks.json")}
}

func (s *hostNetworkStore) List() ([]gamecontent.HostNetwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.HostNetwork, len(file.HostNetworks))
	copy(out, file.HostNetworks)
	return out, nil
}

func (s *hostNetworkStore) Get(id string) (gamecontent.HostNetwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return gamecontent.HostNetwork{}, err
	}
	for _, network := range file.HostNetworks {
		if network.ID == id {
			return network, nil
		}
	}
	return gamecontent.HostNetwork{}, errHostNetworkNotFound
}

func (s *hostNetworkStore) Create(name string) (gamecontent.HostNetwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.HostNetwork{}, err
	}
	id, err := newUUIDv4()
	if err != nil {
		return gamecontent.HostNetwork{}, err
	}
	network := gamecontent.HostNetwork{ID: id, Name: strings.TrimSpace(name)}
	file.HostNetworks = append(file.HostNetworks, network)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.HostNetwork{}, err
	}
	return network, nil
}

func (s *hostNetworkStore) Update(id, name string) (gamecontent.HostNetwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.HostNetwork{}, err
	}
	for i := range file.HostNetworks {
		if file.HostNetworks[i].ID != id {
			continue
		}
		file.HostNetworks[i].Name = strings.TrimSpace(name)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.HostNetwork{}, err
		}
		return file.HostNetworks[i], nil
	}
	return gamecontent.HostNetwork{}, errHostNetworkNotFound
}

func (s *hostNetworkStore) Delete(id string) (gamecontent.HostNetwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.HostNetwork{}, err
	}
	for i, network := range file.HostNetworks {
		if network.ID != id {
			continue
		}
		file.HostNetworks = append(file.HostNetworks[:i], file.HostNetworks[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.HostNetwork{}, err
		}
		return network, nil
	}
	return gamecontent.HostNetwork{}, errHostNetworkNotFound
}

func (s *hostNetworkStore) loadLocked() (gamecontent.HostNetworksFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyHostNetworks(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.HostNetworksFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeHostNetworks(data)
	if err != nil {
		return gamecontent.HostNetworksFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *hostNetworkStore) writeLocked(file gamecontent.HostNetworksFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeHostNetworks(file)
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
			return fmt.Errorf("back up host networks: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write host networks: %w", err)
	}
	return nil
}
