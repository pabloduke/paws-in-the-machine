package engine

import (
	"encoding/json"
	"fmt"
)

// Save/load (docs/systems/saveload.md): a save is the chess position
// — flags, character numbers, entity positions. Content rebuilds from
// code; Placed entities re-derive; the engine never touches files
// (callers marshal/unmarshal and own the I/O).

// SaveVersion guards against loading saves from incompatible builds.
const SaveVersion = 1

// SaveState is everything a session needs to resume.
type SaveState struct {
	DevModified bool            `json:"dev_modified,omitempty"`
	Version     int             `json:"version"`
	Flags       map[string]bool `json:"flags"`
	Stats       Stats           `json:"stats"`
	XP          int             `json:"xp"`
	Level       int             `json:"level"`
	StatPoints  int             `json:"stat_points"`
	Seed        int64           `json:"seed"`
	// Positions maps entity ID -> parent entity ID for every entity
	// in the tree at save time.
	Positions map[string]string `json:"positions"`
}

// Snapshot captures the current world.
func Snapshot(w *World) SaveState {
	s := SaveState{
		DevModified: w.DevModified,
		Version:     SaveVersion,
		Flags:       map[string]bool{},
		Stats:       w.Stats,
		XP:          w.XP,
		Level:       w.Level,
		StatPoints:  w.StatPoints,
		Seed:        w.Seed,
		Positions:   map[string]string{},
	}
	for k, v := range w.Flags {
		s.Flags[k] = v
	}
	w.Root.Walk(func(e *Entity) bool {
		if e.Parent != nil {
			s.Positions[e.ID] = e.Parent.ID
		}
		return true
	})
	// The player hangs off a room, not under Root's walk when carried
	// nowhere odd — Walk covers the whole tree including the player,
	// but record it explicitly in case content ever detaches it.
	if w.Player.Parent != nil {
		s.Positions[w.Player.ID] = w.Player.Parent.ID
	}
	return s
}

// Apply restores a snapshot onto a constructed world (fresh or
// mid-session — positions cover every entity and flags replace
// wholesale, so the world is fully overwritten). Placed entities
// ignore their saved position and re-derive from flags. Arrival
// tracking resets so loading never fires Enter rules.
func (s SaveState) Apply(w *World) error {
	if s.Version != SaveVersion {
		return fmt.Errorf("save is version %d; this build reads %d", s.Version, SaveVersion)
	}
	w.DevModified = w.DevModified || s.DevModified
	w.Flags = map[string]bool{}
	for k, v := range s.Flags {
		w.Flags[k] = v
	}
	w.Stats = s.Stats
	w.XP = s.XP
	w.Level = s.Level
	w.StatPoints = s.StatPoints
	w.Seed = s.Seed
	w.Pending = nil

	for id, parentID := range s.Positions {
		e := w.FindID(id)
		if e == nil || e.Parent == nil {
			continue // content changed since the save; skip quietly
		}
		if _, placed := Part[Placed](e); placed {
			continue // presence re-derives these
		}
		if e.Parent.ID == parentID {
			continue
		}
		if dest := w.FindID(parentID); dest != nil {
			dest.Add(e)
		}
	}
	w.applyPlacements()
	w.prevRoom = w.Room() // loading is not an arrival
	return nil
}

// Marshal renders the snapshot as versioned JSON.
func (s SaveState) Marshal() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// UnmarshalSave parses a snapshot previously produced by Marshal.
func UnmarshalSave(data []byte) (SaveState, error) {
	var s SaveState
	if err := json.Unmarshal(data, &s); err != nil {
		return SaveState{}, fmt.Errorf("save file is unreadable: %w", err)
	}
	return s, nil
}
