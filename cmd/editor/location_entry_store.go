package main

import (
	"errors"
	"path/filepath"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errLocationEntryNotFound = errors.New("location entry not found")

// The optional Room a Location descends into. Keyed by Location, so a
// Location has at most one entry Room.
type locationEntryStore struct {
	*relationStore[gamecontent.LocationEntry]
}

func newLocationEntryStore(contentDir string) *locationEntryStore {
	return &locationEntryStore{&relationStore[gamecontent.LocationEntry]{
		path:  filepath.Join(contentDir, "location_entry_rooms.json"),
		label: "location entries",
		key:   func(e gamecontent.LocationEntry) string { return e.LocationID },
		decode: func(data []byte) ([]gamecontent.LocationEntry, error) {
			file, err := gamecontent.DecodeLocationEntries(data)
			return file.Entries, err
		},
		encode: func(records []gamecontent.LocationEntry) ([]byte, error) {
			return gamecontent.EncodeLocationEntries(gamecontent.LocationEntriesFile{
				Version: gamecontent.LocationEntriesVersion, Entries: records,
			})
		},
	}}
}

func (s *locationEntryStore) Set(locationID, roomID string) (gamecontent.LocationEntry, error) {
	return s.Put(gamecontent.LocationEntry{LocationID: locationID, RoomID: roomID})
}

func (s *locationEntryStore) Clear(locationID string) (gamecontent.LocationEntry, error) {
	entry, err := s.Delete(locationID)
	if errors.Is(err, errRelationNotFound) {
		return entry, errLocationEntryNotFound
	}
	return entry, err
}
