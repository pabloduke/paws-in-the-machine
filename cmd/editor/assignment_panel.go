package main

import (
	"fmt"
	"net/http"
)

type entityAssignmentPanel struct {
	Kind, ID, Current, Error string
	Choices                  []containerOption
}

func (h *editorHandler) assignmentPanel(data *pageData) {
	p := entityAssignmentPanel{}
	switch {
	case data.ItemForm.ID != "":
		p.Kind, p.ID = "world_item", data.ItemForm.ID
	case data.TerminalForm.ID != "":
		p.Kind, p.ID = "terminal", data.TerminalForm.ID
	case data.EntityKey == "npcs" && data.DescribedForm.ID != "":
		p.Kind, p.ID = "npc", data.DescribedForm.ID
	}
	if p.ID == "" {
		return
	}
	current, held, err := h.contents.ContainerOf(p.Kind, p.ID)
	if err != nil {
		p.Error = err.Error()
	}
	if held {
		p.Current = current.ParentKind + ":" + current.ParentID
	}
	snapshot, err := h.worldSnapshot()
	if err != nil {
		p.Error = err.Error()
	} else {
		for _, c := range snapshot.Containers {
			p.Choices = append(p.Choices, containerOption{Value: c.Value(), Path: c.Path, Selected: c.Value() == p.Current})
		}
	}
	data.EntityAssignment = p
}
func (h *editorHandler) saveAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid assignment form.", 400)
		return
	}
	kind, id := r.FormValue("kind"), r.FormValue("id")
	path := map[string]string{"world_item": "/content/world-items/", "npc": "/content/npcs/", "terminal": "/content/terminals/"}[kind]
	problem := h.validateEditorRelationships()
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if _, exists, err := h.entityName(kind, id); err != nil || !exists {
		http.NotFound(w, r)
		return
	}
	current, held, err := h.contents.ContainerOf(kind, id)
	if err != nil {
		problem = err
	}
	value := ""
	if held {
		value = current.ParentKind + ":" + current.ParentID
	}
	if problem == nil && value != r.FormValue("current") {
		problem = fmt.Errorf("Assignment changed. Reload before assigning this entity.")
	}
	if problem == nil {
		destination := r.FormValue("container")
		if destination == "" {
			if held {
				_, problem = h.contents.Remove(kind, id)
			}
		} else {
			pk, pid, ok := parseContainerValue(destination)
			if !ok || !h.containerExists(pk, pid) {
				problem = fmt.Errorf("Choose an existing Location or Room.")
			} else {
				_, problem = h.contents.PlaceIn(kind, id, pk, pid)
			}
		}
	}
	if problem != nil {
		data := basePage("library", "", "Assignment", "selection-error")
		data.StoreError = problem.Error()
		w.Header().Set("HX-Retarget", "#workspace")
		h.renderPage(w, r, data)
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", path+id+"?saved=1")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, path+id+"?saved=1", http.StatusSeeOther)
}
