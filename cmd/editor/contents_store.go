package main

import (
	"errors"
	"path/filepath"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errContentNotFound = errors.New("cell content not found")

// What sits inside a player-standable cell. Unlike containment and
// placement this relation is many-per-parent, so its key is the
// entity, not the cell.
type contentsStore struct {
	*relationStore[gamecontent.Content]
}

func contentKey(kind, id string) string { return kind + ":" + id }

func newContentsStore(contentDir string) *contentsStore {
	return &contentsStore{&relationStore[gamecontent.Content]{
		path:  filepath.Join(contentDir, "contents.json"),
		label: "contents",
		key:   func(c gamecontent.Content) string { return contentKey(c.EntityKind, c.EntityID) },
		decode: func(data []byte) ([]gamecontent.Content, error) {
			file, err := gamecontent.DecodeContents(data)
			return file.Contents, err
		},
		encode: func(records []gamecontent.Content) ([]byte, error) {
			return gamecontent.EncodeContents(gamecontent.ContentsFile{
				Version: gamecontent.ContentsVersion, Contents: records,
			})
		},
	}}
}

// PlaceIn puts the entity in the given cell, moving it out of any cell
// it currently occupies.
func (s *contentsStore) PlaceIn(entityKind, entityID, parentKind, parentID string) (gamecontent.Content, error) {
	return s.Put(gamecontent.Content{
		EntityID: entityID, EntityKind: entityKind,
		ParentID: parentID, ParentKind: parentKind,
	})
}

func (s *contentsStore) Remove(entityKind, entityID string) (gamecontent.Content, error) {
	record, err := s.Delete(contentKey(entityKind, entityID))
	if errors.Is(err, errRelationNotFound) {
		return record, errContentNotFound
	}
	return record, err
}

// ContainerOf reports the cell holding the entity, or "" if it is
// nowhere. Being nowhere is a valid resting state, not an error.
func (s *contentsStore) ContainerOf(entityKind, entityID string) (gamecontent.Content, bool, error) {
	record, err := s.Get(contentKey(entityKind, entityID))
	if errors.Is(err, errRelationNotFound) {
		return record, false, nil
	}
	if err != nil {
		return record, false, err
	}
	return record, true, nil
}
