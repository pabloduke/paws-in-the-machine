package main

import (
	"fmt"
	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"net/url"
)

// saveEntity is shared by contextual creation and the JSON resource interface.
func (h *editorHandler) saveEntity(kind, id string, v url.Values) (string, error) {
	switch kind {
	case "hub":
		if id == "" {
			r, e := h.hubs.Create(v.Get("name"))
			return r.ID, e
		}
		r, e := h.hubs.Update(id, v.Get("name"))
		return r.ID, e
	case "location", "room", "npc":
		s := h.locationScreen()
		if kind == "room" {
			s = h.roomScreen()
		}
		if kind == "npc" {
			s = h.npcScreen()
		}
		in := describedInput{Name: v.Get("name"), Description: v.Get("description")}.normalized()
		if in.Name == "" || in.Description == "" {
			return "", fmt.Errorf("Name and Description are required.")
		}
		if id == "" {
			r, e := s.create(in)
			return r.ID, e
		}
		r, e := s.update(id, in)
		return r.ID, e
	case "world_item":
		in := worldItemInput{Name: v.Get("name"), Kind: content.WorldItemKind(v.Get("kind")), ShortDescription: v.Get("short_description"), FullDescription: v.Get("full_description")}
		if id == "" {
			r, e := h.items.Create(in)
			return r.ID, e
		}
		r, e := h.items.Update(id, in)
		return r.ID, e
	case "terminal":
		in := terminalInput{HostName: v.Get("host_name")}
		if id == "" {
			r, e := h.terminals.Create(in)
			return r.ID, e
		}
		r, e := h.terminals.Update(id, in)
		return r.ID, e
	}
	return "", fmt.Errorf("unknown entity kind")
}

func (h *editorHandler) deleteEntity(kind, id string) error {
	if err := h.playReferenceProblem(kind, id); err != nil {
		return err
	}
	if err := h.guardVertical(kind, id); err != nil {
		return err
	}
	blocked := ""
	var err error
	switch kind {
	case "location", "room":
		blocked, err = h.cellBlockingDelete(kind, id)
		if err != nil {
			return err
		}
		if blocked != "" {
			return fmt.Errorf("%s", blocked)
		}
		parent, e := h.spatialParent(kind, id)
		if e != nil {
			return e
		}
		if parent != "" {
			return fmt.Errorf("Unassign the parent before deletion.")
		}
		if kind == "location" {
			has, e := h.locationHasRooms(id)
			if e != nil {
				return e
			}
			if has {
				return fmt.Errorf("Unassign child Rooms before deletion.")
			}
			_, e = h.locations.Delete(id)
			return e
		}
		_, e = h.rooms.Delete(id)
		return e
	case "hub":
		has, e := h.hubHasLocations(id)
		if e != nil {
			return e
		}
		if has {
			return fmt.Errorf("Unassign child Locations before deletion.")
		}
		_, e = h.hubs.Delete(id)
		return e
	case "world_item", "npc", "terminal":
		blocked, err = h.contentsBlockingDelete(kind, id)
		if err != nil {
			return err
		}
		if blocked != "" {
			return fmt.Errorf("%s", blocked)
		}
		if kind == "world_item" {
			_, err = h.items.Delete(id)
			return err
		}
		if kind == "npc" {
			_, err = h.npcs.Delete(id)
			return err
		}
		network, e := h.terminalNetwork(id)
		if e != nil {
			return e
		}
		if network != "" {
			return fmt.Errorf("Unassign terminal network first.")
		}
		access, e := h.terminalHasAccess(id)
		if e != nil {
			return e
		}
		if access {
			return fmt.Errorf("Revoke terminal access first.")
		}
		_, e = h.terminals.Delete(id)
		return e
	}
	return fmt.Errorf("unknown entity kind")
}
