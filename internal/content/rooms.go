package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const RoomsVersion = 1

type Room struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoomsFile struct {
	Version int    `json:"version"`
	Rooms   []Room `json:"rooms"`
}

func EmptyRooms() RoomsFile { return RoomsFile{Version: RoomsVersion, Rooms: []Room{}} }

func DecodeRooms(data []byte) (RoomsFile, error) {
	var file RoomsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return RoomsFile{}, fmt.Errorf("rooms: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return RoomsFile{}, fmt.Errorf("rooms: %w", err)
	}
	if file.Rooms == nil {
		return RoomsFile{}, fmt.Errorf("rooms: rooms must be an array")
	}
	if err := ValidateRooms(file); err != nil {
		return RoomsFile{}, err
	}
	return file, nil
}

func EncodeRooms(file RoomsFile) ([]byte, error) {
	if file.Rooms == nil {
		file.Rooms = []Room{}
	}
	if err := ValidateRooms(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("rooms: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateRooms(file RoomsFile) error {
	if file.Version != RoomsVersion {
		return fmt.Errorf("rooms: file version %d, want %d", file.Version, RoomsVersion)
	}
	seen := make(map[string]struct{}, len(file.Rooms))
	for i, room := range file.Rooms {
		where := fmt.Sprintf("rooms: room %d", i+1)
		if err := validateDescribedEntity(where, room.ID, room.Name, room.Description, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateDescribedEntity(where, id, name, description string, seen map[string]struct{}) error {
	if !validUUID(id) {
		return fmt.Errorf("%s has invalid UUID %q", where, id)
	}
	if _, exists := seen[id]; exists {
		return fmt.Errorf("%s duplicates UUID %q", where, id)
	}
	seen[id] = struct{}{}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s has an empty name", where)
	}
	if strings.TrimSpace(description) == "" {
		return fmt.Errorf("%s has an empty description", where)
	}
	return nil
}
