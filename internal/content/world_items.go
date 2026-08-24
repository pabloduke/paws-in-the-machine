// Package content defines versioned, data-only authored content formats.
// It has no dependency on the game engine so the editor and game can share
// schemas without sharing filesystem behavior.
package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const WorldItemsVersion = 1

type WorldItemKind string

const (
	WorldItemTakeable WorldItemKind = "takeable"
	WorldItemFixed    WorldItemKind = "fixed"
	WorldItemScenery  WorldItemKind = "scenery"
)

func (k WorldItemKind) Valid() bool {
	switch k {
	case WorldItemTakeable, WorldItemFixed, WorldItemScenery:
		return true
	default:
		return false
	}
}

type WorldItem struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Kind             WorldItemKind `json:"kind"`
	ShortDescription string        `json:"short_description"`
	FullDescription  string        `json:"full_description"`
}

type WorldItemsFile struct {
	Version    int         `json:"version"`
	WorldItems []WorldItem `json:"world_items"`
}

func EmptyWorldItems() WorldItemsFile {
	return WorldItemsFile{Version: WorldItemsVersion, WorldItems: []WorldItem{}}
}

func DecodeWorldItems(data []byte) (WorldItemsFile, error) {
	var file WorldItemsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return WorldItemsFile{}, fmt.Errorf("world items: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return WorldItemsFile{}, fmt.Errorf("world items: %w", err)
	}
	if file.WorldItems == nil {
		return WorldItemsFile{}, fmt.Errorf("world items: world_items must be an array")
	}
	if err := ValidateWorldItems(file); err != nil {
		return WorldItemsFile{}, err
	}
	return file, nil
}

func EncodeWorldItems(file WorldItemsFile) ([]byte, error) {
	if file.WorldItems == nil {
		file.WorldItems = []WorldItem{}
	}
	if err := ValidateWorldItems(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("world items: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateWorldItems(file WorldItemsFile) error {
	if file.Version != WorldItemsVersion {
		return fmt.Errorf("world items: file version %d, want %d", file.Version, WorldItemsVersion)
	}
	seen := make(map[string]struct{}, len(file.WorldItems))
	for i, item := range file.WorldItems {
		where := fmt.Sprintf("world items: item %d", i+1)
		if !validUUID(item.ID) {
			return fmt.Errorf("%s has invalid UUID %q", where, item.ID)
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("%s duplicates UUID %q", where, item.ID)
		}
		seen[item.ID] = struct{}{}
		if strings.TrimSpace(item.Name) == "" {
			return fmt.Errorf("%s has an empty name", where)
		}
		if !item.Kind.Valid() {
			return fmt.Errorf("%s has invalid kind %q", where, item.Kind)
		}
		if strings.TrimSpace(item.ShortDescription) == "" {
			return fmt.Errorf("%s has an empty short description", where)
		}
		if strings.TrimSpace(item.FullDescription) == "" {
			return fmt.Errorf("%s has an empty full description", where)
		}
	}
	return nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("trailing data: %w", err)
	}
	return fmt.Errorf("multiple JSON values")
}

func validUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := 0; i < len(id); i++ {
		switch i {
		case 8, 13, 18, 23:
			if id[i] != '-' {
				return false
			}
		default:
			c := id[i]
			if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') && !('A' <= c && c <= 'F') {
				return false
			}
		}
	}
	return true
}
