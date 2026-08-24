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

var errUserNotFound = errors.New("user not found")

type userInput struct {
	Username string
	Password string
}

func (in userInput) normalized() userInput {
	return userInput{Username: strings.TrimSpace(in.Username), Password: in.Password}
}

type userStore struct {
	mu   sync.Mutex
	path string
}

func newUserStore(contentDir string) *userStore {
	return &userStore{path: filepath.Join(contentDir, "users.json")}
}

func (s *userStore) List() ([]gamecontent.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]gamecontent.User, len(file.Users))
	copy(out, file.Users)
	return out, nil
}

func (s *userStore) Get(id string) (gamecontent.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _, _, err := s.loadLocked()
	if err != nil {
		return gamecontent.User{}, err
	}
	for _, user := range file.Users {
		if user.ID == id {
			return user, nil
		}
	}
	return gamecontent.User{}, errUserNotFound
}

func (s *userStore) Create(input userInput) (gamecontent.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.User{}, err
	}
	id, err := newUUIDv4()
	if err != nil {
		return gamecontent.User{}, err
	}
	input = input.normalized()
	user := gamecontent.User{ID: id, Username: input.Username, Password: input.Password}
	file.Users = append(file.Users, user)
	if err := s.writeLocked(file, old, mode); err != nil {
		return gamecontent.User{}, err
	}
	return user, nil
}

func (s *userStore) Update(id string, input userInput) (gamecontent.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.User{}, err
	}
	input = input.normalized()
	for i := range file.Users {
		if file.Users[i].ID != id {
			continue
		}
		file.Users[i].Username = input.Username
		file.Users[i].Password = input.Password
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.User{}, err
		}
		return file.Users[i], nil
	}
	return gamecontent.User{}, errUserNotFound
}

func (s *userStore) Delete(id string) (gamecontent.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, old, mode, err := s.loadLocked()
	if err != nil {
		return gamecontent.User{}, err
	}
	for i, user := range file.Users {
		if user.ID != id {
			continue
		}
		file.Users = append(file.Users[:i], file.Users[i+1:]...)
		if err := s.writeLocked(file, old, mode); err != nil {
			return gamecontent.User{}, err
		}
		return user, nil
	}
	return gamecontent.User{}, errUserNotFound
}

func (s *userStore) loadLocked() (gamecontent.UsersFile, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return gamecontent.EmptyUsers(), nil, 0o644, nil
	}
	if err != nil {
		return gamecontent.UsersFile{}, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	file, err := gamecontent.DecodeUsers(data)
	if err != nil {
		return gamecontent.UsersFile{}, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return file, data, mode, nil
}

func (s *userStore) writeLocked(file gamecontent.UsersFile, old []byte, mode os.FileMode) error {
	data, err := gamecontent.EncodeUsers(file)
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
			return fmt.Errorf("back up users: %w", err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write users: %w", err)
	}
	return nil
}
