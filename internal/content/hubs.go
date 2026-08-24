package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const HubsVersion = 1

type Hub struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HubsFile struct {
	Version int   `json:"version"`
	Hubs    []Hub `json:"hubs"`
}

func EmptyHubs() HubsFile {
	return HubsFile{Version: HubsVersion, Hubs: []Hub{}}
}

func DecodeHubs(data []byte) (HubsFile, error) {
	var file HubsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return HubsFile{}, fmt.Errorf("hubs: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return HubsFile{}, fmt.Errorf("hubs: %w", err)
	}
	if file.Hubs == nil {
		return HubsFile{}, fmt.Errorf("hubs: hubs must be an array")
	}
	if err := ValidateHubs(file); err != nil {
		return HubsFile{}, err
	}
	return file, nil
}

func EncodeHubs(file HubsFile) ([]byte, error) {
	if file.Hubs == nil {
		file.Hubs = []Hub{}
	}
	if err := ValidateHubs(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("hubs: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateHubs(file HubsFile) error {
	if file.Version != HubsVersion {
		return fmt.Errorf("hubs: file version %d, want %d", file.Version, HubsVersion)
	}
	seen := make(map[string]struct{}, len(file.Hubs))
	for i, hub := range file.Hubs {
		where := fmt.Sprintf("hubs: hub %d", i+1)
		if !validUUID(hub.ID) {
			return fmt.Errorf("%s has invalid UUID %q", where, hub.ID)
		}
		if _, exists := seen[hub.ID]; exists {
			return fmt.Errorf("%s duplicates UUID %q", where, hub.ID)
		}
		seen[hub.ID] = struct{}{}
		if strings.TrimSpace(hub.Name) == "" {
			return fmt.Errorf("%s has an empty name", where)
		}
	}
	return nil
}
