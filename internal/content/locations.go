package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const LocationsVersion = 1

type Location struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type LocationsFile struct {
	Version   int        `json:"version"`
	Locations []Location `json:"locations"`
}

func EmptyLocations() LocationsFile {
	return LocationsFile{Version: LocationsVersion, Locations: []Location{}}
}

func DecodeLocations(data []byte) (LocationsFile, error) {
	var file LocationsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return LocationsFile{}, fmt.Errorf("locations: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return LocationsFile{}, fmt.Errorf("locations: %w", err)
	}
	if file.Locations == nil {
		return LocationsFile{}, fmt.Errorf("locations: locations must be an array")
	}
	if err := ValidateLocations(file); err != nil {
		return LocationsFile{}, err
	}
	return file, nil
}

func EncodeLocations(file LocationsFile) ([]byte, error) {
	if file.Locations == nil {
		file.Locations = []Location{}
	}
	if err := ValidateLocations(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("locations: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateLocations(file LocationsFile) error {
	if file.Version != LocationsVersion {
		return fmt.Errorf("locations: file version %d, want %d", file.Version, LocationsVersion)
	}
	seen := make(map[string]struct{}, len(file.Locations))
	for i, location := range file.Locations {
		if err := validateDescribedEntity(fmt.Sprintf("locations: location %d", i+1), location.ID, location.Name, location.Description, seen); err != nil {
			return err
		}
	}
	return nil
}
