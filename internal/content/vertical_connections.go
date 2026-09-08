package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type VerticalConnection struct {
	Kind    string `json:"kind"` // location or room; both endpoints share a parent
	LowerID string `json:"lower_id"`
	UpperID string `json:"upper_id"`
}
type VerticalConnectionsFile struct {
	Version     int                  `json:"version"`
	Connections []VerticalConnection `json:"connections"`
}

func EmptyVerticalConnections() VerticalConnectionsFile {
	return VerticalConnectionsFile{Version: 1, Connections: []VerticalConnection{}}
}
func DecodeVerticalConnections(data []byte) (VerticalConnectionsFile, error) {
	var f VerticalConnectionsFile
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&f); err != nil {
		return f, err
	}
	if err := rejectTrailingJSON(d); err != nil {
		return f, err
	}
	return f, ValidateVerticalConnections(f)
}
func ValidateVerticalConnections(f VerticalConnectionsFile) error {
	if f.Version != 1 || f.Connections == nil {
		return fmt.Errorf("invalid vertical connections file")
	}
	seen := map[string]bool{}
	for _, v := range f.Connections {
		if (v.Kind != "location" && v.Kind != "room") || !validUUID(v.LowerID) || !validUUID(v.UpperID) || v.LowerID == v.UpperID {
			return fmt.Errorf("invalid vertical connection")
		}
		for _, key := range []string{v.Kind + ":up:" + v.LowerID, v.Kind + ":down:" + v.UpperID} {
			if seen[key] {
				return fmt.Errorf("duplicate vertical direction")
			}
			seen[key] = true
		}
	}
	return nil
}
func EncodeVerticalConnections(f VerticalConnectionsFile) ([]byte, error) {
	if err := ValidateVerticalConnections(f); err != nil {
		return nil, err
	}
	return json.MarshalIndent(f, "", "  ")
}
func ValidateVerticalGeometry(c Catalogs) error {
	if c.VerticalConnections.Version == 0 && c.VerticalConnections.Connections == nil {
		return nil
	}
	if err := ValidateVerticalConnections(c.VerticalConnections); err != nil {
		return err
	}
	type at struct {
		parent  string
		x, y, z int
	}
	cells := map[string]at{}
	for _, p := range c.LocationPlacements.Placements {
		cells["location:"+p.LocationID] = at{p.HubID, p.X, p.Y, p.Z}
	}
	for _, p := range c.RoomPlacements.Placements {
		cells["room:"+p.RoomID] = at{p.LocationID, p.X, p.Y, p.Z}
	}
	for _, v := range c.VerticalConnections.Connections {
		a, okA := cells[v.Kind+":"+v.LowerID]
		b, okB := cells[v.Kind+":"+v.UpperID]
		if !okA || !okB || a.parent != b.parent || a.x != b.x || a.y != b.y || b.z-a.z != 1 {
			return fmt.Errorf("vertical connection %s → %s requires placed cells sharing a parent and x/y on adjacent levels", v.LowerID, v.UpperID)
		}
	}
	return nil
}
