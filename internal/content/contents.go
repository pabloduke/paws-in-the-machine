package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const ContentsVersion = 1

// Kinds of entity that may sit inside a player-standable cell.
const (
	ContentKindWorldItem = "world_item"
	ContentKindNPC       = "npc"
	ContentKindTerminal  = "terminal"
)

// Kinds of cell that may hold contents. Both a Location and an interior
// Room are player-standable (docs/systems/hubs.md), so both may hold
// things.
const (
	ContainerKindLocation = "location"
	ContainerKindRoom     = "room"
)

// Content places one entity inside one cell. Unlike a placement it
// carries no coordinate: a cell's contents are an unordered set, and
// many entities may share one cell.
//
// For a Takeable world item this records where the item starts; the
// runtime owns its position after the player moves it. For an NPC it
// records an unconditional position — flag-conditional presence stays
// with engine.Placed (docs/systems/presence.md).
type Content struct {
	EntityID   string `json:"entity_id"`
	EntityKind string `json:"entity_kind"`
	ParentID   string `json:"parent_id"`
	ParentKind string `json:"parent_kind"`
}

type ContentsFile struct {
	Version  int       `json:"version"`
	Contents []Content `json:"contents"`
}

func EmptyContents() ContentsFile {
	return ContentsFile{Version: ContentsVersion, Contents: []Content{}}
}

func DecodeContents(data []byte) (ContentsFile, error) {
	var file ContentsFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return ContentsFile{}, fmt.Errorf("contents: decode: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return ContentsFile{}, fmt.Errorf("contents: %w", err)
	}
	if file.Contents == nil {
		return ContentsFile{}, fmt.Errorf("contents: contents must be an array")
	}
	if err := ValidateContents(file); err != nil {
		return ContentsFile{}, err
	}
	return file, nil
}

func EncodeContents(file ContentsFile) ([]byte, error) {
	if file.Contents == nil {
		file.Contents = []Content{}
	}
	if err := ValidateContents(file); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("contents: encode: %w", err)
	}
	return append(data, '\n'), nil
}

func ValidateContents(file ContentsFile) error {
	if file.Version != ContentsVersion {
		return fmt.Errorf("contents: file version %d, want %d", file.Version, ContentsVersion)
	}
	seen := map[string]struct{}{}
	for i, record := range file.Contents {
		where := fmt.Sprintf("contents: record %d", i+1)
		if !validContentKind(record.EntityKind) {
			return fmt.Errorf("%s has unknown entity kind %q", where, record.EntityKind)
		}
		if !validContainerKind(record.ParentKind) {
			return fmt.Errorf("%s has unknown parent kind %q", where, record.ParentKind)
		}
		if !validUUID(record.EntityID) {
			return fmt.Errorf("%s has invalid entity UUID %q", where, record.EntityID)
		}
		if !validUUID(record.ParentID) {
			return fmt.Errorf("%s has invalid parent UUID %q", where, record.ParentID)
		}
		// An entity is in at most one cell. The kind is part of the key
		// because UUIDs are only unique within their own catalog.
		key := record.EntityKind + ":" + record.EntityID
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%s duplicates %s %q", where, record.EntityKind, record.EntityID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validContentKind(kind string) bool {
	switch kind {
	case ContentKindWorldItem, ContentKindNPC, ContentKindTerminal:
		return true
	}
	return false
}

func validContainerKind(kind string) bool {
	switch kind {
	case ContainerKindLocation, ContainerKindRoom:
		return true
	}
	return false
}
