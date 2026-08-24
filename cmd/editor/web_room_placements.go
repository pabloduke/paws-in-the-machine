package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const roomPlacementBasePath = "/place/rooms"
const defaultRoomGridSize = 10

type placementHubOption struct {
	ID, Name string
	Selected bool
}

type placementRoomOption struct {
	ID, Name, Location string
}

type roomGridCell struct {
	X, Y             int
	RoomID, RoomName string
	Occupied         bool
}

type roomGridRow struct {
	Y     int
	Cells []roomGridCell
}

type roomPlacementPage struct {
	Hubs            []placementHubOption
	Rooms           []placementRoomOption
	SelectedHubID   string
	SelectedHubName string
	XHeaders        []int
	Rows            []roomGridRow
	GridWidth       int
	Notice          string
	GeneralError    string
	CoordinateError string
}

func (h *editorHandler) serveRoomPlacements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.roomPlacementsPage(r.URL.Query().Get("hub_id"), "", "", "")
	h.renderPage(w, r, data)
}

func (h *editorHandler) placeRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid room placement form", http.StatusBadRequest)
		return
	}
	hubID, roomID := r.FormValue("hub_id"), r.FormValue("room_id")
	x, y, coordinateErr := placementCoordinates(r)
	var notice, generalError string
	if coordinateErr == "" {
		if _, err := h.hubs.Get(hubID); errors.Is(err, errHubNotFound) {
			generalError = "Choose an existing Hub."
		} else if err != nil {
			generalError = err.Error()
		} else if room, err := h.rooms.Get(roomID); errors.Is(err, errDescribedEntityNotFound) {
			generalError = "Choose an existing Room."
		} else if err != nil {
			generalError = err.Error()
		} else if occupied, err := h.roomCellOccupied(hubID, roomID, x, y); err != nil {
			generalError = err.Error()
		} else if occupied {
			coordinateErr = "That cell is already occupied by another Room."
		} else if _, err := h.placements.Place(roomID, hubID, x, y); err != nil {
			generalError = err.Error()
		} else {
			notice = fmt.Sprintf("Placed %s at %d, %d.", room.Name, x, y)
		}
	}
	data := h.roomPlacementsPage(hubID, notice, generalError, coordinateErr)
	h.renderRoomPlacementResponse(w, r, data)
}

func (h *editorHandler) unassignRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid room placement form", http.StatusBadRequest)
		return
	}
	hubID, roomID := r.FormValue("hub_id"), r.FormValue("unassign_room_id")
	room, roomErr := h.rooms.Get(roomID)
	var notice, generalError string
	if roomErr != nil && !errors.Is(roomErr, errDescribedEntityNotFound) {
		generalError = roomErr.Error()
	} else if matches, err := h.roomPlacementMatches(roomID, hubID); err != nil {
		generalError = err.Error()
	} else if !matches {
		http.NotFound(w, r)
		return
	} else if _, err := h.placements.Unassign(roomID); err != nil {
		generalError = err.Error()
	} else if roomErr == nil {
		notice = "Unassigned " + room.Name + "."
	} else {
		notice = "Room unassigned."
	}
	data := h.roomPlacementsPage(hubID, notice, generalError, "")
	h.renderRoomPlacementResponse(w, r, data)
}

func (h *editorHandler) roomPlacementMatches(roomID, hubID string) (bool, error) {
	placements, err := h.placements.List()
	if err != nil {
		return false, err
	}
	for _, placement := range placements {
		if placement.RoomID == roomID {
			return placement.HubID == hubID, nil
		}
	}
	return false, nil
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

func (h *editorHandler) renderRoomPlacementResponse(w http.ResponseWriter, r *http.Request, data pageData) {
	if isHTMX(r) {
		if data.Placement.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
		}
		h.render(w, "room-placement-body", data)
		return
	}
	if data.Placement.Notice != "" {
		http.Redirect(w, r, roomPlacementBasePath+"?hub_id="+url.QueryEscape(data.Placement.SelectedHubID)+"&saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) roomPlacementsPage(hubID, notice, generalError, coordinateError string) pageData {
	data := basePage("place", "rooms", "Assign Rooms", "room-placement")
	data.Placement = roomPlacementPage{Notice: notice, GeneralError: generalError, CoordinateError: coordinateError}
	if err := h.validateRoomPlacementRelationships(); err != nil {
		data.StoreError = err.Error()
		return data
	}
	hubs, err := h.hubs.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	sort.SliceStable(hubs, func(i, j int) bool { return strings.ToLower(hubs[i].Name) < strings.ToLower(hubs[j].Name) })
	if hubID == "" && len(hubs) != 0 {
		hubID = hubs[0].ID
	}
	for _, hub := range hubs {
		selected := hub.ID == hubID
		data.Placement.Hubs = append(data.Placement.Hubs, placementHubOption{ID: hub.ID, Name: hub.Name, Selected: selected})
		if selected {
			data.Placement.SelectedHubID, data.Placement.SelectedHubName = hub.ID, hub.Name
		}
	}
	if hubID != "" && data.Placement.SelectedHubID == "" {
		data.Placement.GeneralError = "The selected Hub no longer exists."
		return data
	}
	rooms, err := h.rooms.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	placements, err := h.placements.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	roomByID := make(map[string]gamecontent.Room, len(rooms))
	placementByRoom := make(map[string]gamecontent.RoomPlacement, len(placements))
	for _, room := range rooms {
		roomByID[room.ID] = room
	}
	for _, placement := range placements {
		placementByRoom[placement.RoomID] = placement
	}
	sort.SliceStable(rooms, func(i, j int) bool { return strings.ToLower(rooms[i].Name) < strings.ToLower(rooms[j].Name) })
	for _, room := range rooms {
		option := placementRoomOption{ID: room.ID, Name: room.Name, Location: "Unassigned"}
		if placement, ok := placementByRoom[room.ID]; ok {
			hubName := "Unknown Hub"
			for _, hub := range hubs {
				if hub.ID == placement.HubID {
					hubName = hub.Name
					break
				}
			}
			option.Location = fmt.Sprintf("%s at %d,%d", hubName, placement.X, placement.Y)
		}
		data.Placement.Rooms = append(data.Placement.Rooms, option)
	}
	width, height := defaultRoomGridSize, defaultRoomGridSize
	cellByCoordinate := make(map[string]gamecontent.RoomPlacement)
	for _, placement := range placements {
		if placement.HubID != hubID {
			continue
		}
		cellByCoordinate[fmt.Sprintf("%d,%d", placement.X, placement.Y)] = placement
		if placement.X >= width {
			width = placement.X + 1
		}
		if placement.Y >= height {
			height = placement.Y + 1
		}
	}
	data.Placement.GridWidth = width
	for x := 0; x < width; x++ {
		data.Placement.XHeaders = append(data.Placement.XHeaders, x)
	}
	for y := 0; y < height; y++ {
		row := roomGridRow{Y: y}
		for x := 0; x < width; x++ {
			cell := roomGridCell{X: x, Y: y}
			if placement, ok := cellByCoordinate[fmt.Sprintf("%d,%d", x, y)]; ok {
				cell.RoomID, cell.Occupied = placement.RoomID, true
				cell.RoomName = roomByID[placement.RoomID].Name
			}
			row.Cells = append(row.Cells, cell)
		}
		data.Placement.Rows = append(data.Placement.Rows, row)
	}
	return data
}

func (h *editorHandler) roomCellOccupied(hubID, exceptRoomID string, x, y int) (bool, error) {
	placements, err := h.placements.List()
	if err != nil {
		return false, err
	}
	for _, placement := range placements {
		if placement.HubID == hubID && placement.RoomID != exceptRoomID && placement.X == x && placement.Y == y {
			return true, nil
		}
	}
	return false, nil
}

func (h *editorHandler) roomIsPlaced(roomID string) (bool, error) {
	placements, err := h.placements.List()
	if err != nil {
		return false, err
	}
	for _, placement := range placements {
		if placement.RoomID == roomID {
			return true, nil
		}
	}
	return false, nil
}

func (h *editorHandler) hubHasRooms(hubID string) (bool, error) {
	placements, err := h.placements.List()
	if err != nil {
		return false, err
	}
	for _, placement := range placements {
		if placement.HubID == hubID {
			return true, nil
		}
	}
	return false, nil
}

func (h *editorHandler) validateRoomPlacementRelationships() error {
	hubs, err := h.hubs.List()
	if err != nil {
		return err
	}
	rooms, err := h.rooms.List()
	if err != nil {
		return err
	}
	placements, err := h.placements.List()
	if err != nil {
		return err
	}
	hubIDs := make(map[string]bool, len(hubs))
	roomIDs := make(map[string]bool, len(rooms))
	for _, hub := range hubs {
		hubIDs[hub.ID] = true
	}
	for _, room := range rooms {
		roomIDs[room.ID] = true
	}
	for _, placement := range placements {
		if !roomIDs[placement.RoomID] {
			return fmt.Errorf("placement references missing room %s", placement.RoomID)
		}
		if !hubIDs[placement.HubID] {
			return fmt.Errorf("room %s references missing hub %s", placement.RoomID, placement.HubID)
		}
	}
	return nil
}
