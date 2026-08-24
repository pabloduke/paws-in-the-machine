package main

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const terminalBasePath = "/content/terminals"

func (h *editorHandler) routeTerminalID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, terminalBasePath+"/")
	parts := strings.Split(rest, "/")
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 1:
		h.serveTerminals(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "delete":
		h.deleteTerminal(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "access":
		h.serveTerminalAccess(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "grant":
		h.grantTerminalAccess(w, r, parts[0])
	case len(parts) == 3 && parts[1] == "revoke":
		h.revokeTerminalAccess(w, r, parts[0], parts[2])
	default:
		http.NotFound(w, r)
	}
}

func (h *editorHandler) serveTerminals(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodPost {
		h.saveTerminal(w, r, id)
		return
	}
	form := terminalForm{Errors: map[string]string{}}
	if id == "" && r.URL.Query().Get("deleted") == "1" {
		form.Notice = "Terminal deleted."
	}
	if id != "" {
		terminal, err := h.terminals.Get(id)
		if errors.Is(err, errTerminalNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form = formFromTerminal(terminal)
			if r.URL.Query().Get("saved") == "1" {
				form.Notice = "Saved " + terminal.HostName + "."
			}
		}
	}
	data := h.terminalsPage(r.URL.Query().Get("q"), form)
	data.TerminalView = "details"
	if isHTMX(r) && (id != "" || r.URL.Path == terminalBasePath+"/new") {
		h.render(w, "terminal-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) saveTerminal(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid terminal form", http.StatusBadRequest)
		return
	}
	form, input := terminalFormFromRequest(r)
	form.ID = id
	if len(form.Errors) == 0 {
		var terminal gamecontent.Terminal
		var err error
		if id != "" {
			networkID, assignmentErr := h.terminalNetwork(id)
			if assignmentErr != nil {
				form.GeneralError = assignmentErr.Error()
			} else if networkID != "" {
				collision, collisionErr := h.networkHasHostname(networkID, id, input.HostName)
				if collisionErr != nil {
					form.GeneralError = collisionErr.Error()
				} else if collision {
					form.Errors["host_name"] = "Machine Hostname must be unique within its assigned network."
				}
			}
		}
		if len(form.Errors) == 0 && form.GeneralError == "" {
			if id == "" {
				terminal, err = h.terminals.Create(input)
			} else {
				terminal, err = h.terminals.Update(id, input)
			}
			switch {
			case errors.Is(err, errTerminalNotFound):
				http.NotFound(w, r)
				return
			case err != nil:
				form.GeneralError = err.Error()
			default:
				form = formFromTerminal(terminal)
				form.Notice = "Saved " + terminal.HostName + "."
			}
		}
	}
	data := h.terminalsPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
			w.Header().Set("HX-Push-Url", terminalBasePath+"/"+url.PathEscape(form.ID))
		}
		h.render(w, "terminals-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, terminalBasePath+"/"+url.PathEscape(form.ID)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) deleteTerminal(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	terminal, err := h.terminals.Get(id)
	if errors.Is(err, errTerminalNotFound) {
		http.NotFound(w, r)
		return
	}
	form := formFromTerminal(terminal)
	if err != nil {
		form.GeneralError = err.Error()
	} else if networkID, assignmentErr := h.terminalNetwork(id); assignmentErr != nil {
		form.GeneralError = assignmentErr.Error()
	} else if networkID != "" {
		form.GeneralError = "Unassign this terminal from its corporation before deleting it."
	} else if granted, accessErr := h.terminalHasAccess(id); accessErr != nil {
		form.GeneralError = accessErr.Error()
	} else if granted {
		form.GeneralError = "Revoke every user's access before deleting this terminal."
	} else {
		deleted, deleteErr := h.terminals.Delete(id)
		if deleteErr != nil {
			form.GeneralError = deleteErr.Error()
		} else {
			form = terminalForm{Errors: map[string]string{}, Notice: "Deleted " + deleted.HostName + "."}
		}
	}
	data := h.terminalsPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentDeleted")
			w.Header().Set("HX-Push-Url", terminalBasePath)
		}
		h.render(w, "terminals-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, terminalBasePath+"?deleted=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func terminalFormFromRequest(r *http.Request) (terminalForm, terminalInput) {
	form := terminalForm{HostName: r.FormValue("host_name"), Errors: map[string]string{}}
	input := terminalInput{HostName: form.HostName}.normalized()
	if !gamecontent.ValidTerminalHostName(input.HostName) {
		form.Errors["host_name"] = "Use lowercase letters, digits, underscores, hyphens, or dots; start and end with a letter or digit."
	}
	return form, input
}

func (h *editorHandler) serveTerminalList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.terminalsPage(r.URL.Query().Get("q"), terminalForm{Errors: map[string]string{}})
	if isHTMX(r) {
		h.render(w, "terminal-catalog", data)
		return
	}
	http.Redirect(w, r, terminalBasePath+"?q="+url.QueryEscape(data.Query), http.StatusSeeOther)
}

func (h *editorHandler) terminalsPage(query string, form terminalForm) pageData {
	data := basePage("content", "terminals", "Terminals", "terminals")
	data.Query = strings.TrimSpace(query)
	data.TerminalForm = form
	if err := h.validateEditorRelationships(); err != nil {
		data.StoreError = err.Error()
		return data
	}
	terminals, err := h.terminals.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	queryLower := strings.ToLower(data.Query)
	for _, terminal := range terminals {
		if queryLower != "" && !strings.Contains(strings.ToLower(terminal.HostName), queryLower) {
			continue
		}
		data.Terminals = append(data.Terminals, terminalRow{
			ID: terminal.ID, HostName: terminal.HostName, Active: terminal.ID == form.ID,
		})
	}
	sort.SliceStable(data.Terminals, func(i, j int) bool {
		left := strings.ToLower(data.Terminals[i].HostName)
		right := strings.ToLower(data.Terminals[j].HostName)
		if left == right {
			return data.Terminals[i].ID < data.Terminals[j].ID
		}
		return left < right
	})
	return data
}

func formFromTerminal(terminal gamecontent.Terminal) terminalForm {
	return terminalForm{
		ID: terminal.ID, HostName: terminal.HostName, Errors: map[string]string{},
	}
}
