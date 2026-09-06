package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

type playOption struct {
	Value, Label string
	Selected     bool
}
type playPage struct {
	Options     []playOption
	Diagnostics []game.Diagnostic
	Error       string
	Ready       bool
}

func (h *editorHandler) playPage(hubID string) playPage {
	page := playPage{}
	c, err := game.ReadCatalogs(filepath.Dir(h.rooms.path))
	if err != nil {
		page.Error = err.Error()
		return page
	}
	if hubID == "" {
		result := game.AssembleAuthored(c)
		page.Diagnostics = result.Diagnostics
		page.Ready = result.Ready()
	}
	placedLocations := map[string]string{}
	locationNames, hubNames := map[string]string{}, map[string]string{}
	for _, l := range c.Locations.Locations {
		locationNames[l.ID] = l.Name
	}
	for _, hub := range c.Hubs.Hubs {
		hubNames[hub.ID] = hub.Name
	}
	assignedLocations, assignedRooms := map[string]string{}, map[string]string{}
	for _, a := range c.LocationAssignments.Assignments {
		assignedLocations[a.LocationID] = a.HubID
	}
	for _, a := range c.RoomAssignments.Assignments {
		assignedRooms[a.RoomID] = a.LocationID
	}
	for _, p := range c.LocationPlacements.Placements {
		if assignedLocations[p.LocationID] != p.HubID || locationNames[p.LocationID] == "" || hubNames[p.HubID] == "" {
			continue
		}
		placedLocations[p.LocationID] = p.HubID
		if hubID != "" && p.HubID != hubID {
			continue
		}
		selected := c.Play.Start != nil && c.Play.Start.Kind == "location" && c.Play.Start.ID == p.LocationID
		value := "location:" + p.LocationID
		if hubID != "" {
			value = p.LocationID
			selected = c.Play.HubArrivals[hubID] == p.LocationID
		}
		page.Options = append(page.Options, playOption{value, hubNames[p.HubID] + " › " + locationNames[p.LocationID], selected})
	}
	if hubID == "" {
		names := map[string]string{}
		for _, r := range c.Rooms.Rooms {
			names[r.ID] = r.Name
		}
		for _, p := range c.RoomPlacements.Placements {
			hub := placedLocations[p.LocationID]
			if hub == "" || names[p.RoomID] == "" || assignedRooms[p.RoomID] != p.LocationID {
				continue
			}
			selected := c.Play.Start != nil && c.Play.Start.Kind == "room" && c.Play.Start.ID == p.RoomID
			page.Options = append(page.Options, playOption{"room:" + p.RoomID, hubNames[hub] + " › " + locationNames[p.LocationID] + " › " + names[p.RoomID], selected})
		}
	}
	return page
}

func (h *editorHandler) savePlaySettings(w http.ResponseWriter, r *http.Request, hubID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid play settings form.", http.StatusBadRequest)
		return
	}
	if hubID != "" {
		if _, err := h.hubs.Get(hubID); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	value := r.FormValue("cell")
	page := h.playPage(hubID)
	if page.Error != "" {
		http.Error(w, page.Error, http.StatusConflict)
		return
	}
	eligible := value == ""
	for _, option := range page.Options {
		if option.Value == value {
			eligible = true
		}
	}
	if !eligible {
		http.Error(w, "Choose a placed cell with placed ancestry.", http.StatusBadRequest)
		return
	}
	err := h.playSettings.Update(func(s *content.PlaySettings) error {
		if hubID != "" {
			if value == "" {
				delete(s.HubArrivals, hubID)
			} else {
				s.HubArrivals[hubID] = value
			}
			return nil
		}
		s.Start = nil
		if value != "" {
			kind, id, ok := strings.Cut(value, ":")
			if !ok {
				return fmt.Errorf("invalid starting cell")
			}
			s.Start = &content.CellRef{Kind: kind, ID: id}
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	target := overviewBasePath
	if hubID != "" {
		target = "/content/hubs/" + hubID
	}
	http.Redirect(w, r, target+"?play_saved=1", http.StatusSeeOther)
}

// Settings protect both the selected cell and its placement ancestry. Existing
// placement rules already require unplacement before a parent reassignment.
func (h *editorHandler) playReferenceProblem(kind, id string) error {
	s, err := h.playSettings.Load()
	if err != nil {
		return err
	}
	protected := map[string]bool{}
	for hub, location := range s.HubArrivals {
		protected["hub:"+hub] = true
		protected["location:"+location] = true
	}
	if s.Start != nil {
		protected[s.Start.Kind+":"+s.Start.ID] = true
		locationID := s.Start.ID
		if s.Start.Kind == "room" {
			records, err := h.roomPlacements.List()
			if err != nil {
				return err
			}
			for _, p := range records {
				if p.RoomID == s.Start.ID {
					locationID = p.LocationID
					protected["location:"+locationID] = true
				}
			}
		}
		records, err := h.locationPlacements.List()
		if err != nil {
			return err
		}
		for _, p := range records {
			if p.LocationID == locationID {
				protected["hub:"+p.HubID] = true
			}
		}
	}
	if protected[kind+":"+id] {
		return fmt.Errorf("clear the starting point or Hub arrival that depends on this %s before removing its placement or deleting it", kind)
	}
	return nil
}

func (h *editorHandler) guardPlayReferences(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return true
	}
	path := r.URL.Path
	kind, id := "", ""
	if path == locationPlacementBasePath+"/unassign" || path == roomPlacementBasePath+"/unassign" {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form.", http.StatusBadRequest)
			return false
		}
		kind = "location"
		if path == roomPlacementBasePath+"/unassign" {
			kind = "room"
		}
		id = r.FormValue("unassign_entity_id")
	} else if strings.HasSuffix(path, "/delete") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 4 && parts[0] == "content" {
			switch parts[1] {
			case "hubs":
				kind = "hub"
			case "locations":
				kind = "location"
			case "rooms":
				kind = "room"
			}
			id = parts[2]
		}
	}
	if kind != "" {
		if err := h.playReferenceProblem(kind, id); err != nil {
			h.renderPlayReferenceError(w, r, kind, id, err)
			return false
		}
	}
	return true
}

// HTMX does not swap error responses by default; use the existing form error
// surfaces so a blocked destructive action always explains how to resolve it.
func (h *editorHandler) renderPlayReferenceError(w http.ResponseWriter, r *http.Request, kind, id string, err error) {
	if !isHTMX(r) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/place/") {
		screen := h.locationPlacementScreen()
		if kind == "room" {
			screen = h.roomPlacementScreen()
		}
		h.renderPlacementResponse(w, r, screen, h.spatialPlacementsPage(screen, r.FormValue("parent_id"), "", err.Error(), ""))
		return
	}
	if kind == "hub" {
		hub, _ := h.hubs.Get(id)
		form := formFromHub(hub)
		form.GeneralError = err.Error()
		h.render(w, "hubs-body", h.hubsPage("", form))
		return
	}
	screen := h.locationScreen()
	if kind == "room" {
		screen = h.roomScreen()
	}
	entity, _ := screen.get(id)
	form := describedForm{ID: id, Name: entity.Name, Description: entity.Description, GeneralError: err.Error(), Errors: map[string]string{}}
	h.render(w, "described-body", h.describedPage(screen, "", form))
}
