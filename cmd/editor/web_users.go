package main

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const userBasePath = "/content/users"

func (h *editorHandler) routeUserID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, userBasePath+"/")
	if strings.HasSuffix(rest, "/delete") {
		id := strings.TrimSuffix(rest, "/delete")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		h.deleteUser(w, r, id)
		return
	}
	if rest == "" || strings.Contains(rest, "/") {
		http.NotFound(w, r)
		return
	}
	h.serveUsers(w, r, rest)
}

func (h *editorHandler) serveUsers(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodPost {
		h.saveUser(w, r, id)
		return
	}
	form := userForm{Errors: map[string]string{}}
	if id != "" {
		user, err := h.users.Get(id)
		if errors.Is(err, errUserNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			form.GeneralError = err.Error()
		} else {
			form = formFromUser(user)
			if r.URL.Query().Get("saved") == "1" {
				form.Notice = "Saved " + user.Username + "."
			}
		}
	}
	data := h.usersPage(r.URL.Query().Get("q"), form)
	if isHTMX(r) && (id != "" || r.URL.Path == userBasePath+"/new") {
		h.render(w, "user-editor", data)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) saveUser(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid user form", http.StatusBadRequest)
		return
	}
	form := userForm{ID: id, Username: r.FormValue("username"), Password: r.FormValue("password"), Errors: map[string]string{}}
	input := userInput{Username: form.Username, Password: form.Password}.normalized()
	if !gamecontent.ValidUsername(input.Username) {
		form.Errors["username"] = "Use lowercase letters, digits, underscores, or hyphens; start with a letter or underscore."
	}
	if strings.TrimSpace(input.Password) == "" {
		form.Errors["password"] = "Fictional Password is required."
	}
	if len(form.Errors) == 0 && id != "" {
		if err := h.validateUserRename(id, input.Username); err != nil {
			form.Errors["username"] = err.Error()
		}
	}
	if len(form.Errors) == 0 {
		var user gamecontent.User
		var err error
		if id == "" {
			user, err = h.users.Create(input)
		} else {
			user, err = h.users.Update(id, input)
		}
		switch {
		case errors.Is(err, errUserNotFound):
			http.NotFound(w, r)
			return
		case err != nil:
			form.GeneralError = err.Error()
		default:
			form = formFromUser(user)
			form.Notice = "Saved " + user.Username + "."
		}
	}
	data := h.usersPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentSaved")
			w.Header().Set("HX-Push-Url", userBasePath+"/"+url.PathEscape(form.ID))
		}
		h.render(w, "users-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, userBasePath+"/"+url.PathEscape(form.ID)+"?saved=1", http.StatusSeeOther)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) deleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	user, err := h.users.Get(id)
	if errors.Is(err, errUserNotFound) {
		http.NotFound(w, r)
		return
	}
	form := formFromUser(user)
	if err != nil {
		form.GeneralError = err.Error()
	} else if granted, accessErr := h.userHasAccess(id); accessErr != nil {
		form.GeneralError = accessErr.Error()
	} else if granted {
		form.GeneralError = "Remove this user's terminal access before deleting the user."
	} else if deleted, deleteErr := h.users.Delete(id); deleteErr != nil {
		form.GeneralError = deleteErr.Error()
	} else {
		form = userForm{Errors: map[string]string{}, Notice: "Deleted " + deleted.Username + "."}
	}
	data := h.usersPage("", form)
	if isHTMX(r) {
		if form.Notice != "" {
			w.Header().Set("HX-Trigger", "contentDeleted")
			w.Header().Set("HX-Push-Url", userBasePath)
		}
		h.render(w, "users-body", data)
		return
	}
	if form.Notice != "" {
		http.Redirect(w, r, userBasePath, http.StatusSeeOther)
		return
	}
	h.renderPage(w, r, data)
}

func (h *editorHandler) serveUserList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := h.usersPage(r.URL.Query().Get("q"), userForm{Errors: map[string]string{}})
	if isHTMX(r) {
		h.render(w, "user-catalog", data)
		return
	}
	http.Redirect(w, r, userBasePath+"?q="+url.QueryEscape(data.Query), http.StatusSeeOther)
}

func (h *editorHandler) usersPage(query string, form userForm) pageData {
	data := basePage("content", "users", "Users", "users")
	data.Query = strings.TrimSpace(query)
	data.UserForm = form
	if err := h.validateAuthRelationships(); err != nil {
		data.StoreError = err.Error()
		return data
	}
	users, err := h.users.List()
	if err != nil {
		data.StoreError = err.Error()
		return data
	}
	queryLower := strings.ToLower(data.Query)
	for _, user := range users {
		if queryLower == "" || strings.Contains(strings.ToLower(user.Username), queryLower) {
			data.Users = append(data.Users, userRow{ID: user.ID, Username: user.Username, Active: user.ID == form.ID})
		}
	}
	sort.SliceStable(data.Users, func(i, j int) bool {
		left, right := strings.ToLower(data.Users[i].Username), strings.ToLower(data.Users[j].Username)
		if left == right {
			return data.Users[i].ID < data.Users[j].ID
		}
		return left < right
	})
	return data
}

func formFromUser(user gamecontent.User) userForm {
	return userForm{ID: user.ID, Username: user.Username, Password: user.Password, Errors: map[string]string{}}
}
