package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errWorldItemNotFound = errors.New("world item not found")

type worldItemInput struct {
	Name             string
	Kind             gamecontent.WorldItemKind
	ShortDescription string
	FullDescription  string
}

func (in worldItemInput) normalized() worldItemInput {
	return worldItemInput{
		Name:             strings.TrimSpace(in.Name),
		Kind:             gamecontent.WorldItemKind(strings.TrimSpace(string(in.Kind))),
		ShortDescription: strings.TrimSpace(strings.ReplaceAll(in.ShortDescription, "\r\n", "\n")),
		FullDescription:  strings.TrimSpace(strings.ReplaceAll(in.FullDescription, "\r\n", "\n")),
	}
}

type worldItemStore struct {
	mu   sync.Mutex
	path string
}

func newWorldItemStore(contentDir string) *worldItemStore {
	return &worldItemStore{path: filepath.Join(contentDir, "world_items.json")}
}

func resolveContentDir(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("find repository: %w", err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil && !info.IsDir() {
			return filepath.Join(dir, "internal", "game", "content"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("find repository: no go.mod above %s; use -content-dir", dir)
		}
		dir = parent
	}
}

func (s *worldItemStore) List() ([]gamecontent.WorldItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.WorldItem, len(file.WorldItems))
	copy(out, file.WorldItems)
	return out, nil
}

func (s *worldItemStore) Get(id string) (gamecontent.WorldItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return gamecontent.WorldItem{}, err
	}
	for _, item := range file.WorldItems {
		if item.ID == id {
			return item, nil
		}
	}
	return gamecontent.WorldItem{}, errWorldItemNotFound
}

func (s *worldItemStore) Create(input worldItemInput) (gamecontent.WorldItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.WorldItem{}, err
	}
	id, err := newUUIDv4()
	if err != nil {
		return gamecontent.WorldItem{}, err
	}
	input = input.normalized()
	item := gamecontent.WorldItem{
		ID:               id,
		Name:             input.Name,
		Kind:             input.Kind,
		ShortDescription: input.ShortDescription,
		FullDescription:  input.FullDescription,
	}
	file.WorldItems = append(file.WorldItems, item)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.WorldItem{}, err
	}
	return item, nil
}

func (s *worldItemStore) Update(id string, input worldItemInput) (gamecontent.WorldItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.WorldItem{}, err
	}
	input = input.normalized()
	for i := range file.WorldItems {
		if file.WorldItems[i].ID != id {
			continue
		}
		file.WorldItems[i].Name = input.Name
		file.WorldItems[i].Kind = input.Kind
		file.WorldItems[i].ShortDescription = input.ShortDescription
		file.WorldItems[i].FullDescription = input.FullDescription
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.WorldItem{}, err
		}
		return file.WorldItems[i], nil
	}
	return gamecontent.WorldItem{}, errWorldItemNotFound
}

func (s *worldItemStore) Delete(id string) (gamecontent.WorldItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.WorldItem{}, err
	}
	for i, item := range file.WorldItems {
		if item.ID != id {
			continue
		}
		file.WorldItems = append(file.WorldItems[:i], file.WorldItems[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.WorldItem{}, err
		}
		return item, nil
	}
	return gamecontent.WorldItem{}, errWorldItemNotFound
}

func (s *worldItemStore) loadLocked() (gamecontent.WorldItemsFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyWorldItems(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.WorldItemsFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeWorldItems(data)
	if err != nil {
		return gamecontent.WorldItemsFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *worldItemStore) writeLocked(file gamecontent.WorldItemsFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeWorldItems(file)
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
			return fmt.Errorf("back up world items: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write world items: %w", err)
	}
	return nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		if removeErr := os.Remove(tmpName); err == nil && removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = removeErr
		}
	}()
	if err = tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	if dirHandle, openErr := os.Open(dir); openErr == nil {
		syncErr := dirHandle.Sync()
		closeErr := dirHandle.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func newUUIDv4() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}
