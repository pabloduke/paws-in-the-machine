package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const LocationAssignmentsVersion = 1
const RoomAssignmentsVersion = 1

type LocationAssignment struct {
	LocationID string `json:"location_id"`
	HubID      string `json:"hub_id"`
}

type LocationAssignmentsFile struct {
	Version     int                  `json:"version"`
	Assignments []LocationAssignment `json:"assignments"`
}

func EmptyLocationAssignments() LocationAssignmentsFile {
	return LocationAssignmentsFile{Version: LocationAssignmentsVersion, Assignments: []LocationAssignment{}}
}

func DecodeLocationAssignments(data []byte) (LocationAssignmentsFile, error) {
	var file LocationAssignmentsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return LocationAssignmentsFile{}, fmt.Errorf("location assignments: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return LocationAssignmentsFile{}, fmt.Errorf("location assignments: %w", err)
	}
	if file.Assignments == nil {
		return LocationAssignmentsFile{}, fmt.Errorf("location assignments: assignments must be an array")
	}
	if err := ValidateLocationAssignments(file); err != nil {
		return LocationAssignmentsFile{}, err
	}
	return file, nil
}

func EncodeLocationAssignments(file LocationAssignmentsFile) ([]byte, error) {
	if file.Assignments == nil {
		file.Assignments = []LocationAssignment{}
	}
	if err := ValidateLocationAssignments(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("location assignments: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateLocationAssignments(file LocationAssignmentsFile) error {
	if file.Version != LocationAssignmentsVersion {
		return fmt.Errorf("location assignments: file version %d, want %d", file.Version, LocationAssignmentsVersion)
	}
	seen := map[string]struct{}{}
	for i, assignment := range file.Assignments {
		where := fmt.Sprintf("location assignments: assignment %d", i+1)
		if !validUUID(assignment.LocationID) {
			return fmt.Errorf("%s has invalid location UUID %q", where, assignment.LocationID)
		}
		if !validUUID(assignment.HubID) {
			return fmt.Errorf("%s has invalid hub UUID %q", where, assignment.HubID)
		}
		if _, ok := seen[assignment.LocationID]; ok {
			return fmt.Errorf("%s duplicates location UUID %q", where, assignment.LocationID)
		}
		seen[assignment.LocationID] = struct{}{}
	}
	return nil
}

type RoomAssignment struct {
	RoomID     string `json:"room_id"`
	LocationID string `json:"location_id"`
}

type RoomAssignmentsFile struct {
	Version     int              `json:"version"`
	Assignments []RoomAssignment `json:"assignments"`
}

func EmptyRoomAssignments() RoomAssignmentsFile {
	return RoomAssignmentsFile{Version: RoomAssignmentsVersion, Assignments: []RoomAssignment{}}
}

func DecodeRoomAssignments(data []byte) (RoomAssignmentsFile, error) {
	var file RoomAssignmentsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return RoomAssignmentsFile{}, fmt.Errorf("room assignments: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return RoomAssignmentsFile{}, fmt.Errorf("room assignments: %w", err)
	}
	if file.Assignments == nil {
		return RoomAssignmentsFile{}, fmt.Errorf("room assignments: assignments must be an array")
	}
	if err := ValidateRoomAssignments(file); err != nil {
		return RoomAssignmentsFile{}, err
	}
	return file, nil
}

func EncodeRoomAssignments(file RoomAssignmentsFile) ([]byte, error) {
	if file.Assignments == nil {
		file.Assignments = []RoomAssignment{}
	}
	if err := ValidateRoomAssignments(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("room assignments: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateRoomAssignments(file RoomAssignmentsFile) error {
	if file.Version != RoomAssignmentsVersion {
		return fmt.Errorf("room assignments: file version %d, want %d", file.Version, RoomAssignmentsVersion)
	}
	seen := map[string]struct{}{}
	for i, assignment := range file.Assignments {
		where := fmt.Sprintf("room assignments: assignment %d", i+1)
		if !validUUID(assignment.RoomID) {
			return fmt.Errorf("%s has invalid room UUID %q", where, assignment.RoomID)
		}
		if !validUUID(assignment.LocationID) {
			return fmt.Errorf("%s has invalid location UUID %q", where, assignment.LocationID)
		}
		if _, ok := seen[assignment.RoomID]; ok {
			return fmt.Errorf("%s duplicates room UUID %q", where, assignment.RoomID)
		}
		seen[assignment.RoomID] = struct{}{}
	}
	return nil
}
