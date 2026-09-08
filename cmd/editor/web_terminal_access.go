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
	h.renderPage(w, r, data)
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

// validateAuthRelationships stops at the first problem, for the write
// gate. authProblems collects them all, for the Overview.
func (h *editorHandler) validateAuthRelationships() error {
	problems, err := h.authProblems()
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return errors.New(problems[0])
	}
	return nil
}

func (h *editorHandler) authProblems() ([]string, error) {
	var problems []string
	report := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	users, err := h.users.List()
	if err != nil {
		return nil, err
	}
	terminals, err := h.terminals.List()
	if err != nil {
		return nil, err
	}
	grants, err := h.access.List()
	if err != nil {
		return nil, err
	}
	userByID := make(map[string]gamecontent.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}
	terminalName := make(map[string]string, len(terminals))
	for _, terminal := range terminals {
		terminalName[terminal.ID] = terminal.HostName
	}
	seen := make(map[string]string)
	for _, grant := range grants {
		user, hasUser := userByID[grant.UserID]
		host, hasTerminal := terminalName[grant.TerminalID]
		switch {
		case !hasUser && !hasTerminal:
			report("An access grant names a User (%s) and a Terminal (%s) that no longer exist.",
				grant.UserID, grant.TerminalID)
			continue
		case !hasUser:
			report("Terminal %q grants access to a User that no longer exists (%s).", host, grant.UserID)
			continue
		case !hasTerminal:
			report("User %q is granted access to a Terminal that no longer exists (%s).",
				user.Username, grant.TerminalID)
			continue
		}
		key := grant.TerminalID + "\x00" + strings.ToLower(user.Username)
		if other, exists := seen[key]; exists {
			report("Terminal %q grants access to two Users named %q (%s and %s).",
				host, user.Username, other, user.ID)
			continue
		}
		seen[key] = user.ID
	}
	return problems, nil
}

func (h *editorHandler) validateEditorRelationships() error {
	if err := h.validateNetworkRelations(); err != nil {
		return err
	}
	if err := h.validateAuthRelationships(); err != nil {
		return err
	}
	if err := h.validateSpatialRelationships(); err != nil {
		return err
	}
	return h.validateContentsRelationships()
}
