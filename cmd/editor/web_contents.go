package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const worldItemContentsBasePath = "/place/world-items"
const npcContentsBasePath = "/place/npcs"
const terminalContentsBasePath = "/place/terminals"

type contentsRow struct {
	ID, Kind, Name string
}

type contentsChoice struct {
	ID, Name, Status string
}

type containerOption struct {
	Value, Path string
	Selected    bool
}

type contentsPage struct {
	EntityLabel, EntityPlural, BasePath string
	Containers                          []containerOption
	SelectedValue, SelectedPath         string
	SelectedKindLabel                   string
	Held                                []contentsRow
	Choices                             []contentsChoice
	LooseCount                          int
	CatalogPath                         string
	Notice, GeneralError                string
}

type contentsScreen struct {
	key, entityKind, label, plural, basePath, catalogPath string
}

func (h *editorHandler) worldItemContentsScreen() contentsScreen {
	return contentsScreen{
		key: "world-items", entityKind: gamecontent.ContentKindWorldItem,
		label: "World Item", plural: "World Items",
		basePath: worldItemContentsBasePath, catalogPath: "/content/world-items",
	}
}

func (h *editorHandler) npcContentsScreen() contentsScreen {
	return contentsScreen{
		key: "npcs", entityKind: gamecontent.ContentKindNPC,
		label: "NPC", plural: "NPCs",
		basePath: npcContentsBasePath, catalogPath: "/content/npcs",
	}
}

func (h *editorHandler) terminalContentsScreen() contentsScreen {
	return contentsScreen{
		key: "terminals", entityKind: gamecontent.ContentKindTerminal,
		label: "Terminal", plural: "Terminals",
		basePath: terminalContentsBasePath, catalogPath: "/content/terminals",
	}
}

func (h *editorHandler) contentsScreenFor(path string) (contentsScreen, bool) {
	switch {
	case strings.HasPrefix(path, worldItemContentsBasePath):
		return h.worldItemContentsScreen(), true
	case strings.HasPrefix(path, npcContentsBasePath):
		return h.npcContentsScreen(), true
	case strings.HasPrefix(path, terminalContentsBasePath):
		return h.terminalContentsScreen(), true
	}
	return contentsScreen{}, false
}

func (h *editorHandler) routeContents(w http.ResponseWriter, r *http.Request, screen contentsScreen) {
	switch r.URL.Path {
	case screen.basePath:
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		h.renderContents(w, r, screen, r.URL.Query().Get("container"), "", "")
	case screen.basePath + "/add":
		h.addContent(w, r, screen)
	case screen.basePath + "/remove":
		h.removeContent(w, r, screen)
	default:
		http.NotFound(w, r)
	}
}

func (h *editorHandler) addContent(w http.ResponseWriter, r *http.Request, screen contentsScreen) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid contents form", http.StatusBadRequest)
		return
	}
	containerValue, entityID := r.FormValue("container"), r.FormValue("entity_id")
	notice, generalError := "", ""
	parentKind, parentID, ok := parseContainerValue(containerValue)
	switch {
	case !ok:
		generalError = "Choose a Location or Room to put this " + screen.label + " in."
	case !h.containerExists(parentKind, parentID):
		generalError = "That cell no longer exists."
	default:
		name, exists, err := h.entityName(screen.entityKind, entityID)
		if err != nil {
			generalError = err.Error()
		} else if !exists {
			generalError = "Choose an existing " + screen.label + "."
		} else if _, err := h.contents.PlaceIn(screen.entityKind, entityID, parentKind, parentID); err != nil {
			generalError = err.Error()
		} else {
			notice = "Put " + name + " here."
		}
	}
	h.renderContents(w, r, screen, containerValue, notice, generalError)
}

func (h *editorHandler) removeContent(w http.ResponseWriter, r *http.Request, screen contentsScreen) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid contents form", http.StatusBadRequest)
		return
	}
	containerValue, entityID := r.FormValue("container"), r.FormValue("entity_id")
	name, _, _ := h.entityName(screen.entityKind, entityID)
	notice, generalError := "", ""
	// The form names the cell it was rendered for. If the entity has
	// moved since — another tab, another window — removing by entity
	// alone would silently delete its new placement, so the submitted
	// cell must still be the one holding it.
	parentKind, parentID, ok := parseContainerValue(containerValue)
	current, held, err := h.contents.ContainerOf(screen.entityKind, entityID)
	switch {
	case !ok:
		generalError = "This screen is out of date; reload it and try again."
	case err != nil:
		generalError = err.Error()
	case !held:
		generalError = name + " has moved or was already removed; this screen is out of date. Reload it."
	case current.ParentKind != parentKind || current.ParentID != parentID:
		where := "another place"
		if path, exists, pathErr := h.containerName(current.ParentKind, current.ParentID); pathErr == nil && exists {
			where = path
		}
		generalError = name + " has moved to " + where + " since this screen was drawn, so nothing was removed. Reload it."
	default:
		if _, removeErr := h.contents.Remove(screen.entityKind, entityID); removeErr != nil {
			generalError = "That " + screen.label + " is not in this cell."
		} else {
			notice = "Removed " + name + ". It still exists in the catalog."
		}
	}
	h.renderContents(w, r, screen, containerValue, notice, generalError)
}

func (h *editorHandler) containerExists(kind, id string) bool {
	switch kind {
	case gamecontent.ContainerKindLocation:
		_, err := h.locations.Get(id)
		return err == nil
	case gamecontent.ContainerKindRoom:
		_, err := h.rooms.Get(id)
		return err == nil
	}
	return false
}

func (h *editorHandler) entityName(kind, id string) (string, bool, error) {
	switch kind {
	case gamecontent.ContentKindWorldItem:
		record, err := h.items.Get(id)
		if err != nil {
			return "", false, nil
		}
		return record.Name, true, nil
	case gamecontent.ContentKindNPC:
		record, err := h.npcs.Get(id)
		if err != nil {
			return "", false, nil
		}
		return record.Name, true, nil
	case gamecontent.ContentKindTerminal:
		record, err := h.terminals.Get(id)
		if err != nil {
			return "", false, nil
		}
		return record.HostName, true, nil
	}
	return "", false, nil
}

func (h *editorHandler) renderContents(w http.ResponseWriter, r *http.Request, screen contentsScreen, containerValue, notice, generalError string) {
	data := h.contentsPageData(screen, containerValue, notice, generalError)
	// Only a form submission swaps the body on its own. A GET is either
	// tab navigation or the cell picker, both of which target #workspace
	// and need the whole screen — tabs, headings, and the
	// #contents-workspace wrapper the forms post back into.
	if isHTMX(r) && r.Method == http.MethodPost {
		if data.Contents.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
		}
		h.render(w, "contents-body", data)
		return
	}
	if isHTMX(r) {
		h.render(w, "workspace", data)
		return
	}
	if data.Contents.Notice != "" && r.Method == http.MethodPost {
		http.Redirect(w, r, screen.basePath+"?container="+url.QueryEscape(data.Contents.SelectedValue), http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) contentsPageData(screen contentsScreen, containerValue, notice, generalError string) pageData {
	data := basePage("place", screen.key, "Place "+screen.plural, "contents")
	page := contentsPage{
		EntityLabel: screen.label, EntityPlural: screen.plural,
		BasePath: screen.basePath, CatalogPath: screen.catalogPath,
		Notice: notice, GeneralError: generalError,
	}
	snapshot, err := h.worldSnapshot()
	if err != nil {
		data.StoreError = err.Error()
		data.Contents = page
		return data
	}
	if len(snapshot.Containers) == 0 {
		data.Contents = page
		return data
	}
	if _, _, ok := parseContainerValue(containerValue); !ok {
		containerValue = snapshot.Containers[0].Value()
	}
	found := false
	for _, c := range snapshot.Containers {
		selected := c.Value() == containerValue
		found = found || selected
		page.Containers = append(page.Containers, containerOption{Value: c.Value(), Path: c.Path, Selected: selected})
		if selected {
			page.SelectedValue, page.SelectedPath = c.Value(), c.Path
			page.SelectedKindLabel = "Location"
			if c.Kind == gamecontent.ContainerKindRoom {
				page.SelectedKindLabel = "Room"
			}
		}
	}
	if !found {
		page.GeneralError = "That cell no longer exists."
		page.SelectedValue = snapshot.Containers[0].Value()
		page.SelectedPath = snapshot.Containers[0].Path
		page.Containers[0].Selected = true
	}

	// Read the cell directly. Walking the Hub tree would hide any cell
	// whose ancestors are unassigned, and unassigned is a valid state.
	page.Held = contentRowsOfKind(snapshot.ContentsOf[page.SelectedValue], screen.entityKind)

	loose := snapshot.Loose[screen.entityKind]
	page.LooseCount = len(loose)
	for _, thing := range loose {
		page.Choices = append(page.Choices, contentsChoice{ID: thing.ID, Name: thing.Name, Status: "not placed"})
	}
	// Things already somewhere else can be moved here in one step; the
	// dropdown says where each one currently is.
	for _, thing := range snapshot.Placed[screen.entityKind] {
		where := snapshot.Placement[contentKey(screen.entityKind, thing.ID)]
		if where == "" || where == page.SelectedPath {
			continue
		}
		page.Choices = append(page.Choices, contentsChoice{ID: thing.ID, Name: thing.Name, Status: "in " + where})
	}
	data.Contents = page
	return data
}

func contentRowsOfKind(contents []worldThing, kind string) []contentsRow {
	rows := make([]contentsRow, 0, len(contents))
	for _, thing := range contents {
		if thing.Kind == kind {
			rows = append(rows, contentsRow{ID: thing.ID, Kind: thing.Kind, Name: thing.Name})
		}
	}
	return rows
}

// Dependency guards. The editor refuses to delete anything another
// authored relationship still points at, so a delete can never leave a
// dangling UUID behind.

// contentsBlockingDelete reports why the entity cannot be deleted, or
// "" when nothing holds it.
func (h *editorHandler) contentsBlockingDelete(entityKind, entityID string) (string, error) {
	record, held, err := h.contents.ContainerOf(entityKind, entityID)
	if err != nil || !held {
		return "", err
	}
	where := "a Location"
	if record.ParentKind == gamecontent.ContainerKindRoom {
		where = "a Room"
	}
	if name, exists, nameErr := h.containerName(record.ParentKind, record.ParentID); nameErr == nil && exists {
		where = name
	}
	return "Remove this " + contentKindLabel(entityKind) + " from " + where + " before deleting it.", nil
}

// cellBlockingDelete reports why the cell cannot be deleted, or "" when
// it holds nothing.
func (h *editorHandler) cellBlockingDelete(parentKind, parentID string) (string, error) {
	records, err := h.contents.List()
	if err != nil {
		return "", err
	}
	for _, record := range records {
		if record.ParentKind == parentKind && record.ParentID == parentID {
			label := "Location"
			if parentKind == gamecontent.ContainerKindRoom {
				label = "Room"
			}
			return "Take everything out of this " + label + " before deleting it.", nil
		}
	}
	return "", nil
}

func (h *editorHandler) containerName(kind, id string) (string, bool, error) {
	switch kind {
	case gamecontent.ContainerKindLocation:
		record, err := h.locations.Get(id)
		if err != nil {
			return "", false, nil
		}
		return record.Name, true, nil
	case gamecontent.ContainerKindRoom:
		record, err := h.rooms.Get(id)
		if err != nil {
			return "", false, nil
		}
		return record.Name, true, nil
	}
	return "", false, nil
}

// validateContentsRelationships is the write-gate form of
// contentsProblems: it stops at the first problem, so a write is
// refused while the authored graph is already broken. The Overview
// uses the collecting form. Both read the same records, so the gate and
// the report can never disagree about whether the graph is valid.
func (h *editorHandler) validateContentsRelationships() error {
	problems, err := h.contentsProblems()
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return errors.New(problems[0])
	}
	return nil
}
