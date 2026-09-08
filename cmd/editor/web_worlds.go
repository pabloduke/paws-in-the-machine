package main

import (
	"errors"
	"net/http"
	"strings"
)

const worldsBasePath = "/worlds"

type worldRow struct {
	Name    string
	Dir     string
	Current bool
	IsMain  bool
}

type worldsPage struct {
	Worlds               []worldRow
	CurrentName          string
	CopyFrom             string
	Notice, GeneralError string
	NameError            string
	Unavailable          bool
}

func (h *editorHandler) routeWorlds(w http.ResponseWriter, r *http.Request) {
	if h.worlds == nil {
		// A handler built without a registry (tests, or a future
		// embedding) has exactly one world and no way to switch.
		h.renderPage(w, r, h.worldsPageData(worldsPage{Unavailable: true}))
		return
	}
	switch r.URL.Path {
	case worldsBasePath:
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		h.renderPage(w, r, h.worldsPageData(worldsPage{}))
	case worldsBasePath + "/load":
		h.loadWorld(w, r)
	case worldsBasePath + "/copy":
		h.copyWorld(w, r)
	case worldsBasePath + "/rename", worldsBasePath + "/delete":
		if r.Method != http.MethodPost {
			h.methodNotAllowed(w, http.MethodPost)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form.", http.StatusBadRequest)
			return
		}
		err := h.worlds.manageInactive(r.FormValue("from"), strings.TrimSpace(r.FormValue("name")), r.URL.Path == worldsBasePath+"/delete")
		page := worldsPage{Notice: "World list updated."}
		if err != nil {
			page.Notice = ""
			page.GeneralError = err.Error()
		}
		h.renderPage(w, r, h.worldsPageData(page))
	case worldsBasePath + "/new":
		h.createWorld(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Loading redirects rather than rendering: the response must come from
// the newly loaded world, and this handler still belongs to the old one.
func (h *editorHandler) loadWorld(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid world form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if err := h.worlds.Load(name); err != nil {
		page := worldsPage{GeneralError: worldErrorMessage(err, name)}
		h.renderPage(w, r, h.worldsPageData(page))
		return
	}
	if isHTMX(r) {
		// The whole document changes worlds, including the header, so
		// let the browser navigate rather than swapping a fragment.
		w.Header().Set("HX-Redirect", "/content/hubs")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/content/hubs", http.StatusSeeOther)
}

func (h *editorHandler) copyWorld(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid world form", http.StatusBadRequest)
		return
	}
	from := strings.TrimSpace(r.FormValue("from"))
	to := strings.TrimSpace(r.FormValue("name"))
	page := worldsPage{CopyFrom: from}
	switch {
	case !validWorldName(to):
		page.NameError = worldNameHelp(to)
	default:
		if err := h.worlds.Copy(from, to); err != nil {
			page.GeneralError = worldErrorMessage(err, to)
		} else {
			page.Notice = "Copied " + from + " to " + to + ". Load it when you want to work in it."
		}
	}
	h.renderPage(w, r, h.worldsPageData(page))
}

func (h *editorHandler) createWorld(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid world form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	page := worldsPage{}
	switch {
	case !validWorldName(name):
		page.NameError = worldNameHelp(name)
	default:
		if err := h.worlds.Create(name); err != nil {
			page.GeneralError = worldErrorMessage(err, name)
		} else {
			page.Notice = "Created the empty world " + name + ". Load it to start authoring."
		}
	}
	h.renderPage(w, r, h.worldsPageData(page))
}

func worldNameHelp(name string) string {
	if name == mainWorld {
		return "main is the repository's own content and cannot be reused as a name."
	}
	return "Use letters, digits, hyphens, and underscores; up to 64 characters."
}

func worldErrorMessage(err error, name string) string {
	switch {
	case errors.Is(err, errWorldExists):
		return "A world named " + name + " already exists. Choose another name."
	case errors.Is(err, errWorldNotFound):
		return "There is no world named " + name + "."
	}
	return err.Error()
}

func (h *editorHandler) worldsPageData(page worldsPage) pageData {
	data := basePage("worlds", "worlds", "Worlds", "worlds")
	if h.worlds == nil {
		page.Unavailable = true
		data.Worlds = page
		return data
	}
	if current := h.worlds.Current(); current != nil {
		page.CurrentName = current.name
		if page.CopyFrom == "" {
			page.CopyFrom = current.name
		}
	}
	worlds, err := h.worlds.List()
	if err != nil {
		data.StoreError = err.Error()
		data.Worlds = page
		return data
	}
	for _, world := range worlds {
		page.Worlds = append(page.Worlds, worldRow{
			Name: world.Name, Dir: world.Dir, Current: world.Current, IsMain: world.IsMain,
		})
	}
	data.Worlds = page
	return data
}

// currentWorldName labels every screen, so which world is being edited
// is never a guess. Switching is per-editor, not per-browser-tab.
func (h *editorHandler) currentWorldName() string {
	if h.worlds == nil {
		return ""
	}
	if current := h.worlds.Current(); current != nil {
		return current.name
	}
	return ""
}
