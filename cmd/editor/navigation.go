package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Selection belongs to a browser, never to the shared editor handler. Stored
// references are revalidated against the loaded world on every navigation.
type editorSelection struct {
	World, Hub, Location, Room string
	Floor                      int
}

func (h *editorHandler) selection(r *http.Request) editorSelection {
	s := editorSelection{World: h.worldName}
	if cookie, err := r.Cookie("editor_selection"); err == nil {
		if b, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil {
			json.Unmarshal(b, &s)
		}
	}
	if s.World != h.worldName {
		return editorSelection{World: h.worldName}
	}
	if _, err := h.hubs.Get(s.Hub); err != nil {
		s.Hub = ""
	}
	if parent, err := h.spatialParent("location", s.Location); err != nil || parent == "" || parent != s.Hub {
		s.Location = ""
	}
	if parent, err := h.spatialParent("room", s.Room); err != nil || parent == "" || parent != s.Location {
		s.Room = ""
	}
	if s.Location == "" {
		s.Room = ""
		s.Floor = 0
	}
	return s
}
func (h *editorHandler) trackSelection(w http.ResponseWriter, r *http.Request) editorSelection {
	s := h.selection(r)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "content" && r.Method == http.MethodGet && r.URL.Query().Get("details") != "1" {
		switch parts[1] {
		case "hubs":
			if _, err := h.hubs.Get(parts[2]); err == nil {
				if s.Hub != parts[2] {
					s.Location = ""
					s.Room = ""
					s.Floor = 0
				}
				s.Hub = parts[2]
			}
		case "locations", "rooms":
			loc := parts[2]
			room := ""
			if parts[1] == "rooms" {
				room = loc
				loc, _ = h.spatialParent("room", room)
			}
			if _, err := h.locations.Get(loc); err == nil {
				hub, _ := h.spatialParent("location", loc)
				if s.Location != loc {
					s.Floor = 0
					s.Room = ""
				}
				s.Hub = hub
				s.Location = loc
				if room != "" {
					s.Room = room
					ps, _ := h.roomPlacements.List()
					for _, p := range ps {
						if p.RoomID == room {
							s.Floor = p.Z
						}
					}
				} else if r.URL.Query().Has("z") {
					if z, err := strconv.Atoi(r.URL.Query().Get("z")); err == nil {
						s.Floor = z
					}
				}
			}
		}
	}
	b, _ := json.Marshal(s)
	http.SetCookie(w, &http.Cookie{Name: "editor_selection", Value: base64.RawURLEncoding.EncodeToString(b), Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	return s
}
func (h *editorHandler) routeNavigation(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/navigate/") {
		return false
	}
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return true
	}
	s := h.selection(r)
	target := ""
	problem := ""
	switch strings.TrimPrefix(r.URL.Path, "/navigate/") {
	case "location":
		if s.Hub == "" {
			problem = "Hub required."
		} else {
			target = entityURL("hub", s.Hub)
		}
	case "room", "floor":
		if s.Location == "" {
			problem = "Location required."
		} else {
			target = entityURL("location", s.Location) + "?z=" + strconv.Itoa(s.Floor)
		}
	default:
		http.NotFound(w, r)
		return true
	}
	if problem != "" {
		data := basePage("hubs", "", "Select a parent", "selection-error")
		data.StoreError = problem
		h.renderPage(w, r, data)
		return true
	}
	if strings.HasSuffix(r.URL.Path, "/floor") || strings.HasSuffix(r.URL.Path, "/room") {
		target += "#children-heading"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
	return true
}
func (h *editorHandler) navigationData(w http.ResponseWriter, r *http.Request, data *pageData) {
	s := h.trackSelection(w, r)
	for prefix, active := range map[string]string{"/content/world-items": "items", "/content/terminals": "terminals", "/content/npcs": "npcs"} {
		if strings.HasPrefix(r.URL.Path, prefix) {
			data.HeaderTabs = headerTabs(active)
		}
	}
	data.DetailTabs = nil
	data.SubTabs = nil
	if data.Content == "hub-home" && r.URL.Query().Get("saved") == "1" {
		data.HubForm.Notice = "Saved."
	}
	data.Selection = s
	data.SelectionHub, _ = h.spatialDefinition("hub", s.Hub)
	data.SelectionLocation, _ = h.spatialDefinition("location", s.Location)
	data.SpatialTabs = []tab{{Label: "Location", URL: "/navigate/location", Active: data.Drill.Kind == "hub"}, {Label: "Room", URL: "/navigate/room", Active: data.Drill.Kind == "location" || data.Drill.Kind == "room"}, {Label: "Floor", URL: "/navigate/floor"}}
	if data.Content == "hub-home" && s.Hub != "" {
		data.SelectedHub, _ = h.hubs.Get(s.Hub)
		data.SelectedArrival = h.playPage(s.Hub)
	}
}
