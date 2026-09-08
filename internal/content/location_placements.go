package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const LocationPlacementsVersion = 2

type LocationPlacement struct {
	LocationID string `json:"location_id"`
	HubID      string `json:"hub_id"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Z          int    `json:"z"`
	W          int    `json:"w"`
}

type LocationPlacementsFile struct {
	Version    int                 `json:"version"`
	Placements []LocationPlacement `json:"placements"`
}

func EmptyLocationPlacements() LocationPlacementsFile {
	return LocationPlacementsFile{Version: LocationPlacementsVersion, Placements: []LocationPlacement{}}
}

func DecodeLocationPlacements(data []byte) (LocationPlacementsFile, error) {
	var file LocationPlacementsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return LocationPlacementsFile{}, fmt.Errorf("location placements: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return LocationPlacementsFile{}, fmt.Errorf("location placements: %w", err)
	}
	if file.Placements == nil {
		return LocationPlacementsFile{}, fmt.Errorf("location placements: placements must be an array")
	}
	if err := ValidateLocationPlacements(file); err != nil {
		return LocationPlacementsFile{}, err
	}
	return file, nil
}

func EncodeLocationPlacements(file LocationPlacementsFile) ([]byte, error) {
	if file.Placements == nil {
		file.Placements = []LocationPlacement{}
	}
	if err := ValidateLocationPlacements(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("location placements: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateLocationPlacements(file LocationPlacementsFile) error {
	if file.Version != 1 && file.Version != LocationPlacementsVersion {
		return fmt.Errorf("location placements: file version %d, want %d", file.Version, LocationPlacementsVersion)
	}
	entities, cells := map[string]struct{}{}, map[string]string{}
	for i, placement := range file.Placements {
		where := fmt.Sprintf("location placements: placement %d", i+1)
		if !validUUID(placement.LocationID) {
			return fmt.Errorf("%s has invalid location UUID %q", where, placement.LocationID)
		}
		if !validUUID(placement.HubID) {
			return fmt.Errorf("%s has invalid hub UUID %q", where, placement.HubID)
		}
		if placement.X < 0 || placement.Y < 0 {
			return fmt.Errorf("%s has negative x or y coordinate", where)
		}
		if (file.Version == 1 && placement.Z != 0) || placement.W != 0 {
			return fmt.Errorf("%s requires w=0 (and z=0 for legacy version 1)", where)
		}
		if _, ok := entities[placement.LocationID]; ok {
			return fmt.Errorf("%s duplicates location UUID %q", where, placement.LocationID)
		}
		entities[placement.LocationID] = struct{}{}
		cell := fmt.Sprintf("%s:%d,%d,%d", placement.HubID, placement.X, placement.Y, placement.Z)
		if other, ok := cells[cell]; ok {
			return fmt.Errorf("%s occupies the same cell as location %s", where, other)
		}
		cells[cell] = placement.LocationID
	}
	return nil
}
