package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const corporationBasePath = "/content/corporations"
const hostNetworkBasePath = corporationBasePath

func (h *editorHandler) routeHostNetworkID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, hostNetworkBasePath+"/")
	parts := strings.Split(rest, "/")
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	switch {
	case len(parts) == 1:
		h.serveHostNetworks(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "delete":
		h.deleteHostNetwork(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "terminals":
		h.serveNetworkTerminals(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "assign":
		h.assignExistingTerminal(w, r, parts[0])
	case len(parts) == 3 && parts[1] == "unassign":
		h.unassignTerminal(w, r, parts[0], parts[2])
	default:
		http.NotFound(w, r)
	}
}

func (h *editorHandler) serveHostNetworks(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodPost {
		h.saveHostNetwork(w, r, id)
		return
	}
	form := hostNetworkForm{}
	if id == "" && r.URL.Query().Get("deleted") == "1" {
		form.Notice = "Corporation deleted."
	}
	if id != "" {
		network, err := h.networks.Get(id)
		if errors.Is(err, errHostNetworkNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form = hostNetworkForm{ID: network.ID, Name: network.Name}
			if r.URL.Query().Get("saved") == "1" {
				form.Notice = "Saved " + network.Name + "."
			}
		}
	}
	data := h.hostNetworksPage(r.URL.Query().Get("q"), form, "details")
	if isHTMX(r) && (id != "" || r.URL.Path == hostNetworkBasePath+"/new") {
		h.render(w, "host-network-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) saveHostNetwork(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid host-network form", http.StatusBadRequest)
		return
	}
	form := hostNetworkForm{ID: id, Name: r.FormValue("name")}
	if strings.TrimSpace(form.Name) == "" {
		form.NameError = "Corporation Name is required."
	} else {
		var network gamecontent.HostNetwork
		var err error
		if id == "" {
			network, err = h.networks.Create(form.Name)
		} else {
			network, err = h.networks.Update(id, form.Name)
		}
		switch {
		case errors.Is(err, errHostNetworkNotFound):
			http.NotFound(w, r)
			return
		case err != nil:
			form.GeneralError = err.Error()
		default:
			form = hostNetworkForm{ID: network.ID, Name: network.Name, Notice: "Saved " + network.Name + "."}
		}
	}
	data := h.hostNetworksPage("", form, "details")
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
			w.Header().Set("HX-Push-Url", hostNetworkBasePath+"/"+url.PathEscape(form.ID))
		}
		h.render(w, "host-networks-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, hostNetworkBasePath+"/"+url.PathEscape(form.ID)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) deleteHostNetwork(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	network, err := h.networks.Get(id)
	if errors.Is(err, errHostNetworkNotFound) {
		http.NotFound(w, r)
		return
	}
	form := hostNetworkForm{ID: network.ID, Name: network.Name}
	if err != nil {
		form.GeneralError = err.Error()
	} else if assigned, listErr := h.assignmentsForNetwork(id); listErr != nil {
		form.GeneralError = listErr.Error()
	} else if len(assigned) != 0 {
		form.GeneralError = "Unassign every terminal before deleting this corporation."
	} else if _, err = h.networks.Delete(id); err != nil {
		form.GeneralError = err.Error()
	} else {
		form = hostNetworkForm{Notice: "Deleted " + network.Name + "."}
	}
	data := h.hostNetworksPage("", form, "details")
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentDeleted")
			w.Header().Set("HX-Push-Url", hostNetworkBasePath)
		}
		h.render(w, "host-networks-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, hostNetworkBasePath+"?deleted=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) serveNetworkTerminals(w http.ResponseWriter, r *http.Request, networkID string) {
	if r.Method == http.MethodPost {
		h.createNetworkTerminal(w, r, networkID)
		return
	}
	network, err := h.networks.Get(networkID)
	if errors.Is(err, errHostNetworkNotFound) {
		http.NotFound(w, r)
		return
	}
	form := hostNetworkForm{ID: network.ID, Name: network.Name}
	if r.URL.Query().Get("saved") == "1" {
		form.Notice = "Network terminals updated."
	}
	data := h.hostNetworksPage("", form, "terminals")
	if isHTMX(r) {
		h.render(w, "host-network-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) createNetworkTerminal(w http.ResponseWriter, r *http.Request, networkID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid terminal form", http.StatusBadRequest)
		return
	}
	network, err := h.networks.Get(networkID)
	if errors.Is(err, errHostNetworkNotFound) {
		http.NotFound(w, r)
		return
	}
	networkForm := hostNetworkForm{ID: network.ID, Name: network.Name}
	form, input := terminalFormFromRequest(r)
	if len(form.Errors) == 0 {
		if collision, collisionErr := h.networkHasHostname(networkID, "", input.HostName); collisionErr != nil {
			form.GeneralError = collisionErr.Error()
		} else if collision {
			form.Errors["host_name"] = "Machine Hostname must be unique within this network."
		} else {
			terminal, createErr := h.terminals.Create(input)
			if createErr != nil {
				form.GeneralError = createErr.Error()
			} else if _, assignErr := h.assignments.Assign(terminal.ID, networkID); assignErr != nil {
				form.GeneralError = "Terminal was created but could not be assigned: " + assignErr.Error()
			} else {
				form = terminalForm{Errors: map[string]string{}, Notice: "Created and assigned " + terminal.HostName + "."}
			}
		}
	}
	data := h.hostNetworksPage("", networkForm, "terminals")
	data.NestedTerminalForm = form
	h.renderNetworkTerminalsResponse(w, r, data, networkID)
}

func (h *editorHandler) assignExistingTerminal(w http.ResponseWriter, r *http.Request, networkID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid network assignment form", http.StatusBadRequest)
		return
	}
	network, err := h.networks.Get(networkID)
	if errors.Is(err, errHostNetworkNotFound) {
		http.NotFound(w, r)
		return
	}
	networkForm := hostNetworkForm{ID: network.ID, Name: network.Name}
	terminalID := r.FormValue("terminal_id")
	terminal, terminalErr := h.terminals.Get(terminalID)
	switch {
	case errors.Is(terminalErr, errTerminalNotFound):
		networkForm.GeneralError = "Choose an existing unassigned terminal."
	case terminalErr != nil:
		networkForm.GeneralError = terminalErr.Error()
	default:
		if _, assignedErr := h.assignments.GetByTerminal(terminalID); assignedErr == nil {
			networkForm.GeneralError = "That terminal is already assigned to a corporation."
		} else if !errors.Is(assignedErr, errNetworkAssignmentNotFound) {
			networkForm.GeneralError = assignedErr.Error()
		} else if collision, collisionErr := h.networkHasHostname(networkID, terminalID, terminal.HostName); collisionErr != nil {
			networkForm.GeneralError = collisionErr.Error()
		} else if collision {
			networkForm.GeneralError = "That hostname is already used within this network."
		} else if _, assignErr := h.assignments.Assign(terminalID, networkID); assignErr != nil {
			networkForm.GeneralError = assignErr.Error()
		} else {
			networkForm.Notice = "Assigned " + terminal.HostName + "."
		}
	}
	data := h.hostNetworksPage("", networkForm, "terminals")
	h.renderNetworkTerminalsResponse(w, r, data, networkID)
}

func (h *editorHandler) unassignTerminal(w http.ResponseWriter, r *http.Request, networkID, terminalID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	network, err := h.networks.Get(networkID)
	if errors.Is(err, errHostNetworkNotFound) {
		http.NotFound(w, r)
		return
	}
	networkForm := hostNetworkForm{ID: network.ID, Name: network.Name}
	assignment, assignmentErr := h.assignments.GetByTerminal(terminalID)
	if errors.Is(assignmentErr, errNetworkAssignmentNotFound) || (assignmentErr == nil && assignment.HostNetworkID != networkID) {
		http.NotFound(w, r)
		return
	}
	if assignmentErr != nil {
		networkForm.GeneralError = assignmentErr.Error()
	} else if _, err := h.assignments.Unassign(terminalID); err != nil {
		networkForm.GeneralError = err.Error()
	} else {
		terminal, getErr := h.terminals.Get(terminalID)
		if getErr == nil {
			networkForm.Notice = "Unassigned " + terminal.HostName + "."
		} else {
			networkForm.Notice = "Terminal unassigned."
		}
	}
	data := h.hostNetworksPage("", networkForm, "terminals")
	h.renderNetworkTerminalsResponse(w, r, data, networkID)
}

func (h *editorHandler) renderNetworkTerminalsResponse(w http.ResponseWriter, r *http.Request, data pageData, networkID string) {
	saved := data.NetworkForm.Notice != "" || data.NestedTerminalForm.Notice != ""
	if isHTMX(r) {
		if saved {
			w.Header().Set("HX-Trigger", "contentSaved")
		}
		h.render(w, "host-network-editor", data)
		return
	}
	if saved {
		http.Redirect(w, r, hostNetworkBasePath+"/"+url.PathEscape(networkID)+"/terminals?saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) serveHostNetworkList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.hostNetworksPage(r.URL.Query().Get("q"), hostNetworkForm{}, "details")
	if isHTMX(r) {
		h.render(w, "host-network-catalog", data)
		return
	}
	http.Redirect(w, r, hostNetworkBasePath+"?q="+url.QueryEscape(data.Query), http.StatusSeeOther)
}

func (h *editorHandler) hostNetworksPage(query string, form hostNetworkForm, view string) pageData {
	data := basePage("content", "corporations", "Corporations", "host-networks")
	data.Query = strings.TrimSpace(query)
	data.NetworkForm = form
	data.NetworkView = view
	data.NestedTerminalForm = terminalForm{Errors: map[string]string{}}
	networks, err := h.networks.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	if err := h.validateNetworkRelations(); err != nil {
		data.StoreError = err.Error()
		return data
	}
	queryLower := strings.ToLower(data.Query)
	for _, network := range networks {
		if queryLower != "" && !strings.Contains(strings.ToLower(network.Name), queryLower) {
			continue
		}
		data.HostNetworks = append(data.HostNetworks, hostNetworkRow{ID: network.ID, Name: network.Name, Active: network.ID == form.ID})
	}
	sort.SliceStable(data.HostNetworks, func(i, j int) bool {
		left := strings.ToLower(data.HostNetworks[i].Name)
		right := strings.ToLower(data.HostNetworks[j].Name)
		if left == right {
			return data.HostNetworks[i].ID < data.HostNetworks[j].ID
		}
		return left < right
	})
	if form.ID != "" {
		h.populateNetworkTerminals(&data, form.ID)
	}
	return data
}

func (h *editorHandler) populateNetworkTerminals(data *pageData, networkID string) {
	terminals, err := h.terminals.List()
	if err != nil {
		data.StoreError = err.Error()
		return
	}
	assignments, err := h.assignments.List()
	if err != nil {
		data.StoreError = err.Error()
		return
	}
	assignedByTerminal := make(map[string]string, len(assignments))
	for _, assignment := range assignments {
		assignedByTerminal[assignment.TerminalID] = assignment.HostNetworkID
	}
	for _, terminal := range terminals {
		row := terminalRow{ID: terminal.ID, HostName: terminal.HostName}
		switch assignedByTerminal[terminal.ID] {
		case networkID:
			data.AssignedTerminals = append(data.AssignedTerminals, row)
		case "":
			data.UnassignedTerminals = append(data.UnassignedTerminals, row)
		}
	}
	sortTerminalRows(data.AssignedTerminals)
	sortTerminalRows(data.UnassignedTerminals)
}

func sortTerminalRows(rows []terminalRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		left := strings.ToLower(rows[i].HostName)
		right := strings.ToLower(rows[j].HostName)
		if left == right {
			return rows[i].ID < rows[j].ID
		}
		return left < right
	})
}

func (h *editorHandler) assignmentsForNetwork(networkID string) ([]gamecontent.NetworkAssignment, error) {
	assignments, err := h.assignments.List()
	if err != nil {
		return nil, err
	}
	var out []gamecontent.NetworkAssignment
	for _, assignment := range assignments {
		if assignment.HostNetworkID == networkID {
			out = append(out, assignment)
		}
	}
	return out, nil
}

func (h *editorHandler) networkHasHostname(networkID, exceptTerminalID, hostname string) (bool, error) {
	assignments, err := h.assignmentsForNetwork(networkID)
	if err != nil {
		return false, err
	}
	for _, assignment := range assignments {
		if assignment.TerminalID == exceptTerminalID {
			continue
		}
		terminal, err := h.terminals.Get(assignment.TerminalID)
		if err != nil {
			return false, err
		}
		if strings.EqualFold(terminal.HostName, hostname) {
			return true, nil
		}
	}
	return false, nil
}

func (h *editorHandler) terminalNetwork(terminalID string) (string, error) {
	assignment, err := h.assignments.GetByTerminal(terminalID)
	if errors.Is(err, errNetworkAssignmentNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return assignment.HostNetworkID, nil
}

func (h *editorHandler) validateNetworkRelations() error {
	networks, err := h.networks.List()
	if err != nil {
		return err
	}
	terminals, err := h.terminals.List()
	if err != nil {
		return err
	}
	assignments, err := h.assignments.List()
	if err != nil {
		return err
	}
	networkIDs := make(map[string]struct{}, len(networks))
	for _, network := range networks {
		networkIDs[network.ID] = struct{}{}
	}
	terminalByID := make(map[string]gamecontent.Terminal, len(terminals))
	for _, terminal := range terminals {
		terminalByID[terminal.ID] = terminal
	}
	hostnames := make(map[string]map[string]string)
	for _, assignment := range assignments {
		if _, exists := networkIDs[assignment.HostNetworkID]; !exists {
			return fmt.Errorf("terminal %s references missing corporation %s", assignment.TerminalID, assignment.HostNetworkID)
		}
		terminal, exists := terminalByID[assignment.TerminalID]
		if !exists {
			return fmt.Errorf("corporation %s references missing terminal %s", assignment.HostNetworkID, assignment.TerminalID)
		}
		key := strings.ToLower(terminal.HostName)
		if hostnames[assignment.HostNetworkID] == nil {
			hostnames[assignment.HostNetworkID] = map[string]string{}
		}
		if otherID, exists := hostnames[assignment.HostNetworkID][key]; exists {
			return fmt.Errorf("corporation %s assigns duplicate hostname %q to terminals %s and %s", assignment.HostNetworkID, terminal.HostName, otherID, terminal.ID)
		}
		hostnames[assignment.HostNetworkID][key] = terminal.ID
	}
	return nil
}
