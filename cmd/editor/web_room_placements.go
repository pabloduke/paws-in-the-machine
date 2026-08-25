package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const locationPlacementBasePath = "/place/locations"
const roomPlacementBasePath = "/place/rooms"
const defaultLocationGridSize = 10
const defaultRoomGridSize = 5

type placementOption struct {
	ID, Name string
	Selected bool
}
type placementEntityOption struct{ ID, Name, Placement string }
type spatialPlacement struct {
	EntityID, ParentID string
	X, Y               int
}
type spatialGridCell struct {
	X, Y                 int
	EntityID, EntityName string
	Occupied             bool
}
type spatialGridRow struct {
	Y     int
	Cells []spatialGridCell
}

type roomPlacementPage struct {
	Parents                               []placementOption
	Entities                              []placementEntityOption
	SelectedParentID, SelectedParentName  string
	ParentLabel, EntityLabel, BasePath    string
	XHeaders                              []int
	Rows                                  []spatialGridRow
	GridWidth                             int
	Notice, GeneralError, CoordinateError string
}

type placementScreen struct {
	key, title, parentLabel, entityLabel, basePath string
	gridSize                                       int
	listParents                                    func() ([]describedFields, error)
	getParent                                      func(string) (describedFields, error)
	listEntities                                   func() ([]describedFields, error)
	getEntity                                      func(string) (describedFields, error)
	listPlacements                                 func() ([]spatialPlacement, error)
	listAssignments                                func() (map[string]string, error)
	place                                          func(string, string, int, int) error
	unassign                                       func(string) error
}

func (h *editorHandler) locationPlacementScreen() placementScreen {
	return placementScreen{
		key: "locations", title: "Assign Locations", parentLabel: "Hub", entityLabel: "Location", basePath: locationPlacementBasePath, gridSize: defaultLocationGridSize,
		listParents: func() ([]describedFields, error) {
			records, err := h.hubs.List()
			out := make([]describedFields, len(records))
			for i, v := range records {
				out[i] = describedFields{ID: v.ID, Name: v.Name}
			}
			return out, err
		},
		getParent: func(id string) (describedFields, error) {
			v, err := h.hubs.Get(id)
			return describedFields{ID: v.ID, Name: v.Name}, err
		},
		listEntities: func() ([]describedFields, error) {
			records, err := h.locations.List()
			out := make([]describedFields, len(records))
			for i, v := range records {
				out[i] = fieldsFromLocation(v)
			}
			return out, err
		},
		getEntity: func(id string) (describedFields, error) {
			v, err := h.locations.Get(id)
			return fieldsFromLocation(v), err
		},
		listPlacements: func() ([]spatialPlacement, error) {
			records, err := h.locationPlacements.List()
			out := make([]spatialPlacement, len(records))
			for i, v := range records {
				out[i] = spatialPlacement{EntityID: v.LocationID, ParentID: v.HubID, X: v.X, Y: v.Y}
			}
			return out, err
		},
		listAssignments: func() (map[string]string, error) {
			records, err := h.locationAssignments.List()
			out := map[string]string{}
			for _, v := range records {
				out[v.LocationID] = v.HubID
			}
			return out, err
		},
		place: func(entityID, parentID string, x, y int) error {
			_, err := h.locationPlacements.Place(entityID, parentID, x, y)
			return err
		},
		unassign: func(id string) error { _, err := h.locationPlacements.Unassign(id); return err },
	}
}

func (h *editorHandler) roomPlacementScreen() placementScreen {
	return placementScreen{
		key: "rooms", title: "Assign Rooms", parentLabel: "Location", entityLabel: "Room", basePath: roomPlacementBasePath, gridSize: defaultRoomGridSize,
		listParents: func() ([]describedFields, error) {
			records, err := h.locations.List()
			out := make([]describedFields, len(records))
			for i, v := range records {
				out[i] = fieldsFromLocation(v)
			}
			return out, err
		},
		getParent: func(id string) (describedFields, error) {
			v, err := h.locations.Get(id)
			return fieldsFromLocation(v), err
		},
		listEntities: func() ([]describedFields, error) {
			records, err := h.rooms.List()
			out := make([]describedFields, len(records))
			for i, v := range records {
				out[i] = fieldsFromRoom(v)
			}
			return out, err
		},
		getEntity: func(id string) (describedFields, error) { v, err := h.rooms.Get(id); return fieldsFromRoom(v), err },
		listPlacements: func() ([]spatialPlacement, error) {
			records, err := h.roomPlacements.List()
			out := make([]spatialPlacement, len(records))
			for i, v := range records {
				out[i] = spatialPlacement{EntityID: v.RoomID, ParentID: v.LocationID, X: v.X, Y: v.Y}
			}
			return out, err
		},
		listAssignments: func() (map[string]string, error) {
			records, err := h.roomAssignments.List()
			out := map[string]string{}
			for _, v := range records {
				out[v.RoomID] = v.LocationID
			}
			return out, err
		},
		place: func(entityID, parentID string, x, y int) error {
			_, err := h.roomPlacements.Place(entityID, parentID, x, y)
			return err
		},
		unassign: func(id string) error { _, err := h.roomPlacements.Unassign(id); return err },
	}
}

func (h *editorHandler) serveLocationPlacements(w http.ResponseWriter, r *http.Request) {
	h.servePlacements(w, r, h.locationPlacementScreen())
}
func (h *editorHandler) serveRoomPlacements(w http.ResponseWriter, r *http.Request) {
	h.servePlacements(w, r, h.roomPlacementScreen())
}
func (h *editorHandler) placeLocation(w http.ResponseWriter, r *http.Request) {
	h.placeSpatialEntity(w, r, h.locationPlacementScreen())
}
func (h *editorHandler) placeRoom(w http.ResponseWriter, r *http.Request) {
	h.placeSpatialEntity(w, r, h.roomPlacementScreen())
}
func (h *editorHandler) unassignLocation(w http.ResponseWriter, r *http.Request) {
	h.unassignSpatialEntity(w, r, h.locationPlacementScreen())
}
func (h *editorHandler) unassignRoom(w http.ResponseWriter, r *http.Request) {
	h.unassignSpatialEntity(w, r, h.roomPlacementScreen())
}

func (h *editorHandler) servePlacements(w http.ResponseWriter, r *http.Request, screen placementScreen) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	h.renderPage(w, r, h.spatialPlacementsPage(screen, r.URL.Query().Get("parent_id"), "", "", ""))
}

func (h *editorHandler) placeSpatialEntity(w http.ResponseWriter, r *http.Request, screen placementScreen) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid placement form", http.StatusBadRequest)
		return
	}
	parentID, entityID := r.FormValue("parent_id"), r.FormValue("entity_id")
	x, y, coordinateErr := placementCoordinates(r)
	var notice, generalError string
	if coordinateErr == "" {
		if _, err := screen.getParent(parentID); err != nil {
			generalError = "Choose an existing " + screen.parentLabel + "."
		} else if entity, err := screen.getEntity(entityID); err != nil {
			generalError = "Choose an existing " + screen.entityLabel + "."
		} else if assignments, err := screen.listAssignments(); err != nil {
			generalError = err.Error()
		} else if assignments[entityID] != parentID {
			generalError = "Assign this " + screen.entityLabel + " to the selected " + screen.parentLabel + " before placing it."
		} else if occupied, err := h.spatialCellOccupied(screen, parentID, entityID, x, y); err != nil {
			generalError = err.Error()
		} else if occupied {
			coordinateErr = "That cell is already occupied by another " + screen.entityLabel + "."
		} else if screen.key == "rooms" && h.roomIsEntryElsewhere(entityID, parentID) {
			generalError = "Clear this Room as its current Location entry before moving it."
		} else if err := screen.place(entityID, parentID, x, y); err != nil {
			generalError = err.Error()
		} else {
			notice = fmt.Sprintf("Placed %s at %d, %d.", entity.Name, x, y)
		}
	}
	h.renderPlacementResponse(w, r, screen, h.spatialPlacementsPage(screen, parentID, notice, generalError, coordinateErr))
}

func (h *editorHandler) unassignSpatialEntity(w http.ResponseWriter, r *http.Request, screen placementScreen) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid placement form", http.StatusBadRequest)
		return
	}
	parentID, entityID := r.FormValue("parent_id"), r.FormValue("unassign_entity_id")
	entity, _ := screen.getEntity(entityID)
	var notice, generalError string
	if matches, err := h.spatialPlacementMatches(screen, entityID, parentID); err != nil {
		generalError = err.Error()
	} else if !matches {
		http.NotFound(w, r)
		return
	} else if screen.key == "rooms" && h.roomIsEntryElsewhere(entityID, "") {
		generalError = "Clear this Room as the Location entry before unassigning it."
	} else if err := screen.unassign(entityID); err != nil {
		generalError = err.Error()
	} else {
		notice = "Unplaced " + entity.Name + "."
	}
	h.renderPlacementResponse(w, r, screen, h.spatialPlacementsPage(screen, parentID, notice, generalError, ""))
}

func placementCoordinates(r *http.Request) (int, int, string) {
	xText, yText := strings.TrimSpace(r.FormValue("x")), strings.TrimSpace(r.FormValue("y"))
	if cell := r.FormValue("cell"); cell != "" {
		parts := strings.Split(cell, ",")
		if len(parts) != 2 {
			return 0, 0, "Choose a valid grid cell."
		}
		xText, yText = parts[0], parts[1]
	}
	x, xErr := strconv.Atoi(xText)
	y, yErr := strconv.Atoi(yText)
	if xErr != nil || yErr != nil || x < 0 || y < 0 {
		return 0, 0, "X and Y must be non-negative whole numbers."
	}
	return x, y, ""
}

func (h *editorHandler) renderPlacementResponse(w http.ResponseWriter, r *http.Request, screen placementScreen, data pageData) {
	if isHTMX(r) {
		if data.Placement.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
		}
		h.render(w, "room-placement-body", data)
		return
	}
	if data.Placement.Notice != "" {
		http.Redirect(w, r, screen.basePath+"?parent_id="+url.QueryEscape(data.Placement.SelectedParentID), http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) spatialPlacementsPage(screen placementScreen, parentID, notice, generalError, coordinateError string) pageData {
	data := basePage("place", screen.key, screen.title, "room-placement")
	data.Placement = roomPlacementPage{Notice: notice, GeneralError: generalError, CoordinateError: coordinateError, ParentLabel: screen.parentLabel, EntityLabel: screen.entityLabel, BasePath: screen.basePath}
	if err := h.validateSpatialRelationships(); err != nil {
		data.StoreError = err.Error()
		return data
	}
	parents, err := screen.listParents()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	sort.SliceStable(parents, func(i, j int) bool { return strings.ToLower(parents[i].Name) < strings.ToLower(parents[j].Name) })
	if parentID == "" && len(parents) > 0 {
		parentID = parents[0].ID
	}
	for _, parent := range parents {
		selected := parent.ID == parentID
		data.Placement.Parents = append(data.Placement.Parents, placementOption{ID: parent.ID, Name: parent.Name, Selected: selected})
		if selected {
			data.Placement.SelectedParentID, parentID = data.Placement.Parents[len(data.Placement.Parents)-1].ID, parent.ID
			data.Placement.SelectedParentName = parent.Name
		}
	}
	if parentID != "" && data.Placement.SelectedParentID == "" {
		data.Placement.GeneralError = "The selected " + screen.parentLabel + " no longer exists."
		return data
	}
	entities, err := screen.listEntities()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	placements, err := screen.listPlacements()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	assignments, err := screen.listAssignments()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	entityByID, placementByEntity := map[string]describedFields{}, map[string]spatialPlacement{}
	for _, entity := range entities {
		entityByID[entity.ID] = entity
	}
	for _, p := range placements {
		placementByEntity[p.EntityID] = p
	}
	ownedEntities := make([]describedFields, 0, len(entities))
	for _, entity := range entities {
		if assignments[entity.ID] == parentID {
			ownedEntities = append(ownedEntities, entity)
		}
	}
	sort.SliceStable(ownedEntities, func(i, j int) bool {
		return strings.ToLower(ownedEntities[i].Name) < strings.ToLower(ownedEntities[j].Name)
	})
	for _, entity := range ownedEntities {
		option := placementEntityOption{ID: entity.ID, Name: entity.Name, Placement: "Unplaced"}
		if p, ok := placementByEntity[entity.ID]; ok {
			parentName := "Unknown"
			for _, parent := range parents {
				if parent.ID == p.ParentID {
					parentName = parent.Name
					break
				}
			}
			option.Placement = fmt.Sprintf("%s at %d,%d", parentName, p.X, p.Y)
		}
		data.Placement.Entities = append(data.Placement.Entities, option)
	}
	width, height := screen.gridSize, screen.gridSize
	cells := map[string]spatialPlacement{}
	for _, p := range placements {
		if p.ParentID != parentID {
			continue
		}
		cells[fmt.Sprintf("%d,%d", p.X, p.Y)] = p
		if p.X >= width {
			width = p.X + 1
		}
		if p.Y >= height {
			height = p.Y + 1
		}
	}
	data.Placement.GridWidth = width
	for x := 0; x < width; x++ {
		data.Placement.XHeaders = append(data.Placement.XHeaders, x)
	}
	for y := 0; y < height; y++ {
		row := spatialGridRow{Y: y}
		for x := 0; x < width; x++ {
			cell := spatialGridCell{X: x, Y: y}
			if p, ok := cells[fmt.Sprintf("%d,%d", x, y)]; ok {
				cell.EntityID, cell.EntityName, cell.Occupied = p.EntityID, entityByID[p.EntityID].Name, true
			}
			row.Cells = append(row.Cells, cell)
		}
		data.Placement.Rows = append(data.Placement.Rows, row)
	}
	return data
}

func (h *editorHandler) spatialCellOccupied(screen placementScreen, parentID, exceptID string, x, y int) (bool, error) {
	ps, err := screen.listPlacements()
	if err != nil {
		return false, err
	}
	for _, p := range ps {
		if p.ParentID == parentID && p.EntityID != exceptID && p.X == x && p.Y == y {
			return true, nil
		}
	}
	return false, nil
}
func (h *editorHandler) spatialPlacementMatches(screen placementScreen, entityID, parentID string) (bool, error) {
	ps, err := screen.listPlacements()
	if err != nil {
		return false, err
	}
	for _, p := range ps {
		if p.EntityID == entityID {
			return p.ParentID == parentID, nil
		}
	}
	return false, nil
}

func (h *editorHandler) roomIsEntryElsewhere(roomID, allowedLocationID string) bool {
	entries, err := h.locationEntries.List()
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.RoomID == roomID && e.LocationID != allowedLocationID {
			return true
		}
	}
	return false
}

func (h *editorHandler) validateSpatialRelationships() error {
	hubs, err := h.hubs.List()
	if err != nil {
		return err
	}
	locations, err := h.locations.List()
	if err != nil {
		return err
	}
	rooms, err := h.rooms.List()
	if err != nil {
		return err
	}
	lp, err := h.locationPlacements.List()
	if err != nil {
		return err
	}
	rp, err := h.roomPlacements.List()
	if err != nil {
		return err
	}
	entries, err := h.locationEntries.List()
	if err != nil {
		return err
	}
	locationAssignments, err := h.locationAssignments.List()
	if err != nil {
		return err
	}
	roomAssignments, err := h.roomAssignments.List()
	if err != nil {
		return err
	}
	hubIDs, locationIDs, roomIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, v := range hubs {
		hubIDs[v.ID] = true
	}
	for _, v := range locations {
		locationIDs[v.ID] = true
	}
	for _, v := range rooms {
		roomIDs[v.ID] = true
	}
	locationParent, roomParent, roomPlacementParent := map[string]string{}, map[string]string{}, map[string]string{}
	for _, a := range locationAssignments {
		if !locationIDs[a.LocationID] {
			return fmt.Errorf("assignment references missing location %s", a.LocationID)
		}
		if !hubIDs[a.HubID] {
			return fmt.Errorf("location %s assignment references missing hub %s", a.LocationID, a.HubID)
		}
		locationParent[a.LocationID] = a.HubID
	}
	for _, a := range roomAssignments {
		if !roomIDs[a.RoomID] {
			return fmt.Errorf("assignment references missing room %s", a.RoomID)
		}
		if !locationIDs[a.LocationID] {
			return fmt.Errorf("room %s assignment references missing location %s", a.RoomID, a.LocationID)
		}
		roomParent[a.RoomID] = a.LocationID
	}
	for _, p := range lp {
		if !locationIDs[p.LocationID] {
			return fmt.Errorf("placement references missing location %s", p.LocationID)
		}
		if !hubIDs[p.HubID] {
			return fmt.Errorf("location %s references missing hub %s", p.LocationID, p.HubID)
		}
		if locationParent[p.LocationID] != p.HubID {
			return fmt.Errorf("location %s placement does not match its Hub assignment", p.LocationID)
		}
	}
	for _, p := range rp {
		if !roomIDs[p.RoomID] {
			return fmt.Errorf("placement references missing room %s", p.RoomID)
		}
		if !locationIDs[p.LocationID] {
			return fmt.Errorf("room %s references missing location %s", p.RoomID, p.LocationID)
		}
		if roomParent[p.RoomID] != p.LocationID {
			return fmt.Errorf("room %s placement does not match its Location assignment", p.RoomID)
		}
		roomPlacementParent[p.RoomID] = p.LocationID
	}
	for _, e := range entries {
		if !locationIDs[e.LocationID] {
			return fmt.Errorf("entry references missing location %s", e.LocationID)
		}
		if !roomIDs[e.RoomID] {
			return fmt.Errorf("location %s entry references missing room %s", e.LocationID, e.RoomID)
		}
		if roomParent[e.RoomID] != e.LocationID || roomPlacementParent[e.RoomID] != e.LocationID {
			return fmt.Errorf("location %s entry room is not placed in that location", e.LocationID)
		}
	}
	return nil
}

func (h *editorHandler) roomIsPlaced(id string) (bool, error) {
	ps, err := h.roomPlacements.List()
	if err != nil {
		return false, err
	}
	for _, p := range ps {
		if p.RoomID == id {
			return true, nil
		}
	}
	return false, nil
}
func (h *editorHandler) roomIsAssigned(id string) (bool, error) {
	assignments, err := h.roomAssignments.List()
	if err != nil {
		return false, err
	}
	for _, assignment := range assignments {
		if assignment.RoomID == id {
			return true, nil
		}
	}
	return false, nil
}
func (h *editorHandler) locationIsPlaced(id string) (bool, error) {
	ps, err := h.locationPlacements.List()
	if err != nil {
		return false, err
	}
	for _, p := range ps {
		if p.LocationID == id {
			return true, nil
		}
	}
	return false, nil
}
func (h *editorHandler) locationIsAssigned(id string) (bool, error) {
	assignments, err := h.locationAssignments.List()
	if err != nil {
		return false, err
	}
	for _, assignment := range assignments {
		if assignment.LocationID == id {
			return true, nil
		}
	}
	return false, nil
}
func (h *editorHandler) locationHasRooms(id string) (bool, error) {
	ps, err := h.roomAssignments.List()
	if err != nil {
		return false, err
	}
	for _, p := range ps {
		if p.LocationID == id {
			return true, nil
		}
	}
	return false, nil
}
func (h *editorHandler) hubHasLocations(id string) (bool, error) {
	ps, err := h.locationAssignments.List()
	if err != nil {
		return false, err
	}
	for _, p := range ps {
		if p.HubID == id {
			return true, nil
		}
	}
	return false, nil
}
