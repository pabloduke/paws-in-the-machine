package main

import (
	"fmt"
	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"path/filepath"
	"strconv"
)

type passageRow struct {
	Direction, OtherID, Name, URL string
	Blocked                       bool
}

func (h *editorHandler) passageStore() *relationStore[content.BlockedPassage] {
	return &relationStore[content.BlockedPassage]{path: filepath.Join(filepath.Dir(h.rooms.path), "blocked_passages.json"), label: "blocked passages", key: content.PassageKey,
		decode: func(b []byte) ([]content.BlockedPassage, error) {
			f, e := content.DecodeBlockedPassages(b)
			return f.Passages, e
		},
		encode: func(v []content.BlockedPassage) ([]byte, error) {
			return content.EncodeBlockedPassages(content.BlockedPassagesFile{Version: 1, Passages: v})
		}}
}
func (h *editorHandler) guardPassages(kind, id string) error {
	rows, err := h.passageStore().List()
	if err != nil {
		return err
	}
	for _, v := range rows {
		if v.Kind == kind && (v.AID == id || v.BID == id) {
			return fmt.Errorf("Reopen this cell's blocked passages before moving, unplacing, or deleting it.")
		}
	}
	return nil
}
func (h *editorHandler) setPassage(kind, id, other, blocked, expected string) error {
	if kind != "location" && kind != "room" {
		return fmt.Errorf("Choose a Location or Room.")
	}
	c, err := game.ReadCatalogs(filepath.Dir(h.rooms.path))
	if err != nil {
		return err
	}
	if _, ok := c.HorizontalNeighbors(kind, id, other); !ok {
		return fmt.Errorf("Choose adjoining cells in the same parent and floor.")
	}
	want, err := strconv.ParseBool(blocked)
	if err != nil {
		return fmt.Errorf("Set blocked to true or false.")
	}
	was, err := strconv.ParseBool(expected)
	if err != nil {
		return fmt.Errorf("Current passage state is required; reload this screen.")
	}
	v := content.BlockedPassage{Kind: kind, AID: id, BID: other}
	current := false
	for _, p := range c.BlockedPassages.Passages {
		if content.PassageKey(p) == content.PassageKey(v) {
			current = true
		}
	}
	if was != current {
		return fmt.Errorf("Passage changed; reload before changing it again.")
	}
	if want == current {
		return nil
	}
	if want {
		if v.AID > v.BID {
			v.AID, v.BID = v.BID, v.AID
		}
		_, err = h.passageStore().Put(v)
	} else {
		_, err = h.passageStore().Delete(content.PassageKey(v))
	}
	return err
}
func (h *editorHandler) passagePanel(d *drillPage) {
	if d.Kind == "hub" {
		return
	}
	c, err := game.ReadCatalogs(filepath.Dir(h.rooms.path))
	if err != nil {
		d.Error = err.Error()
		return
	}
	names := map[string]string{}
	if d.Kind == "location" {
		for _, v := range c.Locations.Locations {
			names[v.ID] = v.Name
		}
	} else {
		for _, v := range c.Rooms.Rooms {
			names[v.ID] = v.Name
		}
	}
	for _, dir := range []string{"north", "east", "south", "west"} {
		row := passageRow{Direction: dir}
		for id, name := range names {
			if direction, ok := c.HorizontalNeighbors(d.Kind, d.ID, id); ok && direction == dir {
				row.OtherID = id
				row.Name = name
				row.URL = entityURL(d.Kind, id)
				for _, v := range c.BlockedPassages.Passages {
					if content.PassageKey(v) == content.PassageKey(content.BlockedPassage{Kind: d.Kind, AID: d.ID, BID: id}) {
						row.Blocked = true
					}
				}
				break
			}
		}
		d.Passages = append(d.Passages, row)
	}
	// Entrance choices include all placed floors, independently of the grid filter.
	if d.Kind == "location" {
		d.Entries = nil
		for _, e := range c.LocationEntries.Entries {
			if e.LocationID == d.ID {
				d.EntryID = e.RoomID
			}
		}
		for _, room := range c.Rooms.Rooms {
			for _, p := range c.RoomPlacements.Placements {
				if p.RoomID == room.ID && p.LocationID == d.ID {
					d.Entries = append(d.Entries, placementOption{ID: room.ID, Name: fmt.Sprintf("%s (floor %d)", room.Name, p.Z), Selected: room.ID == d.EntryID})
				}
			}
		}
	} else {
		for _, entry := range c.LocationEntries.Entries {
			if entry.RoomID == d.ID {
				d.IsEntrance = true
			}
		}
	}
}
