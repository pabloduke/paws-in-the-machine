package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
)

type playSettingsStore struct {
	mu   sync.Mutex
	path string
}

func newPlaySettingsStore(dir string) *playSettingsStore {
	return &playSettingsStore{path: filepath.Join(dir, "play_settings.json")}
}
func (s *playSettingsStore) Load() (content.PlaySettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}
func (s *playSettingsStore) load() (content.PlaySettings, error) {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return content.EmptyPlaySettings(), nil
	}
	if err != nil {
		return content.PlaySettings{}, err
	}
	return content.DecodePlaySettings(b)
}
func (s *playSettingsStore) Update(change func(*content.PlaySettings) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.load()
	if err != nil {
		return err
	}
	if err = change(&settings); err != nil {
		return err
	}
	b, err := content.EncodePlaySettings(settings)
	if err != nil {
		return err
	}
	old, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if bytes.Equal(old, b) {
		return nil
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	mode := os.FileMode(0644)
	if info, err := os.Stat(s.path); err == nil {
		mode = info.Mode().Perm()
	}
	if old != nil {
		if err = writeAtomic(s.path+".bak", old, mode); err != nil {
			return err
		}
	}
	return writeAtomic(s.path, b, mode)
}
