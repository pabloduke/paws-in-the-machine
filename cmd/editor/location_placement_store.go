package main

import (
	"errors"
	"path/filepath"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

var errLocationPlacementNotFound = errors.New("location placement not found")

// Coordinates for a Location inside its assigned Hub.
type locationPlacementStore struct {
	*relationStore[gamecontent.LocationPlacement]
}

func newLocationPlacementStore(contentDir string) *locationPlacementStore {
	return &locationPlacementStore{&relationStore[gamecontent.LocationPlacement]{
		path:  filepath.Join(contentDir, "location_placements.json"),
		label: "location placements",
		key:   func(p gamecontent.LocationPlacement) string { return p.LocationID },
		decode: func(data []byte) ([]gamecontent.LocationPlacement, error) {
			file, err := gamecontent.DecodeLocationPlacements(data)
			return file.Placements, err
		},
		encode: func(records []gamecontent.LocationPlacement) ([]byte, error) {
			return gamecontent.EncodeLocationPlacements(gamecontent.LocationPlacementsFile{
				Version: gamecontent.LocationPlacementsVersion, Placements: records,
			})
		},
	}}
}

func (s *locationPlacementStore) Place(locationID, hubID string, x, y int) (gamecontent.LocationPlacement, error) {
	return s.Put(gamecontent.LocationPlacement{LocationID: locationID, HubID: hubID, X: x, Y: y})
}

func (s *locationPlacementStore) Unassign(locationID string) (gamecontent.LocationPlacement, error) {
	placement, err := s.Delete(locationID)
	if errors.Is(err, errRelationNotFound) {
		return placement, errLocationPlacementNotFound
	}
	return placement, err
}

func (s *locationPlacementStore) PlaceAt(entityID, parentID string, x, y, z int) (gamecontent.LocationPlacement, error) {
	return s.Put(gamecontent.LocationPlacement{LocationID: entityID, HubID: parentID, X: x, Y: y, Z: z})
}
