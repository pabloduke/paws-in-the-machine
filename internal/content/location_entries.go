package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const LocationEntriesVersion = 1

type LocationEntry struct {
	LocationID string `json:"location_id"`
	RoomID     string `json:"room_id"`
}

type LocationEntriesFile struct {
	Version int             `json:"version"`
	Entries []LocationEntry `json:"entries"`
}

func EmptyLocationEntries() LocationEntriesFile {
	return LocationEntriesFile{Version: LocationEntriesVersion, Entries: []LocationEntry{}}
}

func DecodeLocationEntries(data []byte) (LocationEntriesFile, error) {
	var file LocationEntriesFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return LocationEntriesFile{}, fmt.Errorf("location entries: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return LocationEntriesFile{}, fmt.Errorf("location entries: %w", err)
	}
	if file.Entries == nil {
		return LocationEntriesFile{}, fmt.Errorf("location entries: entries must be an array")
	}
	if err := ValidateLocationEntries(file); err != nil {
		return LocationEntriesFile{}, err
	}
	return file, nil
}

func EncodeLocationEntries(file LocationEntriesFile) ([]byte, error) {
	if file.Entries == nil {
		file.Entries = []LocationEntry{}
	}
	if err := ValidateLocationEntries(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("location entries: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateLocationEntries(file LocationEntriesFile) error {
	if file.Version != LocationEntriesVersion {
		return fmt.Errorf("location entries: file version %d, want %d", file.Version, LocationEntriesVersion)
	}
	seen := map[string]struct{}{}
	for i, entry := range file.Entries {
		where := fmt.Sprintf("location entries: entry %d", i+1)
		if !validUUID(entry.LocationID) {
			return fmt.Errorf("%s has invalid location UUID %q", where, entry.LocationID)
		}
		if !validUUID(entry.RoomID) {
			return fmt.Errorf("%s has invalid room UUID %q", where, entry.RoomID)
		}
		if _, ok := seen[entry.LocationID]; ok {
			return fmt.Errorf("%s duplicates location UUID %q", where, entry.LocationID)
		}
		seen[entry.LocationID] = struct{}{}
	}
	return nil
}
