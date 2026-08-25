package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const RoomPlacementsVersion = 1

type RoomPlacement struct {
	RoomID     string `json:"room_id"`
	LocationID string `json:"location_id"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Z          int    `json:"z"`
	W          int    `json:"w"`
}

type RoomPlacementsFile struct {
	Version    int             `json:"version"`
	Placements []RoomPlacement `json:"placements"`
}

func EmptyRoomPlacements() RoomPlacementsFile {
	return RoomPlacementsFile{Version: RoomPlacementsVersion, Placements: []RoomPlacement{}}
}

func DecodeRoomPlacements(data []byte) (RoomPlacementsFile, error) {
	var file RoomPlacementsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return RoomPlacementsFile{}, fmt.Errorf("room placements: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return RoomPlacementsFile{}, fmt.Errorf("room placements: %w", err)
	}
	if file.Placements == nil {
		return RoomPlacementsFile{}, fmt.Errorf("room placements: placements must be an array")
	}
	if err := ValidateRoomPlacements(file); err != nil {
		return RoomPlacementsFile{}, err
	}
	return file, nil
}

func EncodeRoomPlacements(file RoomPlacementsFile) ([]byte, error) {
	if file.Placements == nil {
		file.Placements = []RoomPlacement{}
	}
	if err := ValidateRoomPlacements(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("room placements: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateRoomPlacements(file RoomPlacementsFile) error {
	if file.Version != RoomPlacementsVersion {
		return fmt.Errorf("room placements: file version %d, want %d", file.Version, RoomPlacementsVersion)
	}
	rooms := make(map[string]struct{}, len(file.Placements))
	cells := make(map[string]string, len(file.Placements))
	for i, placement := range file.Placements {
		where := fmt.Sprintf("room placements: placement %d", i+1)
		if !validUUID(placement.RoomID) {
			return fmt.Errorf("%s has invalid room UUID %q", where, placement.RoomID)
		}
		if !validUUID(placement.LocationID) {
			return fmt.Errorf("%s has invalid location UUID %q", where, placement.LocationID)
		}
		if placement.X < 0 || placement.Y < 0 {
			return fmt.Errorf("%s has negative x or y coordinate", where)
		}
		if placement.Z != 0 || placement.W != 0 {
			return fmt.Errorf("%s must use z=0 and w=0 in version 1", where)
		}
		if _, exists := rooms[placement.RoomID]; exists {
			return fmt.Errorf("%s duplicates room UUID %q", where, placement.RoomID)
		}
		rooms[placement.RoomID] = struct{}{}
		cell := fmt.Sprintf("%s:%d,%d,%d,%d", placement.LocationID, placement.X, placement.Y, placement.Z, placement.W)
		if other, exists := cells[cell]; exists {
			return fmt.Errorf("%s occupies the same cell as room %s", where, other)
		}
		cells[cell] = placement.RoomID
	}
	return nil
}
