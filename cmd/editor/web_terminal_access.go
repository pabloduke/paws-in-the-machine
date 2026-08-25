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

func (h *editorHandler) serveTerminalAccess(w http.ResponseWriter, r *http.Request, terminalID string) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	terminal, err := h.terminals.Get(terminalID)
	if errors.Is(err, errTerminalNotFound) {
		http.NotFound(w, r)
		return
	}
	form := formFromTerminal(terminal)
	if err != nil {
		form.GeneralError = err.Error()
	} else if r.URL.Query().Get("saved") == "1" {
		form.Notice = "Terminal access updated."
	}
	data := h.terminalsPage("", form)
	data.TerminalView = "access"
	h.populateTerminalAccess(&data, terminalID)
	if isHTMX(r) {
		h.render(w, "terminal-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) grantTerminalAccess(w http.ResponseWriter, r *http.Request, terminalID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid terminal access form", http.StatusBadRequest)
		return
	}
	terminal, err := h.terminals.Get(terminalID)
	if errors.Is(err, errTerminalNotFound) {
		http.NotFound(w, r)
		return
	}
	form := formFromTerminal(terminal)
	userID := r.FormValue("user_id")
	user, userErr := h.users.Get(userID)
	switch {
	case errors.Is(userErr, errUserNotFound):
		form.GeneralError = "Choose an existing user."
	case userErr != nil:
		form.GeneralError = userErr.Error()
	case err != nil:
		form.GeneralError = err.Error()
	default:
		if collisionErr := h.terminalUsernameTaken(terminalID, userID, user.Username); collisionErr != nil {
			form.GeneralError = collisionErr.Error()
		} else if _, grantErr := h.access.Grant(userID, terminalID); grantErr != nil {
			form.GeneralError = grantErr.Error()
		} else {
			form.Notice = "Granted access to " + user.Username + "."
		}
	}
	h.renderTerminalAccessResponse(w, r, terminalID, form)
}

func (h *editorHandler) revokeTerminalAccess(w http.ResponseWriter, r *http.Request, terminalID, userID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	terminal, err := h.terminals.Get(terminalID)
	if errors.Is(err, errTerminalNotFound) {
		http.NotFound(w, r)
		return
	}
	form := formFromTerminal(terminal)
	if err != nil {
		form.GeneralError = err.Error()
	} else if _, err := h.access.Revoke(userID, terminalID); errors.Is(err, errTerminalAccessNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		form.GeneralError = err.Error()
	} else {
		form.Notice = "Terminal access revoked."
	}
	h.renderTerminalAccessResponse(w, r, terminalID, form)
}

func (h *editorHandler) renderTerminalAccessResponse(w http.ResponseWriter, r *http.Request, terminalID string, form terminalForm) {
	data := h.terminalsPage("", form)
	data.TerminalView = "access"
	h.populateTerminalAccess(&data, terminalID)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
		}
		h.render(w, "terminal-editor", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, terminalBasePath+"/"+url.PathEscape(terminalID)+"/access?saved=1", http.StatusSeeOther)
		return
	}
	h.render(w, "page", data)
}

func (h *editorHandler) populateTerminalAccess(data *pageData, terminalID string) {
	users, err := h.users.List()
	if err != nil {
		data.StoreError = err.Error()
		return
	}
	grants, err := h.access.List()
	if err != nil {
		data.StoreError = err.Error()
		return
	}
	granted := make(map[string]bool)
	for _, grant := range grants {
		if grant.TerminalID == terminalID {
			granted[grant.UserID] = true
		}
	}
	for _, user := range users {
		row := userRow{ID: user.ID, Username: user.Username}
		if granted[user.ID] {
			data.GrantedUsers = append(data.GrantedUsers, row)
		} else {
			data.AvailableUsers = append(data.AvailableUsers, row)
		}
	}
	sortUserRows(data.GrantedUsers)
	sortUserRows(data.AvailableUsers)
}

func sortUserRows(rows []userRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := strings.ToLower(rows[i].Username), strings.ToLower(rows[j].Username)
		if left == right {
			return rows[i].ID < rows[j].ID
		}
		return left < right
	})
}

func (h *editorHandler) terminalHasAccess(terminalID string) (bool, error) {
	grants, err := h.access.List()
	if err != nil {
		return false, err
	}
	for _, grant := range grants {
		if grant.TerminalID == terminalID {
			return true, nil
		}
	}
	return false, nil
}

func (h *editorHandler) userHasAccess(userID string) (bool, error) {
	grants, err := h.access.List()
	if err != nil {
		return false, err
	}
	for _, grant := range grants {
		if grant.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (h *editorHandler) terminalUsernameTaken(terminalID, exceptUserID, username string) error {
	users, err := h.users.List()
	if err != nil {
		return err
	}
	grants, err := h.access.List()
	if err != nil {
		return err
	}
	byID := make(map[string]gamecontent.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}
	for _, grant := range grants {
		if grant.TerminalID != terminalID || grant.UserID == exceptUserID {
			continue
		}
		if user, ok := byID[grant.UserID]; ok && strings.EqualFold(user.Username, username) {
			return fmt.Errorf("Username %q is already granted access to this terminal.", username)
		}
	}
	return nil
}

func (h *editorHandler) validateUserRename(userID, username string) error {
	grants, err := h.access.List()
	if err != nil {
		return err
	}
	for _, grant := range grants {
		if grant.UserID == userID {
			if err := h.terminalUsernameTaken(grant.TerminalID, userID, username); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *editorHandler) validateAuthRelationships() error {
	users, err := h.users.List()
	if err != nil {
		return err
	}
	terminals, err := h.terminals.List()
	if err != nil {
		return err
	}
	grants, err := h.access.List()
	if err != nil {
		return err
	}
	userByID := make(map[string]gamecontent.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}
	terminalIDs := make(map[string]bool, len(terminals))
	for _, terminal := range terminals {
		terminalIDs[terminal.ID] = true
	}
	seen := make(map[string]string)
	for _, grant := range grants {
		user, ok := userByID[grant.UserID]
		if !ok {
			return fmt.Errorf("terminal %s grants access to missing user %s", grant.TerminalID, grant.UserID)
		}
		if !terminalIDs[grant.TerminalID] {
			return fmt.Errorf("user %s references missing terminal %s", grant.UserID, grant.TerminalID)
		}
		key := grant.TerminalID + "\x00" + strings.ToLower(user.Username)
		if otherID, exists := seen[key]; exists {
			return fmt.Errorf("terminal %s grants duplicate username %q to users %s and %s", grant.TerminalID, user.Username, otherID, user.ID)
		}
		seen[key] = user.ID
	}
	return nil
}

func (h *editorHandler) validateEditorRelationships() error {
	if err := h.validateNetworkRelations(); err != nil {
		return err
	}
	if err := h.validateAuthRelationships(); err != nil {
		return err
	}
	return h.validateSpatialRelationships()
}
