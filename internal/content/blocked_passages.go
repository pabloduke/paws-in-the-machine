package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// A blocked passage removes both directions of an otherwise automatic
// horizontal adjacency. Endpoint IDs keep the authored wall explicit.
type BlockedPassage struct {
	Kind string `json:"kind"`
	AID  string `json:"a_id"`
	BID  string `json:"b_id"`
}
type BlockedPassagesFile struct {
	Version  int              `json:"version"`
	Passages []BlockedPassage `json:"passages"`
}

func EmptyBlockedPassages() BlockedPassagesFile {
	return BlockedPassagesFile{Version: 1, Passages: []BlockedPassage{}}
}
func PassageKey(v BlockedPassage) string {
	a, b := v.AID, v.BID
	if a > b {
		a, b = b, a
	}
	return v.Kind + ":" + a + ":" + b
}
func DecodeBlockedPassages(data []byte) (BlockedPassagesFile, error) {
	var f BlockedPassagesFile
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&f); err != nil {
		return f, err
	}
	if err := rejectTrailingJSON(d); err != nil {
		return f, err
	}
	return f, ValidateBlockedPassages(f)
}
func ValidateBlockedPassages(f BlockedPassagesFile) error {
	if f.Version != 1 || f.Passages == nil {
		return fmt.Errorf("invalid blocked passages file")
	}
	seen := map[string]bool{}
	for _, v := range f.Passages {
		if (v.Kind != "location" && v.Kind != "room") || !validUUID(v.AID) || !validUUID(v.BID) || v.AID == v.BID {
			return fmt.Errorf("invalid blocked passage")
		}
		key := PassageKey(v)
		if seen[key] {
			return fmt.Errorf("duplicate blocked passage")
		}
		seen[key] = true
	}
	return nil
}
func EncodeBlockedPassages(f BlockedPassagesFile) ([]byte, error) {
	if err := ValidateBlockedPassages(f); err != nil {
		return nil, err
	}
	return json.MarshalIndent(f, "", "  ")
}

// HorizontalNeighbors returns the compass direction only for adjoining cells
// within the same parent and floor. It cannot create long-distance exits.
func (c Catalogs) HorizontalNeighbors(kind, aID, bID string) (string, bool) {
	type at struct {
		parent  string
		x, y, z int
	}
	cells := map[string]at{}
	switch kind {
	case "location":
		for _, p := range c.LocationPlacements.Placements {
			cells[p.LocationID] = at{p.HubID, p.X, p.Y, p.Z}
		}
	case "room":
		for _, p := range c.RoomPlacements.Placements {
			cells[p.RoomID] = at{p.LocationID, p.X, p.Y, p.Z}
		}
	default:
		return "", false
	}
	a, okA := cells[aID]
	b, okB := cells[bID]
	if !okA || !okB || a.parent != b.parent || a.z != b.z {
		return "", false
	}
	switch {
	case b.x == a.x && b.y-a.y == 1:
		return "north", true
	case b.x == a.x && b.y-a.y == -1:
		return "south", true
	case b.y == a.y && b.x-a.x == 1:
		return "east", true
	case b.y == a.y && b.x-a.x == -1:
		return "west", true
	}
	return "", false
}
func ValidateBlockedGeometry(c Catalogs) error {
	if c.BlockedPassages.Version == 0 && c.BlockedPassages.Passages == nil {
		return nil
	}
	if err := ValidateBlockedPassages(c.BlockedPassages); err != nil {
		return err
	}
	for _, v := range c.BlockedPassages.Passages {
		if _, ok := c.HorizontalNeighbors(v.Kind, v.AID, v.BID); !ok {
			return fmt.Errorf("blocked passage %s to %s requires adjoining cells in the same parent and floor", v.AID, v.BID)
		}
	}
	return nil
}
