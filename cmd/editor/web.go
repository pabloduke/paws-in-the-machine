package main

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

//go:embed templates/*.html static/*
var editorFiles embed.FS

type tab struct {
	Label  string
	URL    string
	Key    string
	Active bool
}

type routeSpec struct {
	section string
	detail  string
	title   string
	content string
}

type worldItemRow struct {
	ID               string
	Name             string
	Kind             string
	ShortDescription string
	Active           bool
}

type worldItemForm struct {
	ID               string
	Name             string
	Kind             string
	ShortDescription string
	FullDescription  string
	Errors           map[string]string
	GeneralError     string
	Notice           string
}

func (f worldItemForm) Editing() bool { return f.ID != "" }

type hubRow struct {
	ID     string
	Name   string
	Active bool
}

type hubForm struct {
	ID           string
	Name         string
	NameError    string
	GeneralError string
	Notice       string
}

func (f hubForm) Editing() bool { return f.ID != "" }

type describedRow struct {
	ID          string
	Name        string
	Description string
	Active      bool
}

type describedForm struct {
	ID           string
	Name         string
	Description  string
	Errors       map[string]string
	GeneralError string
	Notice       string
}

func (f describedForm) Editing() bool { return f.ID != "" }

type terminalRow struct {
	ID       string
	HostName string
	Active   bool
}

type terminalForm struct {
	ID           string
	HostName     string
	Errors       map[string]string
	GeneralError string
	Notice       string
}

func (f terminalForm) Editing() bool { return f.ID != "" }

type userRow struct {
	ID       string
	Username string
	Active   bool
}

type userForm struct {
	ID           string
	Username     string
	Password     string
	Errors       map[string]string
	GeneralError string
	Notice       string
}

func (f userForm) Editing() bool { return f.ID != "" }

type hostNetworkRow struct {
	ID     string
	Name   string
	Active bool
}

type hostNetworkForm struct {
	ID           string
	Name         string
	NameError    string
	GeneralError string
	Notice       string
}

func (f hostNetworkForm) Editing() bool { return f.ID != "" }

type pageData struct {
	Title               string
	Section             string
	HeaderTabs          []tab
	DetailTabs          []tab
	SubTabs             []tab
	Content             string
	Query               string
	WorldItems          []worldItemRow
	ItemForm            worldItemForm
	Hubs                []hubRow
	HubForm             hubForm
	DescribedRows       []describedRow
	DescribedForm       describedForm
	EntityKey           string
	EntityLabel         string
	EntityPlural        string
	BasePath            string
	Terminals           []terminalRow
	TerminalForm        terminalForm
	TerminalView        string
	Users               []userRow
	UserForm            userForm
	GrantedUsers        []userRow
	AvailableUsers      []userRow
	HostNetworks        []hostNetworkRow
	NetworkForm         hostNetworkForm
	NetworkView         string
	AssignedTerminals   []terminalRow
	UnassignedTerminals []terminalRow
	NestedTerminalForm  terminalForm
	HubView             string
	DescribedView       string
	Ownership           spatialOwnershipPage
	StoreError          string
	Placement           roomPlacementPage
	Contents            contentsPage
	Overview            overviewPage
	Worlds              worldsPage
	WorldName           string
}

var editorRoutes = map[string]routeSpec{
	"/content/quests": {"content", "quests", "Quests", "placeholder"},
}

var detailTabs = map[string][]tab{
	"overview": {
		{Label: "World Overview", URL: overviewBasePath, Key: "overview"},
	},
	"worlds": {
		{Label: "Worlds", URL: worldsBasePath, Key: "worlds"},
	},
	"content": {
		{Label: "World", URL: "/content/world-items", Key: "world"},
		{Label: "Characters", URL: "/content/npcs", Key: "characters"},
		{Label: "Corporations", URL: "/content/corporations", Key: "corporations"},
	},
	"place": {
		{Label: "Locations on a Hub", URL: "/place/locations", Key: "locations"},
		{Label: "Rooms in a Location", URL: "/place/rooms", Key: "rooms"},
		{Label: "World Items", URL: worldItemContentsBasePath, Key: "world-items"},
		{Label: "NPCs", URL: npcContentsBasePath, Key: "npcs"},
		{Label: "Terminals", URL: terminalContentsBasePath, Key: "terminals"},
	},
}

var contentSubTabs = map[string][]tab{
	"world": {
		{Label: "World Items", URL: "/content/world-items", Key: "world-items"},
		{Label: "Locations", URL: "/content/locations", Key: "locations"},
		{Label: "Rooms", URL: "/content/rooms", Key: "rooms"},
		{Label: "Hubs", URL: "/content/hubs", Key: "hubs"},
	},
	"characters": {
		{Label: "NPCs", URL: "/content/npcs", Key: "npcs"},
	},
	"corporations": {
		{Label: "Corporations", URL: "/content/corporations", Key: "corporations"},
		{Label: "Terminals", URL: "/content/terminals", Key: "terminals"},
		{Label: "Users", URL: "/content/users", Key: "users"},
	},
}

type editorHandler struct {
	templates           *template.Template
	static              http.Handler
	items               *worldItemStore
	hubs                *hubStore
	locations           *locationStore
	rooms               *roomStore
	npcs                *npcStore
	terminals           *terminalStore
	networks            *hostNetworkStore
	assignments         *networkAssignmentStore
	users               *userStore
	access              *terminalAccessStore
	roomPlacements      *roomPlacementStore
	locationPlacements  *locationPlacementStore
	locationEntries     *locationEntryStore
	locationAssignments *locationAssignmentStore
	roomAssignments     *roomAssignmentStore
	contents            *contentsStore
	worlds              *worldRegistry
}

// newWorldHandler builds a complete editor over one content directory.
// The registry is carried so the Worlds screen can list and switch
// worlds; it is nil in tests that construct a handler directly.
func newWorldHandler(contentDir string, worlds *worldRegistry) http.Handler {
	return newEditorHandler(
		newWorldItemStore(contentDir), newHubStore(contentDir), newRoomStore(contentDir),
		newNPCStore(contentDir), newTerminalStore(contentDir), newHostNetworkStore(contentDir),
		newNetworkAssignmentStore(contentDir), newUserStore(contentDir),
		newTerminalAccessStore(contentDir), newRoomPlacementStore(contentDir), worlds,
	)
}

func newEditorHandler(items *worldItemStore, hubs *hubStore, rooms *roomStore, npcs *npcStore, terminals *terminalStore, networks *hostNetworkStore, assignments *networkAssignmentStore, users *userStore, access *terminalAccessStore, roomPlacements *roomPlacementStore, worlds ...*worldRegistry) http.Handler {
	templates := template.Must(template.ParseFS(editorFiles, "templates/*.html"))
	staticFiles, err := fs.Sub(editorFiles, "static")
	if err != nil {
		panic(err)
	}
	contentDir := filepath.Dir(rooms.path)
	var registry *worldRegistry
	if len(worlds) > 0 {
		registry = worlds[0]
	}
	h := &editorHandler{
		worlds:              registry,
		templates:           templates,
		static:              http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))),
		items:               items,
		hubs:                hubs,
		locations:           newLocationStore(contentDir),
		rooms:               rooms,
		npcs:                npcs,
		terminals:           terminals,
		networks:            networks,
		assignments:         assignments,
		users:               users,
		access:              access,
		roomPlacements:      roomPlacements,
		locationPlacements:  newLocationPlacementStore(contentDir),
		locationEntries:     newLocationEntryStore(contentDir),
		locationAssignments: newLocationAssignmentStore(contentDir),
		roomAssignments:     newRoomAssignmentStore(contentDir),
		contents:            newContentsStore(contentDir),
	}
	mux := http.NewServeMux()
	mux.Handle("/static/", h.static)
	mux.HandleFunc("/", h.serveEditor)
	return mux
}

func (h *editorHandler) serveEditor(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		http.Redirect(w, r, overviewBasePath, http.StatusSeeOther)
		return
	}
	if target, ok := legacyRoute(r.URL.Path); ok {
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	if r.Method == http.MethodPost && !sameOrigin(r) {
		http.Error(w, "editor writes require a same-origin request", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost && (strings.HasPrefix(r.URL.Path, terminalBasePath) || strings.HasPrefix(r.URL.Path, hostNetworkBasePath) || strings.HasPrefix(r.URL.Path, userBasePath) || strings.HasPrefix(r.URL.Path, roomPlacementBasePath) || strings.HasPrefix(r.URL.Path, locationPlacementBasePath) || strings.HasPrefix(r.URL.Path, "/content/locations") || strings.HasPrefix(r.URL.Path, "/content/rooms") || strings.HasPrefix(r.URL.Path, "/content/hubs") || strings.HasPrefix(r.URL.Path, worldItemContentsBasePath) || strings.HasPrefix(r.URL.Path, npcContentsBasePath) || strings.HasPrefix(r.URL.Path, terminalContentsBasePath)) {
		if err := h.validateEditorRelationships(); err != nil {
			http.Error(w, "editor relationships are invalid: "+err.Error(), http.StatusConflict)
			return
		}
	}

	switch {
	case r.URL.Path == overviewBasePath:
		h.serveOverview(w, r)
	case r.URL.Path == worldsBasePath || strings.HasPrefix(r.URL.Path, worldsBasePath+"/"):
		h.routeWorlds(w, r)
	case strings.HasPrefix(r.URL.Path, worldItemContentsBasePath),
		strings.HasPrefix(r.URL.Path, npcContentsBasePath),
		strings.HasPrefix(r.URL.Path, terminalContentsBasePath):
		screen, ok := h.contentsScreenFor(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.routeContents(w, r, screen)
	case r.URL.Path == "/content/users/list":
		h.serveUserList(w, r)
	case r.URL.Path == "/content/users" || r.URL.Path == "/content/users/new":
		h.serveUsers(w, r, "")
	case strings.HasPrefix(r.URL.Path, "/content/users/"):
		h.routeUserID(w, r)
	case r.URL.Path == "/content/host-networks" || strings.HasPrefix(r.URL.Path, "/content/host-networks/"):
		http.Redirect(w, r, strings.Replace(r.URL.Path, "/content/host-networks", corporationBasePath, 1), http.StatusPermanentRedirect)
	case r.URL.Path == corporationBasePath+"/list":
		h.serveHostNetworkList(w, r)
	case r.URL.Path == corporationBasePath || r.URL.Path == corporationBasePath+"/new":
		h.serveHostNetworks(w, r, "")
	case strings.HasPrefix(r.URL.Path, corporationBasePath+"/"):
		h.routeHostNetworkID(w, r)
	case r.URL.Path == "/content/locations/list":
		h.serveDescribedList(w, r, h.locationScreen())
	case r.URL.Path == "/content/locations" || r.URL.Path == "/content/locations/new":
		h.serveDescribed(w, r, h.locationScreen(), "")
	case strings.HasPrefix(r.URL.Path, "/content/locations/"):
		h.routeLocationID(w, r)
	case r.URL.Path == locationPlacementBasePath:
		h.serveLocationPlacements(w, r)
	case r.URL.Path == locationPlacementBasePath+"/place":
		h.placeLocation(w, r)
	case r.URL.Path == locationPlacementBasePath+"/unassign":
		h.unassignLocation(w, r)
	case r.URL.Path == roomPlacementBasePath:
		h.serveRoomPlacements(w, r)
	case r.URL.Path == roomPlacementBasePath+"/place":
		h.placeRoom(w, r)
	case r.URL.Path == roomPlacementBasePath+"/unassign":
		h.unassignRoom(w, r)
	case r.URL.Path == "/content/rooms/list":
		h.serveDescribedList(w, r, h.roomScreen())
	case r.URL.Path == "/content/rooms" || r.URL.Path == "/content/rooms/new":
		h.serveDescribed(w, r, h.roomScreen(), "")
	case strings.HasPrefix(r.URL.Path, "/content/rooms/"):
		h.routeDescribedID(w, r, h.roomScreen())
	case r.URL.Path == "/content/npcs/list":
		h.serveDescribedList(w, r, h.npcScreen())
	case r.URL.Path == "/content/npcs" || r.URL.Path == "/content/npcs/new":
		h.serveDescribed(w, r, h.npcScreen(), "")
	case strings.HasPrefix(r.URL.Path, "/content/npcs/"):
		h.routeDescribedID(w, r, h.npcScreen())
	case r.URL.Path == "/content/terminals/list":
		h.serveTerminalList(w, r)
	case r.URL.Path == "/content/terminals" || r.URL.Path == "/content/terminals/new":
		h.serveTerminals(w, r, "")
	case strings.HasPrefix(r.URL.Path, "/content/terminals/"):
		h.routeTerminalID(w, r)
	case r.URL.Path == "/content/hubs/list":
		h.serveHubList(w, r)
	case r.URL.Path == "/content/hubs" || r.URL.Path == "/content/hubs/new":
		h.serveHubs(w, r, "")
	case strings.HasPrefix(r.URL.Path, "/content/hubs/"):
		h.routeHubID(w, r)
	case r.URL.Path == "/content/world-items/list":
		h.serveWorldItemList(w, r)
	case r.URL.Path == "/content/world-items" || r.URL.Path == "/content/world-items/new":
		h.serveWorldItems(w, r, "")
	case strings.HasSuffix(r.URL.Path, "/delete") && strings.HasPrefix(r.URL.Path, "/content/world-items/"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/content/world-items/"), "/delete")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		h.deleteWorldItem(w, r, id)
	case strings.HasPrefix(r.URL.Path, "/content/world-items/"):
		id := strings.TrimPrefix(r.URL.Path, "/content/world-items/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		h.serveWorldItems(w, r, id)
	default:
		h.serveStaticScreen(w, r)
	}
}

func (h *editorHandler) serveHubs(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodPost {
		h.saveHub(w, r, id)
		return
	}
	form := hubForm{}
	if id == "" && r.URL.Query().Get("deleted") == "1" {
		form.Notice = "Hub deleted."
	}
	if id != "" {
		hub, err := h.hubs.Get(id)
		if errors.Is(err, errHubNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form = formFromHub(hub)
			if r.URL.Query().Get("saved") == "1" {
				form.Notice = "Saved " + hub.Name + "."
			}
		}
	}
	data := h.hubsPage(r.URL.Query().Get("q"), form)
	if isHTMX(r) && (id != "" || r.URL.Path == "/content/hubs/new") {
		h.render(w, "hub-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) saveHub(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid hub form", http.StatusBadRequest)
		return
	}
	form := hubForm{ID: id, Name: r.FormValue("name")}
	name := strings.TrimSpace(form.Name)
	if name == "" {
		form.NameError = "Hub Name is required."
	} else {
		var hub gamecontent.Hub
		var err error
		if id == "" {
			hub, err = h.hubs.Create(name)
		} else {
			hub, err = h.hubs.Update(id, name)
		}
		switch {
		case errors.Is(err, errHubNotFound):
			http.NotFound(w, r)
			return
		case err != nil:
			form.GeneralError = err.Error()
		default:
			form = formFromHub(hub)
			form.Notice = "Saved " + hub.Name + "."
		}
	}
	data := h.hubsPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
			w.Header().Set("HX-Push-Url", "/content/hubs/"+url.PathEscape(form.ID))
		}
		h.render(w, "hubs-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, "/content/hubs/"+url.PathEscape(form.ID)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) deleteHub(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	hub, err := h.hubs.Get(id)
	if errors.Is(err, errHubNotFound) {
		http.NotFound(w, r)
		return
	}
	form := hubForm{}
	if err != nil {
		form.GeneralError = err.Error()
	} else if placed, placementErr := h.hubHasLocations(id); placementErr != nil {
		form.GeneralError = placementErr.Error()
	} else if placed {
		form = formFromHub(hub)
		form.GeneralError = "Unassign every Location before deleting this Hub."
	} else {
		deleted, deleteErr := h.hubs.Delete(id)
		if deleteErr != nil {
			form.GeneralError = deleteErr.Error()
		} else {
			form.Notice = "Deleted " + deleted.Name + "."
		}
	}
	data := h.hubsPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentDeleted")
			w.Header().Set("HX-Push-Url", "/content/hubs")
		}
		h.render(w, "hubs-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, "/content/hubs?deleted=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) serveHubList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.hubsPage(r.URL.Query().Get("q"), hubForm{})
	if isHTMX(r) {
		h.render(w, "hub-catalog", data)
		return
	}
	http.Redirect(w, r, "/content/hubs?q="+url.QueryEscape(data.Query), http.StatusSeeOther)
}

func (h *editorHandler) hubsPage(query string, form hubForm) pageData {
	data := basePage("content", "hubs", "Hubs", "hubs")
	data.Query = strings.TrimSpace(query)
	data.HubForm = form
	data.HubView = "details"
	hubs, err := h.hubs.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	queryLower := strings.ToLower(data.Query)
	for _, hub := range hubs {
		if queryLower != "" && !strings.Contains(strings.ToLower(hub.Name), queryLower) {
			continue
		}
		data.Hubs = append(data.Hubs, hubRow{ID: hub.ID, Name: hub.Name, Active: hub.ID == form.ID})
	}
	sort.SliceStable(data.Hubs, func(i, j int) bool {
		left := strings.ToLower(data.Hubs[i].Name)
		right := strings.ToLower(data.Hubs[j].Name)
		if left == right {
			return data.Hubs[i].ID < data.Hubs[j].ID
		}
		return left < right
	})
	return data
}

func (h *editorHandler) serveWorldItems(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodPost {
		h.saveWorldItem(w, r, id)
		return
	}
	form := worldItemForm{Kind: "takeable", Errors: map[string]string{}}
	if id == "" && r.URL.Query().Get("deleted") == "1" {
		form.Notice = "World item deleted."
	}
	if id != "" {
		item, err := h.items.Get(id)
		if errors.Is(err, errWorldItemNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form = formFromItem(item)
			if r.URL.Query().Get("saved") == "1" {
				form.Notice = "Saved " + item.Name + "."
			}
		}
	}
	data := h.worldItemsPage(r.URL.Query().Get("q"), form)
	if isHTMX(r) && (id != "" || r.URL.Path == "/content/world-items/new") {
		h.render(w, "world-item-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) saveWorldItem(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid world-item form", http.StatusBadRequest)
		return
	}
	input, form := worldItemInputFromRequest(r)
	form.ID = id
	if len(form.Errors) == 0 {
		var err error
		var savedID string
		if id == "" {
			item, createErr := h.items.Create(input)
			err = createErr
			savedID = item.ID
		} else {
			item, updateErr := h.items.Update(id, input)
			err = updateErr
			savedID = item.ID
		}
		switch {
		case errors.Is(err, errWorldItemNotFound):
			http.NotFound(w, r)
			return
		case err != nil:
			form.GeneralError = err.Error()
		default:
			item, getErr := h.items.Get(savedID)
			if getErr != nil {
				form.GeneralError = getErr.Error()
			} else {
				form = formFromItem(item)
				form.Notice = "Saved " + item.Name + "."
			}
		}
	}
	data := h.worldItemsPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
			w.Header().Set("HX-Push-Url", "/content/world-items/"+url.PathEscape(form.ID))
		}
		h.render(w, "world-items-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, "/content/world-items/"+url.PathEscape(form.ID)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) deleteWorldItem(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	blocked, guardErr := h.contentsBlockingDelete(gamecontent.ContentKindWorldItem, id)
	form := worldItemForm{Kind: "takeable", Errors: map[string]string{}}
	var err error
	if guardErr != nil || blocked != "" {
		existing, getErr := h.items.Get(id)
		if errors.Is(getErr, errWorldItemNotFound) {
			http.NotFound(w, r)
			return
		}
		form = formFromItem(existing)
		form.GeneralError = blocked
		if guardErr != nil {
			form.GeneralError = guardErr.Error()
		}
		err = errors.New(form.GeneralError)
	} else {
		var deleted gamecontent.WorldItem
		deleted, err = h.items.Delete(id)
		if errors.Is(err, errWorldItemNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form.Notice = "Deleted " + deleted.Name + "."
		}
	}
	data := h.worldItemsPage("", form)
	if isHTMX(r) {
		if err == nil {
			w.Header().Set("HX-Trigger", "contentDeleted")
			w.Header().Set("HX-Push-Url", "/content/world-items")
		}
		h.render(w, "world-items-body", data)
		return
	}
	if err == nil {
		http.Redirect(w, r, "/content/world-items?deleted=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) serveWorldItemList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.worldItemsPage(r.URL.Query().Get("q"), worldItemForm{Kind: "takeable"})
	if isHTMX(r) {
		h.render(w, "world-item-catalog", data)
		return
	}
	http.Redirect(w, r, "/content/world-items?q="+url.QueryEscape(data.Query), http.StatusSeeOther)
}

func (h *editorHandler) worldItemsPage(query string, form worldItemForm) pageData {
	data := basePage("content", "world-items", "World Items", "world-items")
	data.Query = strings.TrimSpace(query)
	data.ItemForm = form
	items, err := h.items.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	queryLower := strings.ToLower(data.Query)
	for _, item := range items {
		if queryLower != "" && !strings.Contains(strings.ToLower(item.Name), queryLower) {
			continue
		}
		data.WorldItems = append(data.WorldItems, worldItemRow{
			ID:               item.ID,
			Name:             item.Name,
			Kind:             titleKind(string(item.Kind)),
			ShortDescription: item.ShortDescription,
			Active:           item.ID == form.ID,
		})
	}
	sort.SliceStable(data.WorldItems, func(i, j int) bool {
		left := strings.ToLower(data.WorldItems[i].Name)
		right := strings.ToLower(data.WorldItems[j].Name)
		if left == right {
			return data.WorldItems[i].ID < data.WorldItems[j].ID
		}
		return left < right
	})
	return data
}

func (h *editorHandler) serveStaticScreen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	spec, ok := editorRoutes[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.renderPage(w, r, basePage(spec.section, spec.detail, spec.title, spec.content))
}

func (h *editorHandler) renderPage(w http.ResponseWriter, r *http.Request, data pageData) {
	data.WorldName = h.currentWorldName()
	name := "page"
	if isHTMX(r) {
		name = "workspace"
	}
	h.render(w, name, data)
}

func (h *editorHandler) render(w http.ResponseWriter, name string, data pageData) {
	if data.WorldName == "" {
		data.WorldName = h.currentWorldName()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "HX-Request")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "editor template error", http.StatusInternalServerError)
	}
}

func (h *editorHandler) methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func basePage(section, detail, title, content string) pageData {
	group := detail
	if section == "content" {
		group = contentGroup(detail)
	}
	return pageData{
		Title:      title,
		Section:    section,
		Content:    content,
		HeaderTabs: headerTabs(section),
		DetailTabs: activeTabs(detailTabs[section], group),
		SubTabs:    activeTabs(contentSubTabs[group], detail),
	}
}

func contentGroup(detail string) string {
	switch detail {
	case "world-items", "locations", "rooms", "hubs":
		return "world"
	case "npcs", "quests":
		return "characters"
	case "corporations", "terminals", "users", "host-networks":
		return "corporations"
	default:
		return detail
	}
}

func headerTabs(active string) []tab {
	return activeTabs([]tab{
		{Label: "Overview", URL: overviewBasePath, Key: "overview"},
		{Label: "Content", URL: "/content/world-items", Key: "content"},
		{Label: "Place", URL: "/place/locations", Key: "place"},
		{Label: "Worlds", URL: worldsBasePath, Key: "worlds"},
	}, active)
}

func activeTabs(tabs []tab, active string) []tab {
	out := make([]tab, len(tabs))
	copy(out, tabs)
	for i := range out {
		out[i].Active = out[i].Key == active
	}
	return out
}

func legacyRoute(path string) (string, bool) {
	for _, prefix := range []string{"/create/", "/edit/"} {
		if strings.HasPrefix(path, prefix) {
			detail := strings.TrimPrefix(path, prefix)
			for _, tabs := range contentSubTabs {
				for _, tab := range tabs {
					if detail == tab.Key {
						return tab.URL, true
					}
				}
			}
		}
	}
	return "", false
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

func titleKind(kind string) string {
	if kind == "" {
		return ""
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func formFromItem(item gamecontent.WorldItem) worldItemForm {
	return worldItemForm{
		ID:               item.ID,
		Name:             item.Name,
		Kind:             string(item.Kind),
		ShortDescription: item.ShortDescription,
		FullDescription:  item.FullDescription,
		Errors:           map[string]string{},
	}
}

func formFromHub(hub gamecontent.Hub) hubForm {
	return hubForm{ID: hub.ID, Name: hub.Name}
}

func worldItemInputFromRequest(r *http.Request) (worldItemInput, worldItemForm) {
	form := worldItemForm{
		Name:             r.FormValue("name"),
		Kind:             r.FormValue("kind"),
		ShortDescription: r.FormValue("short_description"),
		FullDescription:  r.FormValue("full_description"),
		Errors:           map[string]string{},
	}
	input := worldItemInput{
		Name:             form.Name,
		Kind:             gamecontent.WorldItemKind(form.Kind),
		ShortDescription: form.ShortDescription,
		FullDescription:  form.FullDescription,
	}.normalized()
	if input.Name == "" {
		form.Errors["name"] = "Item Name is required."
	}
	if !input.Kind.Valid() {
		form.Errors["kind"] = "Choose Takeable, Fixed, or Scenery."
	}
	if input.ShortDescription == "" {
		form.Errors["short_description"] = "Short Description is required."
	}
	if input.FullDescription == "" {
		form.Errors["full_description"] = "Full Description is required."
	}
	return input, form
}
