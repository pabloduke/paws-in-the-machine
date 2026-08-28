package main

import (
	"errors"
	"path/filepath"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errRoomPlacementNotFound = errors.New("room placement not found")

// Coordinates for a Room inside its assigned Location. A placement must
// agree with the Room's ownership; the web layer enforces that before
// writing here.
type roomPlacementStore struct {
	*relationStore[gamecontent.RoomPlacement]
}

func newRoomPlacementStore(contentDir string) *roomPlacementStore {
	return &roomPlacementStore{&relationStore[gamecontent.RoomPlacement]{
		path:  filepath.Join(contentDir, "room_placements.json"),
		label: "room placements",
		key:   func(p gamecontent.RoomPlacement) string { return p.RoomID },
		decode: func(data []byte) ([]gamecontent.RoomPlacement, error) {
			file, err := gamecontent.DecodeRoomPlacements(data)
			return file.Placements, err
		},
		encode: func(records []gamecontent.RoomPlacement) ([]byte, error) {
			return gamecontent.EncodeRoomPlacements(gamecontent.RoomPlacementsFile{
				Version: gamecontent.RoomPlacementsVersion, Placements: records,
			})
		},
	}}
}

func (s *roomPlacementStore) Place(roomID, locationID string, x, y int) (gamecontent.RoomPlacement, error) {
	return s.Put(gamecontent.RoomPlacement{RoomID: roomID, LocationID: locationID, X: x, Y: y})
}

func (s *roomPlacementStore) Unassign(roomID string) (gamecontent.RoomPlacement, error) {
	placement, err := s.Delete(roomID)
	if errors.Is(err, errRelationNotFound) {
		return placement, errRoomPlacementNotFound
	}
	return placement, err
}
