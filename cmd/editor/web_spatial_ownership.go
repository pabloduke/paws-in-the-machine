package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type ownedChildRow struct {
	ID, Name, Description, Placement string
}

type spatialOwnershipPage struct {
	ParentID, ParentName, ParentLabel                           string
	ChildLabel, ChildPlural, ChildPath, ChildBasePath, BasePath string
	Assigned                                                    []ownedChildRow
	Unassigned                                                  []describedFields
	NestedForm                                                  describedForm
	GeneralError, Notice                                        string
	SupportsEntry, HasEntry                                     bool
	EntryRoomID                                                 string
	EligibleEntries                                             []placementOption
}

type ownershipScreen struct {
	parentLabel, childLabel, childPlural, basePath string
	getParent                                      func(string) (describedFields, error)
	listChildren                                   func() ([]describedFields, error)
	getChild                                       func(string) (describedFields, error)
	createChild                                    func(describedInput) (describedFields, error)
	listAssignments                                func() (map[string]string, error)
	assign                                         func(string, string) error
	unassign                                       func(string) error
	isPlaced                                       func(string) (bool, error)
	placementLabels                                func() (map[string]string, error)
	supportsEntry                                  bool
}

func (h *editorHandler) hubLocationOwnership() ownershipScreen {
	return ownershipScreen{
		parentLabel: "Hub", childLabel: "Location", childPlural: "Locations", basePath: "/content/hubs",
		getParent: func(id string) (describedFields, error) {
			v, err := h.hubs.Get(id)
			return describedFields{ID: v.ID, Name: v.Name}, err
		},
		listChildren: func() ([]describedFields, error) {
			records, err := h.locations.List()
			out := make([]describedFields, len(records))
			for i, v := range records {
				out[i] = fieldsFromLocation(v)
			}
			return out, err
		},
		getChild: func(id string) (describedFields, error) {
			v, err := h.locations.Get(id)
			return fieldsFromLocation(v), err
		},
		createChild: func(in describedInput) (describedFields, error) {
			v, err := h.locations.Create(in)
			return fieldsFromLocation(v), err
		},
		listAssignments: func() (map[string]string, error) {
			records, err := h.locationAssignments.List()
			out := map[string]string{}
			for _, v := range records {
				out[v.LocationID] = v.HubID
			}
			return out, err
		},
		assign: func(childID, parentID string) error {
			_, err := h.locationAssignments.Assign(childID, parentID)
			return err
		},
		unassign: func(id string) error { _, err := h.locationAssignments.Unassign(id); return err },
		isPlaced: h.locationIsPlaced,
		placementLabels: func() (map[string]string, error) {
			records, err := h.locationPlacements.List()
			out := map[string]string{}
			for _, v := range records {
				out[v.LocationID] = fmt.Sprintf("%d,%d", v.X, v.Y)
			}
			return out, err
		},
	}
}

func (h *editorHandler) locationRoomOwnership() ownershipScreen {
	return ownershipScreen{
		parentLabel: "Location", childLabel: "Room", childPlural: "Rooms", basePath: "/content/locations", supportsEntry: true,
		getParent: func(id string) (describedFields, error) {
			v, err := h.locations.Get(id)
			return fieldsFromLocation(v), err
		},
		listChildren: func() ([]describedFields, error) {
			records, err := h.rooms.List()
			out := make([]describedFields, len(records))
			for i, v := range records {
				out[i] = fieldsFromRoom(v)
			}
			return out, err
		},
		getChild: func(id string) (describedFields, error) { v, err := h.rooms.Get(id); return fieldsFromRoom(v), err },
		createChild: func(in describedInput) (describedFields, error) {
			v, err := h.rooms.Create(in)
			return fieldsFromRoom(v), err
		},
		listAssignments: func() (map[string]string, error) {
			records, err := h.roomAssignments.List()
			out := map[string]string{}
			for _, v := range records {
				out[v.RoomID] = v.LocationID
			}
			return out, err
		},
		assign: func(childID, parentID string) error {
			_, err := h.roomAssignments.Assign(childID, parentID)
			return err
		},
		unassign: func(id string) error { _, err := h.roomAssignments.Unassign(id); return err },
		isPlaced: h.roomIsPlaced,
		placementLabels: func() (map[string]string, error) {
			records, err := h.roomPlacements.List()
			out := map[string]string{}
			for _, v := range records {
				out[v.RoomID] = fmt.Sprintf("%d,%d", v.X, v.Y)
			}
			return out, err
		},
	}
}

func (h *editorHandler) routeHubID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/content/hubs/")
	parts := strings.Split(rest, "/")
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 1:
		h.serveHubs(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "arrival":
		h.savePlaySettings(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "delete":
		h.deleteHub(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "locations":
		h.serveOwnedChildren(w, r, h.hubLocationOwnership(), parts[0])
	case len(parts) == 2 && parts[1] == "assign":
		h.assignOwnedChild(w, r, h.hubLocationOwnership(), parts[0])
	case len(parts) == 3 && parts[1] == "unassign":
		h.unassignOwnedChild(w, r, h.hubLocationOwnership(), parts[0], parts[2])
	default:
		http.NotFound(w, r)
	}
}

func (h *editorHandler) routeLocationID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/content/locations/")
	parts := strings.Split(rest, "/")
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 1:
		h.serveDescribed(w, r, h.locationScreen(), parts[0])
	case len(parts) == 2 && parts[1] == "delete":
		h.deleteDescribed(w, r, h.locationScreen(), parts[0])
	case len(parts) == 2 && parts[1] == "rooms":
		h.serveOwnedChildren(w, r, h.locationRoomOwnership(), parts[0])
	case len(parts) == 2 && parts[1] == "assign":
		h.assignOwnedChild(w, r, h.locationRoomOwnership(), parts[0])
	case len(parts) == 3 && parts[1] == "unassign":
		h.unassignOwnedChild(w, r, h.locationRoomOwnership(), parts[0], parts[2])
	case len(parts) == 2 && parts[1] == "entry":
		h.setOwnedLocationEntry(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "entry-clear":
		h.clearOwnedLocationEntry(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (h *editorHandler) serveOwnedChildren(w http.ResponseWriter, r *http.Request, screen ownershipScreen, parentID string) {
	if r.Method == http.MethodPost {
		h.createOwnedChild(w, r, screen, parentID)
		return
	}
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	h.renderOwnership(w, r, screen, parentID, describedForm{Errors: map[string]string{}}, "", "")
}

func (h *editorHandler) createOwnedChild(w http.ResponseWriter, r *http.Request, screen ownershipScreen, parentID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid child form", http.StatusBadRequest)
		return
	}
	form := describedForm{Name: r.FormValue("name"), Description: r.FormValue("description"), Errors: map[string]string{}}
	input := describedInput{Name: form.Name, Description: form.Description}.normalized()
	if input.Name == "" {
		form.Errors["name"] = screen.childLabel + " Name is required."
	}
	if input.Description == "" {
		form.Errors["description"] = "Description is required."
	}
	if _, err := screen.getParent(parentID); err != nil {
		http.NotFound(w, r)
		return
	}
	if len(form.Errors) == 0 {
		child, err := screen.createChild(input)
		if err != nil {
			form.GeneralError = err.Error()
		} else if err := screen.assign(child.ID, parentID); err != nil {
			form.GeneralError = screen.childLabel + " was created but could not be assigned; it remains unassigned: " + err.Error()
		} else {
			form = describedForm{Errors: map[string]string{}, Notice: "Created and assigned " + child.Name + "."}
		}
	}
	notice := form.Notice
	form.Notice = ""
	h.renderOwnership(w, r, screen, parentID, form, notice, "")
}

func (h *editorHandler) assignOwnedChild(w http.ResponseWriter, r *http.Request, screen ownershipScreen, parentID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid assignment form", http.StatusBadRequest)
		return
	}
	childID := r.FormValue("child_id")
	parent, parentErr := screen.getParent(parentID)
	child, childErr := screen.getChild(childID)
	generalError, notice := "", ""
	assignments, listErr := screen.listAssignments()
	switch {
	case parentErr != nil:
		http.NotFound(w, r)
		return
	case childErr != nil:
		generalError = "Choose an existing unassigned " + screen.childLabel + "."
	case listErr != nil:
		generalError = listErr.Error()
	case assignments[childID] != "":
		generalError = "That " + screen.childLabel + " is already assigned to a " + screen.parentLabel + "."
	case screen.assign(childID, parent.ID) != nil:
		generalError = "Could not assign " + child.Name + "."
	default:
		notice = "Assigned " + child.Name + "."
	}
	h.renderOwnership(w, r, screen, parentID, describedForm{Errors: map[string]string{}}, notice, generalError)
}

func (h *editorHandler) unassignOwnedChild(w http.ResponseWriter, r *http.Request, screen ownershipScreen, parentID, childID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	assignments, err := screen.listAssignments()
	if err != nil {
		h.renderOwnership(w, r, screen, parentID, describedForm{Errors: map[string]string{}}, "", err.Error())
		return
	}
	if assignments[childID] != parentID {
		http.NotFound(w, r)
		return
	}
	child, _ := screen.getChild(childID)
	placed, err := screen.isPlaced(childID)
	generalError, notice := "", ""
	if err != nil {
		generalError = err.Error()
	} else if placed {
		generalError = "Remove " + child.Name + " from the grid before unassigning it."
	} else if screen.supportsEntry && h.roomIsAnyEntry(childID) {
		generalError = "Clear this Room as the Location entry before unassigning it."
	} else if err := screen.unassign(childID); err != nil {
		generalError = err.Error()
	} else {
		notice = "Unassigned " + child.Name + "."
	}
	h.renderOwnership(w, r, screen, parentID, describedForm{Errors: map[string]string{}}, notice, generalError)
}

func (h *editorHandler) renderOwnership(w http.ResponseWriter, r *http.Request, screen ownershipScreen, parentID string, nestedForm describedForm, notice, generalError string) {
	parent, err := screen.getParent(parentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ownership, storeErr := h.ownershipPage(screen, parent, nestedForm, notice, generalError)
	var data pageData
	if screen.parentLabel == "Hub" {
		data = h.hubsPage("", hubForm{ID: parent.ID, Name: parent.Name})
		data.HubView = "locations"
	} else {
		data = h.describedPage(h.locationScreen(), "", formFromDescribed(parent))
		data.DescribedView = "rooms"
	}
	data.Ownership = ownership
	if storeErr != nil {
		data.StoreError = storeErr.Error()
	}
	if isHTMX(r) {
		if notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
		}
		if screen.parentLabel == "Hub" {
			h.render(w, "hub-editor", data)
		} else {
			h.render(w, "described-editor", data)
		}
		return
	}
	if notice != "" && r.Method == http.MethodPost {
		http.Redirect(w, r, screen.basePath+"/"+url.PathEscape(parentID)+"/"+strings.ToLower(screen.childPlural)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) ownershipPage(screen ownershipScreen, parent describedFields, nestedForm describedForm, notice, generalError string) (spatialOwnershipPage, error) {
	childBasePath := "/content/locations"
	if screen.childLabel == "Room" {
		childBasePath = "/content/rooms"
	}
	page := spatialOwnershipPage{ParentID: parent.ID, ParentName: parent.Name, ParentLabel: screen.parentLabel, ChildLabel: screen.childLabel, ChildPlural: screen.childPlural, ChildPath: strings.ToLower(screen.childPlural), ChildBasePath: childBasePath, BasePath: screen.basePath, NestedForm: nestedForm, Notice: notice, GeneralError: generalError, SupportsEntry: screen.supportsEntry}
	children, err := screen.listChildren()
	if err != nil {
		return page, err
	}
	assignments, err := screen.listAssignments()
	if err != nil {
		return page, err
	}
	placements, err := screen.placementLabels()
	if err != nil {
		return page, err
	}
	for _, child := range children {
		if assignments[child.ID] == parent.ID {
			placement := "Unplaced"
			if coordinates := placements[child.ID]; coordinates != "" {
				placement = coordinates
			}
			page.Assigned = append(page.Assigned, ownedChildRow{ID: child.ID, Name: child.Name, Description: child.Description, Placement: placement})
		} else if assignments[child.ID] == "" {
			page.Unassigned = append(page.Unassigned, child)
		}
	}
	sort.SliceStable(page.Assigned, func(i, j int) bool {
		return strings.ToLower(page.Assigned[i].Name) < strings.ToLower(page.Assigned[j].Name)
	})
	sort.SliceStable(page.Unassigned, func(i, j int) bool {
		return strings.ToLower(page.Unassigned[i].Name) < strings.ToLower(page.Unassigned[j].Name)
	})
	if screen.supportsEntry {
		entries, err := h.locationEntries.List()
		if err != nil {
			return page, err
		}
		for _, entry := range entries {
			if entry.LocationID == parent.ID {
				page.HasEntry = true
				page.EntryRoomID = entry.RoomID
			}
		}
		for _, child := range page.Assigned {
			if child.Placement != "Unplaced" {
				page.EligibleEntries = append(page.EligibleEntries, placementOption{ID: child.ID, Name: child.Name, Selected: child.ID == page.EntryRoomID})
			}
		}
	}
	return page, nil
}

func (h *editorHandler) setOwnedLocationEntry(w http.ResponseWriter, r *http.Request, locationID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid entry-room form", http.StatusBadRequest)
		return
	}
	roomID := r.FormValue("entry_room_id")
	assignments, err := h.roomAssignments.List()
	assigned := false
	for _, a := range assignments {
		if a.RoomID == roomID && a.LocationID == locationID {
			assigned = true
		}
	}
	placed, placeErr := h.spatialPlacementMatches(h.roomPlacementScreen(), roomID, locationID)
	generalError, notice := "", ""
	if err != nil {
		generalError = err.Error()
	} else if placeErr != nil {
		generalError = placeErr.Error()
	} else if !assigned || !placed {
		generalError = "The entry Room must be assigned to and placed in this Location."
	} else if _, err := h.locationEntries.Set(locationID, roomID); err != nil {
		generalError = err.Error()
	} else {
		notice = "Entry Room saved."
	}
	h.renderOwnership(w, r, h.locationRoomOwnership(), locationID, describedForm{Errors: map[string]string{}}, notice, generalError)
}

func (h *editorHandler) clearOwnedLocationEntry(w http.ResponseWriter, r *http.Request, locationID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	_, err := h.locationEntries.Clear(locationID)
	notice, generalError := "Entry Room cleared.", ""
	if errors.Is(err, errLocationEntryNotFound) {
		notice = "Entry Room is already unset."
	} else if err != nil {
		notice, generalError = "", err.Error()
	}
	h.renderOwnership(w, r, h.locationRoomOwnership(), locationID, describedForm{Errors: map[string]string{}}, notice, generalError)
}

func (h *editorHandler) roomIsAnyEntry(roomID string) bool {
	entries, err := h.locationEntries.List()
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.RoomID == roomID {
			return true
		}
	}
	return false
}
