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

var errDescribedEntityNotFound = errors.New("described entity not found")

type describedFields struct {
	ID          string
	Name        string
	Description string
}

type describedInput struct {
	Name        string
	Description string
}

func (in describedInput) normalized() describedInput {
	return describedInput{
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(strings.ReplaceAll(in.Description, "\r\n", "\n")),
	}
}

type describedStore[T any] struct {
	mu       sync.Mutex
	path     string
	label    string
	empty    func() []T
	decode   func([]byte) ([]T, error)
	encode   func([]T) ([]byte, error)
	fields   func(T) describedFields
	makeItem func(describedFields) T
}

type roomStore struct {
	*describedStore[gamecontent.Room]
}
type npcStore struct {
	*describedStore[gamecontent.NPC]
}

func newRoomStore(contentDir string) *roomStore {
	return &roomStore{&describedStore[gamecontent.Room]{
		path:  filepath.Join(contentDir, "rooms.json"),
		label: "rooms",
		empty: func() []gamecontent.Room { return []gamecontent.Room{} },
		decode: func(data []byte) ([]gamecontent.Room, error) {
			file, err := gamecontent.DecodeRooms(data)
			return file.Rooms, err
		},
		encode: func(records []gamecontent.Room) ([]byte, error) {
			return gamecontent.EncodeRooms(gamecontent.RoomsFile{Version: gamecontent.RoomsVersion, Rooms: records})
		},
		fields: func(room gamecontent.Room) describedFields {
			return describedFields{ID: room.ID, Name: room.Name, Description: room.Description}
		},
		makeItem: func(fields describedFields) gamecontent.Room {
			return gamecontent.Room{ID: fields.ID, Name: fields.Name, Description: fields.Description}
		},
	}}
}

func newNPCStore(contentDir string) *npcStore {
	return &npcStore{&describedStore[gamecontent.NPC]{
		path:  filepath.Join(contentDir, "npcs.json"),
		label: "npcs",
		empty: func() []gamecontent.NPC { return []gamecontent.NPC{} },
		decode: func(data []byte) ([]gamecontent.NPC, error) {
			file, err := gamecontent.DecodeNPCs(data)
			return file.NPCs, err
		},
		encode: func(records []gamecontent.NPC) ([]byte, error) {
			return gamecontent.EncodeNPCs(gamecontent.NPCsFile{Version: gamecontent.NPCsVersion, NPCs: records})
		},
		fields: func(npc gamecontent.NPC) describedFields {
			return describedFields{ID: npc.ID, Name: npc.Name, Description: npc.Description}
		},
		makeItem: func(fields describedFields) gamecontent.NPC {
			return gamecontent.NPC{ID: fields.ID, Name: fields.Name, Description: fields.Description}
		},
	}}
}

func (s *describedStore[T]) List() ([]T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, _, _, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]T, len(records))
	copy(out, records)
	return out, nil
}

func (s *describedStore[T]) Get(id string) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, _, _, err := s.loadLocked()
	if err != nil {
		var zero T
		return zero, err
	}
	for _, record := range records {
		if s.fields(record).ID == id {
			return record, nil
		}
	}
	var zero T
	return zero, errDescribedEntityNotFound
}

func (s *describedStore[T]) Create(input describedInput) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, old, mode, err := s.loadLocked()
	if err != nil {
		var zero T
		return zero, err
	}
	id, err := newUUIDv4()
	if err != nil {
		var zero T
		return zero, err
	}
	input = input.normalized()
	record := s.makeItem(describedFields{ID: id, Name: input.Name, Description: input.Description})
	records = append(records, record)
	if err := s.writeLocked(records, old, mode); err != nil {
		var zero T
		return zero, err
	}
	return record, nil
}

func (s *describedStore[T]) Update(id string, input describedInput) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, old, mode, err := s.loadLocked()
	if err != nil {
		var zero T
		return zero, err
	}
	input = input.normalized()
	for i, record := range records {
		if s.fields(record).ID != id {
			continue
		}
		records[i] = s.makeItem(describedFields{ID: id, Name: input.Name, Description: input.Description})
		if err := s.writeLocked(records, old, mode); err != nil {
			var zero T
			return zero, err
		}
		return records[i], nil
	}
	var zero T
	return zero, errDescribedEntityNotFound
}

func (s *describedStore[T]) Delete(id string) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, old, mode, err := s.loadLocked()
	if err != nil {
		var zero T
		return zero, err
	}
	for i, record := range records {
		if s.fields(record).ID != id {
			continue
		}
		records = append(records[:i], records[i+1:]...)
		if err := s.writeLocked(records, old, mode); err != nil {
			var zero T
			return zero, err
		}
		return record, nil
	}
	var zero T
	return zero, errDescribedEntityNotFound
}

func (s *describedStore[T]) loadLocked() ([]T, []byte, os.FileMode, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.empty(), nil, 0o644, nil
	}
	if err != nil {
		return nil, nil, 0, fmt.Errorf("read %s: %w", s.path, err)
	}
	records, err := s.decode(data)
	if err != nil {
		return nil, nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(s.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	return records, data, mode, nil
}

func (s *describedStore[T]) writeLocked(records []T, old []byte, mode os.FileMode) error {
	data, err := s.encode(records)
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
			return fmt.Errorf("back up %s: %w", s.label, err)
		}
	}
	if err := writeAtomic(s.path, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", s.label, err)
	}
	return nil
}
