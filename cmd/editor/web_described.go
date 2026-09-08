package main

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

type describedScreen struct {
	key      string
	label    string
	plural   string
	basePath string
	list     func() ([]describedFields, error)
	get      func(string) (describedFields, error)
	create   func(describedInput) (describedFields, error)
	update   func(string, describedInput) (describedFields, error)
	delete   func(string) (describedFields, error)
}

func (h *editorHandler) roomScreen() describedScreen {
	return describedScreen{
		key: "rooms", label: "Room", plural: "Rooms", basePath: "/content/rooms",
		list: func() ([]describedFields, error) {
			records, err := h.rooms.List()
			out := make([]describedFields, len(records))
			for i, record := range records {
				out[i] = describedFields{ID: record.ID, Name: record.Name, Description: record.Description}
			}
			return out, err
		},
		get: func(id string) (describedFields, error) {
			record, err := h.rooms.Get(id)
			return fieldsFromRoom(record), err
		},
		create: func(input describedInput) (describedFields, error) {
			record, err := h.rooms.Create(input)
			return fieldsFromRoom(record), err
		},
		update: func(id string, input describedInput) (describedFields, error) {
			record, err := h.rooms.Update(id, input)
			return fieldsFromRoom(record), err
		},
		delete: func(id string) (describedFields, error) {
			record, err := h.rooms.Delete(id)
			return fieldsFromRoom(record), err
		},
	}
}

func (h *editorHandler) locationScreen() describedScreen {
	return describedScreen{
		key: "locations", label: "Location", plural: "Locations", basePath: "/content/locations",
		list: func() ([]describedFields, error) {
			records, err := h.locations.List()
			out := make([]describedFields, len(records))
			for i, record := range records {
				out[i] = fieldsFromLocation(record)
			}
			return out, err
		},
		get: func(id string) (describedFields, error) {
			record, err := h.locations.Get(id)
			return fieldsFromLocation(record), err
		},
		create: func(input describedInput) (describedFields, error) {
			record, err := h.locations.Create(input)
			return fieldsFromLocation(record), err
		},
		update: func(id string, input describedInput) (describedFields, error) {
			record, err := h.locations.Update(id, input)
			return fieldsFromLocation(record), err
		},
		delete: func(id string) (describedFields, error) {
			record, err := h.locations.Delete(id)
			return fieldsFromLocation(record), err
		},
	}
}

func (h *editorHandler) npcScreen() describedScreen {
	return describedScreen{
		key: "npcs", label: "NPC", plural: "NPCs", basePath: "/content/npcs",
		list: func() ([]describedFields, error) {
			records, err := h.npcs.List()
			out := make([]describedFields, len(records))
			for i, record := range records {
				out[i] = describedFields{ID: record.ID, Name: record.Name, Description: record.Description}
			}
			return out, err
		},
		get: func(id string) (describedFields, error) {
			record, err := h.npcs.Get(id)
			return fieldsFromNPC(record), err
		},
		create: func(input describedInput) (describedFields, error) {
			record, err := h.npcs.Create(input)
			return fieldsFromNPC(record), err
		},
		update: func(id string, input describedInput) (describedFields, error) {
			record, err := h.npcs.Update(id, input)
			return fieldsFromNPC(record), err
		},
		delete: func(id string) (describedFields, error) {
			record, err := h.npcs.Delete(id)
			return fieldsFromNPC(record), err
		},
	}
}

func fieldsFromRoom(record gamecontent.Room) describedFields {
	return describedFields{ID: record.ID, Name: record.Name, Description: record.Description}
}

func fieldsFromLocation(record gamecontent.Location) describedFields {
	return describedFields{ID: record.ID, Name: record.Name, Description: record.Description}
}

func fieldsFromNPC(record gamecontent.NPC) describedFields {
	return describedFields{ID: record.ID, Name: record.Name, Description: record.Description}
}

func (h *editorHandler) routeDescribedID(w http.ResponseWriter, r *http.Request, screen describedScreen) {
	rest := strings.TrimPrefix(r.URL.Path, screen.basePath+"/")
	if strings.HasSuffix(rest, "/delete") {
		id := strings.TrimSuffix(rest, "/delete")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		h.deleteDescribed(w, r, screen, id)
		return
	}
	if rest == "" || strings.Contains(rest, "/") {
		http.NotFound(w, r)
		return
	}
	h.serveDescribed(w, r, screen, rest)
}

func (h *editorHandler) serveDescribed(w http.ResponseWriter, r *http.Request, screen describedScreen, id string) {
	if r.Method == http.MethodPost {
		h.saveDescribed(w, r, screen, id)
		return
	}
	form := describedForm{Errors: map[string]string{}}
	if id == "" && r.URL.Query().Get("deleted") == "1" {
		form.Notice = screen.label + " deleted."
	}
	if id != "" {
		record, err := screen.get(id)
		if errors.Is(err, errDescribedEntityNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form = formFromDescribed(record)
			if r.URL.Query().Get("saved") == "1" {
				form.Notice = "Saved " + record.Name + "."
			}
		}
	}
	data := h.describedPage(screen, r.URL.Query().Get("q"), form)
	if isHTMX(r) && (id != "" || r.URL.Path == screen.basePath+"/new") {
		h.render(w, "described-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) saveDescribed(w http.ResponseWriter, r *http.Request, screen describedScreen, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid "+strings.ToLower(screen.label)+" form", http.StatusBadRequest)
		return
	}
	form := describedForm{
		ID: id, Name: r.FormValue("name"), Description: r.FormValue("description"),
		Errors: map[string]string{},
	}
	input := describedInput{Name: form.Name, Description: form.Description}.normalized()
	if input.Name == "" {
		form.Errors["name"] = screen.label + " Name is required."
	}
	if input.Description == "" {
		form.Errors["description"] = "Description is required."
	}
	if len(form.Errors) == 0 {
		var record describedFields
		var err error
		if id == "" {
			record, err = screen.create(input)
		} else {
			record, err = screen.update(id, input)
		}
		switch {
		case errors.Is(err, errDescribedEntityNotFound):
			http.NotFound(w, r)
			return
		case err != nil:
			form.GeneralError = err.Error()
		default:
			form = formFromDescribed(record)
			form.Notice = "Saved " + record.Name + "."
		}
	}
	data := h.describedPage(screen, "", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
			w.Header().Set("HX-Push-Url", screen.basePath+"/"+url.PathEscape(form.ID))
		}
		h.render(w, "described-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, screen.basePath+"/"+url.PathEscape(form.ID)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) deleteDescribed(w http.ResponseWriter, r *http.Request, screen describedScreen, id string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	existing, err := screen.get(id)
	if errors.Is(err, errDescribedEntityNotFound) {
		http.NotFound(w, r)
		return
	}
	form := describedForm{Errors: map[string]string{}}
	// A cell that still holds things, or an NPC still standing
	// somewhere, cannot be deleted: the relation would dangle.
	blocked, guardErr := h.describedDeleteGuard(screen, id)
	if err != nil {
		form.GeneralError = err.Error()
	} else if guardErr != nil {
		form = formFromDescribed(existing)
		form.GeneralError = guardErr.Error()
	} else if blocked != "" {
		form = formFromDescribed(existing)
		form.GeneralError = blocked
	} else if screen.key == "rooms" {
		placed, placementErr := h.roomIsAssigned(id)
		if placementErr != nil {
			form = formFromDescribed(existing)
			form.GeneralError = placementErr.Error()
		} else if placed {
			form = formFromDescribed(existing)
			form.GeneralError = "Unassign this Room from its Location before deleting it."
		} else {
			deleted, deleteErr := screen.delete(id)
			if deleteErr != nil {
				form = formFromDescribed(existing)
				form.GeneralError = deleteErr.Error()
			} else {
				form.Notice = "Deleted " + deleted.Name + "."
			}
		}
	} else if screen.key == "locations" {
		placed, placementErr := h.locationIsAssigned(id)
		containsRooms, roomsErr := h.locationHasRooms(id)
		switch {
		case placementErr != nil:
			form = formFromDescribed(existing)
			form.GeneralError = placementErr.Error()
		case roomsErr != nil:
			form = formFromDescribed(existing)
			form.GeneralError = roomsErr.Error()
		case placed:
			form = formFromDescribed(existing)
			form.GeneralError = "Unassign this Location from its Hub before deleting it."
		case containsRooms:
			form = formFromDescribed(existing)
			form.GeneralError = "Unassign every Room before deleting this Location."
		default:
			deleted, deleteErr := screen.delete(id)
			if deleteErr != nil {
				form = formFromDescribed(existing)
				form.GeneralError = deleteErr.Error()
			} else {
				form.Notice = "Deleted " + deleted.Name + "."
			}
		}
	} else {
		deleted, deleteErr := screen.delete(id)
		if deleteErr != nil {
			form = formFromDescribed(existing)
			form.GeneralError = deleteErr.Error()
		} else {
			form.Notice = "Deleted " + deleted.Name + "."
		}
	}
	data := h.describedPage(screen, "", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentDeleted")
			w.Header().Set("HX-Push-Url", screen.basePath)
		}
		h.render(w, "described-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, screen.basePath+"?deleted=1", http.StatusSeeOther)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) serveDescribedList(w http.ResponseWriter, r *http.Request, screen describedScreen) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.describedPage(screen, r.URL.Query().Get("q"), describedForm{Errors: map[string]string{}})
	if isHTMX(r) {
		h.render(w, "described-catalog", data)
		return
	}
	http.Redirect(w, r, screen.basePath+"?q="+url.QueryEscape(data.Query), http.StatusSeeOther)
}

func (h *editorHandler) describedPage(screen describedScreen, query string, form describedForm) pageData {
	data := basePage("content", screen.key, screen.plural, "described")
	data.Query = strings.TrimSpace(query)
	data.DescribedForm = form
	data.DescribedView = "details"
	data.EntityKey = screen.key
	data.EntityLabel = screen.label
	data.EntityPlural = screen.plural
	data.BasePath = screen.basePath
	records, err := screen.list()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	queryLower := strings.ToLower(data.Query)
	for _, record := range records {
		if queryLower != "" && !strings.Contains(strings.ToLower(record.Name), queryLower) {
			continue
		}
		data.DescribedRows = append(data.DescribedRows, describedRow{
			ID: record.ID, Name: record.Name, Description: record.Description, Active: record.ID == form.ID,
		})
	}
	sort.SliceStable(data.DescribedRows, func(i, j int) bool {
		left := strings.ToLower(data.DescribedRows[i].Name)
		right := strings.ToLower(data.DescribedRows[j].Name)
		if left == right {
			return data.DescribedRows[i].ID < data.DescribedRows[j].ID
		}
		return left < right
	})
	return data
}

func formFromDescribed(record describedFields) describedForm {
	return describedForm{ID: record.ID, Name: record.Name, Description: record.Description, Errors: map[string]string{}}
}

// describedDeleteGuard reports why a Room, Location, or NPC cannot be
// deleted yet, or "" when nothing else refers to it.
func (h *editorHandler) describedDeleteGuard(screen describedScreen, id string) (string, error) {
	switch screen.key {
	case "rooms":
		return h.cellBlockingDelete(gamecontent.ContainerKindRoom, id)
	case "locations":
		return h.cellBlockingDelete(gamecontent.ContainerKindLocation, id)
	case "npcs":
		return h.contentsBlockingDelete(gamecontent.ContentKindNPC, id)
	}
	return "", nil
}
