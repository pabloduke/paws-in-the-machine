package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
)

type crumb struct{ Name, URL string }
type drillChild struct{ ID, Name, URL string }
type drillGroup struct {
	Kind, Label string
	Held        []contentsRow
	Choices     []contentsChoice
}
type drillVertical struct{ LowerID, UpperID, LowerName, UpperName string }
type drillPage struct {
	Passages                         []passageRow
	IsEntrance                       bool
	InteriorOpen                     bool
	Level, Z                         int
	Levels                           []int
	Vertical                         []drillVertical
	VerticalChoices                  []drillChild
	Kind, ID, Name, Description, URL string
	Breadcrumbs                      []crumb
	ChildKind, ChildLabel, ChildBase string
	Grid                             roomPlacementPage
	Unplaced                         []drillChild
	Unassigned                       []describedFields
	Groups                           []drillGroup
	ParentID, ParentURL, Placement   string
	Parents                          []placementOption
	Placed                           bool
	X, Y                             int
	CellSelected                     bool
	CellX, CellY                     string
	EntryID                          string
	Entries                          []placementOption
	Arrival                          playPage
	Error, Notice, Action            string
	Values                           url.Values
	ThingKind, ThingID               string
}

func entityURL(kind, id string) string {
	switch kind {
	case "hub":
		return "/content/hubs/" + id
	case "location":
		return "/content/locations/" + id
	case "room":
		return "/content/rooms/" + id
	}
	return ""
}

// Canonical spatial pages are contextual. Legacy detail forms remain available
// through ?details=1; old mutation and placement endpoints keep working.
func (h *editorHandler) routeDrill(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/library" {
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return true
		}
		data := basePage("library", "", "Library", "library")
		snapshot, err := h.worldSnapshot()
		if err != nil {
			data.StoreError = err.Error()
		} else {
			data.Overview.Advisories = advisoryGroups(snapshot)
			for _, l := range snapshot.Unrooted {
				data.LibraryLoose = append(data.LibraryLoose, crumb{l.Name, entityURL("location", l.ID)})
			}
			for _, room := range snapshot.OrphanRooms {
				data.LibraryLoose = append(data.LibraryLoose, crumb{room.Name, entityURL("room", room.ID)})
			}
		}
		h.renderPage(w, r, data)
		return true
	}
	if r.URL.Path == "/content/hubs" && r.Method == http.MethodGet && r.URL.Query().Get("details") != "1" {
		data := basePage("hubs", "", "Hubs", "hub-home")
		hubs, err := h.hubs.List()
		if err != nil {
			data.StoreError = err.Error()
		}
		sort.Slice(hubs, func(i, j int) bool { return strings.ToLower(hubs[i].Name) < strings.ToLower(hubs[j].Name) })
		for _, hub := range hubs {
			data.Hubs = append(data.Hubs, hubRow{ID: hub.ID, Name: hub.Name})
		}
		h.renderDrillPage(w, r, data)
		return true
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "content" || parts[2] == "new" || parts[2] == "list" {
		return false
	}
	kind := ""
	switch parts[1] {
	case "hubs":
		kind = "hub"
	case "locations":
		kind = "location"
	case "rooms":
		kind = "room"
	default:
		return false
	}
	if len(parts) == 4 && parts[3] == "work" {
		if r.Method != http.MethodPost {
			h.methodNotAllowed(w, http.MethodPost)
			return true
		}
		h.mutateDrill(w, r, kind, parts[2])
		return true
	}
	if len(parts) != 3 || r.Method != http.MethodGet || r.URL.Query().Get("details") == "1" {
		return false
	}
	h.serveDrill(w, r, kind, parts[2], "", "", nil)
	return true
}
func (h *editorHandler) renderDrillPage(w http.ResponseWriter, r *http.Request, data pageData) {
	if isHTMX(r) {
		w.Header().Set("HX-Retarget", "#workspace")
		w.Header().Set("HX-Reswap", "innerHTML")
		if r.Method == http.MethodGet {
			if data.Drill.ThingKind != "" {
				w.Header().Set("HX-Reswap", "innerHTML show:#thing-editor:top")
			} else if data.Drill.CellSelected {
				w.Header().Set("HX-Reswap", "innerHTML show:#cell-editor:top")
			}
		}
	}
	h.renderPage(w, r, data)
}
func (h *editorHandler) spatialDefinition(kind, id string) (describedFields, error) {
	switch kind {
	case "hub":
		v, err := h.hubs.Get(id)
		return describedFields{ID: v.ID, Name: v.Name}, err
	case "location":
		v, err := h.locations.Get(id)
		return fieldsFromLocation(v), err
	case "room":
		v, err := h.rooms.Get(id)
		return fieldsFromRoom(v), err
	}
	return describedFields{}, fmt.Errorf("unknown spatial kind")
}
func (h *editorHandler) spatialParent(kind, id string) (string, error) {
	if kind == "location" {
		records, err := h.locationAssignments.List()
		for _, a := range records {
			if a.LocationID == id {
				return a.HubID, err
			}
		}
		return "", err
	}
	if kind == "room" {
		records, err := h.roomAssignments.List()
		for _, a := range records {
			if a.RoomID == id {
				return a.LocationID, err
			}
		}
		return "", err
	}
	return "", nil
}
func (h *editorHandler) drillCrumbs(kind, id string) ([]crumb, error) {
	v, err := h.spatialDefinition(kind, id)
	if err != nil {
		return nil, err
	}
	if kind == "hub" {
		return []crumb{{"Hubs", "/content/hubs"}, {v.Name, entityURL(kind, id)}}, nil
	}
	parent, err := h.spatialParent(kind, id)
	if err != nil {
		return nil, err
	}
	if parent == "" {
		return []crumb{{"Library", "/library"}, {"Unassigned " + kind, strings.TrimSuffix(entityURL(kind, id), "/"+id)}, {v.Name, entityURL(kind, id)}}, nil
	}
	parentKind := "hub"
	if kind == "room" {
		parentKind = "location"
	}
	crumbs, err := h.drillCrumbs(parentKind, parent)
	if err != nil {
		return nil, err
	}
	screen := h.locationPlacementScreen()
	if kind == "room" {
		screen = h.roomPlacementScreen()
	}
	ps, e := screen.listPlacements()
	if e != nil {
		return nil, e
	}
	for _, p := range ps {
		if p.EntityID == id && p.Z != 0 {
			crumbs[len(crumbs)-1].URL += "?z=" + strconv.Itoa(p.Z)
		}
	}
	return append(crumbs, crumb{v.Name, entityURL(kind, id)}), nil
}
func (h *editorHandler) serveDrill(w http.ResponseWriter, r *http.Request, kind, id, notice, problem string, values url.Values) {
	v, err := h.spatialDefinition(kind, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	d := drillPage{Kind: kind, ID: id, Name: v.Name, Description: v.Description, URL: entityURL(kind, id), Notice: notice, Error: problem, Values: values}
	if d.Values == nil {
		d.Values = url.Values{}
	}
	if r.URL.Query().Get("saved") == "1" {
		d.Notice = "Saved."
	}
	d.InteriorOpen = r.URL.Query().Has("z")
	d.Level, _ = strconv.Atoi(r.URL.Query().Get("z"))
	if values != nil && values.Has("z") {
		d.Level, _ = strconv.Atoi(values.Get("z"))
	}
	d.Action = d.Values.Get("action")
	d.Breadcrumbs, err = h.drillCrumbs(kind, id)
	if err != nil {
		d.Error = err.Error()
	}
	d.ParentID, err = h.spatialParent(kind, id)
	if err != nil {
		d.Error = err.Error()
	}
	if kind != "hub" {
		screen := h.locationPlacementScreen()
		parentKind := "hub"
		if kind == "room" {
			screen = h.roomPlacementScreen()
			parentKind = "location"
		}
		d.ParentURL = entityURL(parentKind, d.ParentID)
		parents, err := screen.listParents()
		if err != nil {
			d.Error = err.Error()
		}
		for _, p := range parents {
			d.Parents = append(d.Parents, placementOption{ID: p.ID, Name: p.Name, Selected: p.ID == d.ParentID})
		}
		placements, err := screen.listPlacements()
		if err != nil {
			d.Error = err.Error()
		}
		d.Placement = "Unplaced"
		for _, p := range placements {
			if p.EntityID == id {
				d.Placed = true
				d.X = p.X
				d.Y = p.Y
				d.Z = p.Z
				if p.Z != 0 {
					d.ParentURL += "?z=" + strconv.Itoa(p.Z)
				}
				d.Placement = fmt.Sprintf("%d, %d (level %d)", p.X, p.Y, p.Z)
			}
		}
	}
	if kind != "hub" && d.Placed {
		vs, e := h.verticalStore().List()
		if e != nil {
			d.Error = e.Error()
		}
		for _, v := range vs {
			if v.Kind == kind && (v.LowerID == id || v.UpperID == id) {
				lower, _ := h.spatialDefinition(kind, v.LowerID)
				upper, _ := h.spatialDefinition(kind, v.UpperID)
				d.Vertical = append(d.Vertical, drillVertical{v.LowerID, v.UpperID, lower.Name, upper.Name})
			}
		}
		screen := h.locationPlacementScreen()
		if kind == "room" {
			screen = h.roomPlacementScreen()
		}
		ps, _ := screen.listPlacements()
		for _, p := range ps {
			if p.ParentID == d.ParentID && p.X == d.X && p.Y == d.Y && p.Z == d.Z+1 {
				v, e := screen.getEntity(p.EntityID)
				if e == nil {
					d.VerticalChoices = append(d.VerticalChoices, drillChild{v.ID, v.Name, entityURL(kind, v.ID)})
				}
			}
		}
	}

	if kind != "room" {
		screen := h.locationPlacementScreen()
		ownership := h.hubLocationOwnership()
		d.ChildKind = "location"
		d.ChildLabel = "Location"
		d.ChildBase = "/content/locations/"
		if kind == "location" {
			screen = h.roomPlacementScreen()
			ownership = h.locationRoomOwnership()
			d.ChildKind = "room"
			d.ChildLabel = "Room"
			d.ChildBase = "/content/rooms/"
		}
		screen.level = d.Level
		ps, _ := screen.listPlacements()
		levels := map[int]bool{0: true, d.Level: true}
		for _, p := range ps {
			if p.ParentID == id {
				levels[p.Z] = true
			}
		}
		for z := range levels {
			d.Levels = append(d.Levels, z)
		}
		sort.Ints(d.Levels)
		grid := h.spatialPlacementsPage(screen, id, "", "", "")
		d.Grid = grid.Placement
		if grid.StoreError != "" {
			d.Error = grid.StoreError
		}
		if d.Grid.GeneralError != "" {
			d.Error = d.Grid.GeneralError
		}
		for _, child := range d.Grid.Entities {
			if child.Placement == "Unplaced" {
				d.Unplaced = append(d.Unplaced, drillChild{child.ID, child.Name, entityURL(d.ChildKind, child.ID)})
			}
		}
		assignments, err := ownership.listAssignments()
		if err != nil {
			d.Error = err.Error()
		}
		children, err := ownership.listChildren()
		if err != nil {
			d.Error = err.Error()
		}
		for _, child := range children {
			if assignments[child.ID] == "" {
				d.Unassigned = append(d.Unassigned, child)
			}
		}
		x, y := r.URL.Query().Get("x"), r.URL.Query().Get("y")
		if values != nil && (values.Get("action") == "create-child" || values.Get("action") == "place-child") {
			x, y = values.Get("x"), values.Get("y")
		}
		if x != "" || y != "" {
			d.CellSelected = true
			d.CellX = x
			d.CellY = y
		}
		if kind == "hub" {
			d.Arrival = h.playPage(id)
		} else {
			entries, err := h.locationEntries.List()
			if err != nil {
				d.Error = err.Error()
			}
			for _, e := range entries {
				if e.LocationID == id {
					d.EntryID = e.RoomID
				}
			}
			for _, e := range d.Grid.Entities {
				if e.Placement != "Unplaced" {
					d.Entries = append(d.Entries, placementOption{ID: e.ID, Name: e.Name, Selected: e.ID == d.EntryID})
				}
			}
		}
	}
	h.passagePanel(&d)
	if kind != "hub" {
		for _, screen := range []contentsScreen{h.worldItemContentsScreen(), h.npcContentsScreen(), h.terminalContentsScreen()} {
			data := h.contentsPageData(screen, kind+":"+id, "", "")
			if data.StoreError != "" {
				d.Error = data.StoreError
			}
			d.Groups = append(d.Groups, drillGroup{screen.entityKind, screen.plural, data.Contents.Held, data.Contents.Choices})
		}
		if values == nil {
			if k := r.URL.Query().Get("new"); k == "world_item" || k == "npc" || k == "terminal" {
				d.ThingKind = k
				d.Values.Set("item_kind", "takeable")
			}
			if thing := r.URL.Query().Get("thing"); thing != "" {
				k, tid, ok := strings.Cut(thing, ":")
				if !ok {
					d.Error = "Choose an item, NPC, or terminal in this cell."
				} else if err := h.populateDrillThing(&d, k, tid); err != nil {
					d.Error = err.Error()
				}
			}
		} else if d.Action == "save-thing" {
			d.ThingKind = values.Get("thing_kind")
			d.ThingID = values.Get("thing_id")
		}
	}
	data := basePage("hubs", "", v.Name, "drill")
	data.Drill = d
	if len(d.Breadcrumbs) > 0 && d.Breadcrumbs[0].Name == "Library" {
		data.HeaderTabs = headerTabs("library")
	}
	h.renderDrillPage(w, r, data)
}

func (h *editorHandler) populateDrillThing(d *drillPage, kind, id string) error {
	if err := h.requireHeld(kind, id, d.Kind, d.ID); err != nil {
		return err
	}
	d.ThingKind = kind
	d.ThingID = id
	switch kind {
	case "world_item":
		v, err := h.items.Get(id)
		if err != nil {
			return err
		}
		d.Values.Set("thing_name", v.Name)
		d.Values.Set("item_kind", string(v.Kind))
		d.Values.Set("short_description", v.ShortDescription)
		d.Values.Set("full_description", v.FullDescription)
	case "npc":
		v, err := h.npcs.Get(id)
		if err != nil {
			return err
		}
		d.Values.Set("thing_name", v.Name)
		d.Values.Set("thing_description", v.Description)
	case "terminal":
		v, err := h.terminals.Get(id)
		if err != nil {
			return err
		}
		d.Values.Set("host_name", v.HostName)
	default:
		return fmt.Errorf("unknown content kind")
	}
	return nil
}
func (h *editorHandler) requireHeld(kind, id, parentKind, parentID string) error {
	p, held, err := h.contents.ContainerOf(kind, id)
	if err != nil {
		return err
	}
	if !held || p.ParentKind != parentKind || p.ParentID != parentID {
		return fmt.Errorf("this entity is no longer in this cell; reload before editing or removing it")
	}
	return nil
}

func (h *editorHandler) mutateDrill(w http.ResponseWriter, r *http.Request, kind, id string) {
	if _, err := h.spatialDefinition(kind, id); err != nil {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form.", http.StatusBadRequest)
		return
	}
	if err := h.validateEditorRelationships(); err != nil {
		h.serveDrill(w, r, kind, id, "", err.Error(), r.PostForm)
		return
	}
	target, err := h.applyDrillAction(kind, id, r)
	if err != nil {
		h.serveDrill(w, r, kind, id, "", err.Error(), r.PostForm)
		return
	}
	if target == "" {
		target = entityURL(kind, id)
		if r.FormValue("action") == "move-child" {
			target += "?z=" + url.QueryEscape(r.FormValue("z"))
		}
	}
	if r.FormValue("return") == "hubs" && kind == "hub" {
		target = "/content/hubs"
	}
	h.redirectDrill(w, r, target)
}

func (h *editorHandler) redirectDrill(w http.ResponseWriter, r *http.Request, target string) {
	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "contentSaved")
		location, _ := json.Marshal(map[string]string{"path": drillSavedURL(target), "target": "#workspace", "swap": "innerHTML"})
		w.Header().Set("HX-Location", string(location))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, drillSavedURL(target), http.StatusSeeOther)
}

func (h *editorHandler) applyDrillAction(kind, id string, r *http.Request) (string, error) {
	action := r.FormValue("action")
	switch action {
	case "delete-self":
		parent, _ := h.spatialParent(kind, id)
		if err := h.deleteEntity(kind, id); err != nil {
			return "", err
		}
		if kind == "hub" {
			return "/content/hubs", nil
		}
		if kind == "room" && parent != "" {
			return entityURL("location", parent), nil
		}
		return "/library", nil
	case "save-details":
		if kind == "hub" {
			_, err := h.hubs.Update(id, r.FormValue("name"))
			return "", err
		}
		input := describedInput{Name: r.FormValue("name"), Description: r.FormValue("description")}.normalized()
		if input.Name == "" || input.Description == "" {
			return "", fmt.Errorf("Name and Description are required.")
		}
		screen := h.locationScreen()
		if kind == "room" {
			screen = h.roomScreen()
		}
		_, err := screen.update(id, input)
		return "", err
	case "create-child", "place-child", "assign-child", "move-child":
		if kind == "room" {
			return "", fmt.Errorf("Rooms do not own a grid in this editor.")
		}
		return h.drillChildAction(kind, id, r)
	case "position", "unplace", "assign-parent", "unassign-parent":
		if kind == "hub" {
			return "", fmt.Errorf("Hubs do not have a spatial parent.")
		}
		return "", h.drillPositionAction(kind, id, r)
	case "connect-vertical", "disconnect-vertical":
		if kind == "hub" {
			return "", fmt.Errorf("Choose a Location or Room for a vertical connection.")
		}
		if r.FormValue("action") == "disconnect-vertical" {
			lower := r.FormValue("lower_id")
			vs, e := h.verticalStore().List()
			if e != nil {
				return "", e
			}
			for _, v := range vs {
				if v.Kind == kind && v.LowerID == lower && (v.LowerID == id || v.UpperID == id) {
					_, e = h.verticalStore().Delete(kind + ":" + lower)
					return "", e
				}
			}
			return "", fmt.Errorf("Connection no longer belongs to this cell.")
		}
		return "", h.connectVertical(kind, id, r.FormValue("upper_id"))
	case "arrival":
		if kind != "hub" {
			return "", fmt.Errorf("Arrival belongs to a Hub.")
		}
		loc := r.FormValue("arrival_id")
		if loc != "" {
			valid := false
			for _, p := range h.playPage(id).Options {
				if p.Value == loc {
					valid = true
				}
			}
			if !valid {
				return "", fmt.Errorf("Choose a placed Location in this Hub.")
			}
		}
		return "", h.playSettings.Update(func(s *content.PlaySettings) error {
			if loc == "" {
				delete(s.HubArrivals, id)
			} else {
				s.HubArrivals[id] = loc
			}
			return nil
		})
	case "set-passage":
		return "", h.setPassage(kind, id, r.FormValue("other_id"), r.FormValue("blocked"), r.FormValue("expected_blocked"))
	case "entry":
		if kind != "location" {
			return "", fmt.Errorf("Entry belongs to a Location.")
		}
		room := r.FormValue("entry_id")
		if room == "" {
			entries, err := h.locationEntries.List()
			if err != nil {
				return "", err
			}
			for _, e := range entries {
				if e.LocationID == id {
					_, err = h.locationEntries.Clear(id)
					return "", err
				}
			}
			return "", nil
		}
		ok, err := h.spatialPlacementMatches(h.roomPlacementScreen(), room, id)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("Choose a Room placed in this Location.")
		}
		_, err = h.locationEntries.Set(id, room)
		return "", err
	case "add-thing", "remove-thing", "save-thing":
		if kind == "hub" {
			return "", fmt.Errorf("Contents belong to a Location or Room.")
		}
		return "", h.drillThingAction(kind, id, r)
	}
	return "", fmt.Errorf("Unknown editor action.")
}

func (h *editorHandler) drillChildAction(kind, id string, r *http.Request) (string, error) {
	screen := h.locationPlacementScreen()
	owner := h.hubLocationOwnership()
	childKind := "location"
	if kind == "location" {
		screen = h.roomPlacementScreen()
		owner = h.locationRoomOwnership()
		childKind = "room"
	}
	z, err := levelCoordinate(r)
	if err != nil {
		return "", err
	}
	action := r.FormValue("action")
	childID := r.FormValue("child_id")
	if action == "move-child" {
		assignments, err := owner.listAssignments()
		if err != nil {
			return "", err
		}
		if assignments[childID] != id {
			return "", fmt.Errorf("This child does not belong to the selected parent.")
		}
		sx, ex := strconv.Atoi(r.FormValue("source_x"))
		sy, ey := strconv.Atoi(r.FormValue("source_y"))
		placements, err := screen.listPlacements()
		if err != nil {
			return "", err
		}
		matches := false
		for _, p := range placements {
			if p.EntityID == childID && p.ParentID == id && ex == nil && ey == nil && p.X == sx && p.Y == sy && p.Z == z {
				matches = true
			}
		}
		if !matches {
			return "", fmt.Errorf("This cell has moved or is no longer placed here. Refresh and try again.")
		}
		x, y, problem := placementCoordinates(r)
		if problem != "" {
			return "", fmt.Errorf("%s", problem)
		}
		occupied, err := h.spatialCellOccupied(screen, id, childID, x, y, z)
		if err != nil {
			return "", err
		}
		if occupied {
			return "", fmt.Errorf("That cell is occupied. Choose an empty cell.")
		}
		return "", screen.place(childID, id, x, y, z)
	}
	wantsPlace := action == "place-child" || (action == "create-child" && (r.FormValue("x") != "" || r.FormValue("y") != ""))
	x, y := 0, 0
	if wantsPlace {
		var problem string
		x, y, problem = placementCoordinates(r)
		if problem != "" {
			return "", fmt.Errorf("%s", problem)
		}
		occupied, err := h.spatialCellOccupied(screen, id, "", x, y, z)
		if err != nil {
			return "", err
		}
		if occupied {
			return "", fmt.Errorf("That cell is occupied. Choose an empty cell.")
		}
	}
	if action == "create-child" {
		input := describedInput{Name: r.FormValue("child_name"), Description: r.FormValue("child_description")}.normalized()
		if input.Name == "" || input.Description == "" {
			return "", fmt.Errorf("Name and Description are required.")
		}
		child, err := owner.createChild(input)
		if err != nil {
			return "", err
		}
		childID = child.ID
		if err = owner.assign(childID, id); err != nil {
			return "", fmt.Errorf("Created %s, but assignment failed; find it in Library: %w", child.Name, err)
		}
	} else {
		if _, err := owner.getChild(childID); err != nil {
			return "", fmt.Errorf("Choose an existing %s.", screen.entityLabel)
		}
		assignments, err := owner.listAssignments()
		if err != nil {
			return "", err
		}
		if action == "assign-child" {
			if assignments[childID] != "" {
				return "", fmt.Errorf("Choose an unassigned %s.", screen.entityLabel)
			}
			return "", owner.assign(childID, id)
		}
		if assignments[childID] != id {
			return "", fmt.Errorf("This child does not belong to the selected parent.")
		}
		placed, err := owner.isPlaced(childID)
		if err != nil {
			return "", err
		}
		if placed {
			return "", fmt.Errorf("Use the child's Reposition control to move a placed cell.")
		}
	}
	if wantsPlace {
		if err := screen.place(childID, id, x, y, z); err != nil {
			return "", fmt.Errorf("%s remains assigned but unplaced: %w", screen.entityLabel, err)
		}
	}
	return entityURL(childKind, childID), nil
}

func (h *editorHandler) drillPositionAction(kind, id string, r *http.Request) error {
	screen := h.locationPlacementScreen()
	owner := h.hubLocationOwnership()
	if kind == "room" {
		screen = h.roomPlacementScreen()
		owner = h.locationRoomOwnership()
	}
	parent, err := h.spatialParent(kind, id)
	if err != nil {
		return err
	}
	action := r.FormValue("action")
	if action == "assign-parent" || action == "unassign-parent" {
		placed, err := owner.isPlaced(id)
		if err != nil {
			return err
		}
		if placed {
			return fmt.Errorf("Unplace this %s before changing its parent.", kind)
		}
		if err = h.playReferenceProblem(kind, id); err != nil {
			return err
		}
		if action == "unassign-parent" {
			if parent == "" {
				return nil
			}
			return owner.unassign(id)
		}
		selected := r.FormValue("parent_id")
		if _, err := owner.getParent(selected); err != nil {
			return fmt.Errorf("Choose an existing parent.")
		}
		return owner.assign(id, selected)
	}
	if parent == "" {
		return fmt.Errorf("Assign a parent before placing this %s.", kind)
	}
	// The containing context must still match the form the user saw.
	if r.FormValue("parent_id") != parent {
		return fmt.Errorf("The parent changed; reload before changing placement.")
	}
	if action == "unplace" {
		if err := h.playReferenceProblem(kind, id); err != nil {
			return err
		}
		if kind == "room" && h.roomIsEntryElsewhere(id, "") {
			return fmt.Errorf("Clear this Room as the Location entry before unplacing it.")
		}
		return screen.unassign(id)
	}
	z, err := levelCoordinate(r)
	if err != nil {
		return err
	}
	x, y, problem := placementCoordinates(r)
	if problem != "" {
		return fmt.Errorf("%s", problem)
	}
	occupied, err := h.spatialCellOccupied(screen, parent, id, x, y, z)
	if err != nil {
		return err
	}
	if occupied {
		return fmt.Errorf("That cell is occupied.")
	}
	return screen.place(id, parent, x, y, z)
}

func (h *editorHandler) drillThingAction(parentKind, parentID string, r *http.Request) error {
	kind, id := r.FormValue("thing_kind"), r.FormValue("thing_id")
	if kind != "world_item" && kind != "npc" && kind != "terminal" {
		return fmt.Errorf("Choose an item, NPC, or terminal.")
	}
	action := r.FormValue("action")
	if action == "remove-thing" || (action == "save-thing" && id != "") {
		if err := h.requireHeld(kind, id, parentKind, parentID); err != nil {
			return err
		}
	}
	if action == "remove-thing" {
		_, err := h.contents.Remove(kind, id)
		return err
	}
	if action == "add-thing" {
		_, exists, err := h.entityName(kind, id)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Choose an existing entity.")
		}
	} else {
		v := url.Values{"name": {r.FormValue("thing_name")}, "description": {r.FormValue("thing_description")}, "kind": {r.FormValue("item_kind")}, "short_description": {r.FormValue("short_description")}, "full_description": {r.FormValue("full_description")}, "host_name": {r.FormValue("host_name")}}
		var err error
		id, err = h.saveEntity(kind, id, v)
		if err != nil {
			return err
		}

	}
	if _, err := h.contents.PlaceIn(kind, id, parentKind, parentID); err != nil {
		return fmt.Errorf("Entity was saved but could not be placed; it is available in Library: %w", err)
	}
	return nil
}

func levelCoordinate(r *http.Request) (int, error) {
	v := r.FormValue("z")
	if v == "" {
		return 0, nil
	}
	z, e := strconv.Atoi(v)
	if e != nil {
		return 0, fmt.Errorf("Level must be an integer.")
	}
	return z, nil
}

func drillSavedURL(target string) string {
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	return target + sep + "saved=1"
}
