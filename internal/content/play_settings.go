package content

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const PlaySettingsVersion = 1

// CellRef identifies a standable authored cell; UUIDs are catalog-local.
type CellRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type PlaySettings struct {
	Version     int               `json:"version"`
	Start       *CellRef          `json:"start,omitempty"`
	HubArrivals map[string]string `json:"hub_arrivals"`
}

func EmptyPlaySettings() PlaySettings {
	return PlaySettings{Version: PlaySettingsVersion, HubArrivals: map[string]string{}}
}

func ValidatePlaySettings(s PlaySettings) error {
	if s.Version != PlaySettingsVersion {
		return fmt.Errorf("play settings: unsupported version %d", s.Version)
	}
	if s.HubArrivals == nil {
		return fmt.Errorf("play settings: hub_arrivals must be an object")
	}
	if s.Start != nil && (!validContainerKind(s.Start.Kind) || !validUUID(s.Start.ID)) {
		return fmt.Errorf("play settings: invalid starting cell")
	}
	for hub, location := range s.HubArrivals {
		if !validUUID(hub) || !validUUID(location) {
			return fmt.Errorf("play settings: invalid Hub arrival reference")
		}
	}
	return nil
}

func DecodePlaySettings(data []byte) (PlaySettings, error) {
	var s PlaySettings
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return s, fmt.Errorf("play settings: %w", err)
	}
	if err := rejectTrailingJSON(d); err != nil {
		return s, err
	}
	return s, ValidatePlaySettings(s)
}

func EncodePlaySettings(s PlaySettings) ([]byte, error) {
	if err := ValidatePlaySettings(s); err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	return append(b, '\n'), err
}
