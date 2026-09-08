package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// API operations deliberately call the same application commands as the forms.
// A world-scoped URL never silently follows the browser to another world.
func (h *editorHandler) serveAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(status int, code string, err error) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": err.Error()}})
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "worlds" {
		fail(404, "not_found", fmt.Errorf("unknown API route"))
		return
	}
	world := h.worldName
	if parts[3] != world {
		fail(409, "world_mismatch", fmt.Errorf("requested world is not loaded; loaded world is %s", world))
		return
	}
	h.mutationMu.Lock()
	defer h.mutationMu.Unlock()
	snapshot := func() (map[string]any, string, error) {
		c, err := game.ReadCatalogs(filepath.Dir(h.rooms.path))
		if err != nil {
			return nil, "", err
		}
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, "", err
		}
		revision := fmt.Sprintf("\"%x\"", sha256.Sum256(raw))
		tree, err := h.worldSnapshot()
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"world": world, "revision": revision, "catalogs": c, "hierarchy": tree}, revision, nil
	}
	before, revision, err := snapshot()
	if err != nil {
		fail(500, "read_failed", err)
		return
	}
	w.Header().Set("ETag", revision)
	if len(parts) >= 5 && parts[4] == "quests" {
		h.serveQuestAPI(w, r, parts, revision)
		return
	}
	if r.Method == http.MethodGet {
		if len(parts) == 4 {
			json.NewEncoder(w).Encode(before)
			return
		}
		if (len(parts) == 6 || len(parts) == 7) && parts[4] == "entities" {
			catalogKey := map[string]string{"hub": "Hubs", "location": "Locations", "room": "Rooms", "world_item": "WorldItems", "npc": "NPCs", "terminal": "Terminals"}[parts[5]]
			raw, _ := json.Marshal(before["catalogs"])
			var catalogs map[string]map[string]json.RawMessage
			json.Unmarshal(raw, &catalogs)
			key := map[string]string{"hub": "hubs", "location": "locations", "room": "rooms", "world_item": "world_items", "npc": "npcs", "terminal": "terminals"}[parts[5]]
			rows := []map[string]any{}
			json.Unmarshal(catalogs[catalogKey][key], &rows)
			if catalogKey == "" {
				fail(404, "not_found", fmt.Errorf("unknown entity kind"))
				return
			}
			if len(parts) == 6 {
				json.NewEncoder(w).Encode(rows)
				return
			}
			for _, row := range rows {
				if row["id"] == parts[6] {
					json.NewEncoder(w).Encode(row)
					return
				}
			}
			fail(404, "not_found", fmt.Errorf("entity not found"))
			return
		}
		if len(parts) == 5 && parts[4] == "validation" {
			json.NewEncoder(w).Encode(h.playPage(""))
			return
		}
		fail(404, "not_found", fmt.Errorf("read the world snapshot or validation endpoint"))
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
		fail(405, "method_not_allowed", fmt.Errorf("use GET, POST, PUT, or DELETE"))
		return
	}
	if (r.Header.Get("Origin") != "" && !sameOrigin(r)) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		fail(403, "origin", fmt.Errorf("cross-origin writes are forbidden"))
		return
	}
	if r.Header.Get("If-Match") == "" {
		fail(428, "revision_required", fmt.Errorf("send the world ETag in If-Match"))
		return
	}
	if r.Header.Get("If-Match") != revision {
		fail(409, "stale_revision", fmt.Errorf("world changed; read it again before writing"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(415, "content_type", fmt.Errorf("send application/json"))
		return
	}
	var fields map[string]string
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err = decoder.Decode(&fields); err != nil {
		fail(400, "invalid_json", err)
		return
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		fail(400, "invalid_json", fmt.Errorf("expected one JSON object"))
		return
	}
	values := url.Values{}
	for k, v := range fields {
		values.Set(k, v)
	}
	command := r.Clone(r.Context())
	command.Form = values
	command.PostForm = values
	if err = h.validateEditorRelationships(); err != nil {
		fail(409, "relationships", err)
		return
	}
	target := ""
	switch {
	case (len(parts) == 6 || len(parts) == 7) && parts[4] == "entities":
		kind := parts[5]
		id := ""
		if len(parts) == 7 {
			id = parts[6]
		}
		if r.Method == http.MethodDelete {
			if id == "" {
				err = fmt.Errorf("entity ID required")
			} else {
				err = h.deleteEntity(kind, id)
			}
		} else if (r.Method == http.MethodPost && id == "") || (r.Method == http.MethodPut && id != "") {
			id, err = h.saveEntity(kind, id, values)
			target = entityURL(kind, id)
		} else {
			err = fmt.Errorf("POST creates a collection member; PUT replaces an existing entity")
		}
	case len(parts) == 5 && parts[4] == "play-settings":
		if r.Method != http.MethodPost {
			err = fmt.Errorf("play settings require POST")
			break
		}
		value := values.Get("cell")
		eligible := value == ""
		for _, o := range h.playPage("").Options {
			if o.Value == value {
				eligible = true
			}
		}
		if !eligible {
			err = fmt.Errorf("Choose a placed cell with placed ancestry.")
			break
		}
		err = h.playSettings.Update(func(s *content.PlaySettings) error {
			s.Start = nil
			if value != "" {
				k, id, _ := strings.Cut(value, ":")
				s.Start = &content.CellRef{Kind: k, ID: id}
			}
			return nil
		})
	case len(parts) == 5 && parts[4] == "hubs":
		if r.Method != http.MethodPost {
			err = fmt.Errorf("Hub creation requires POST")
			break
		}
		v, e := h.hubs.Create(values.Get("name"))
		err = e
		target = entityURL("hub", v.ID)
	case len(parts) == 7 && parts[6] == "actions":
		if r.Method != http.MethodPost {
			err = fmt.Errorf("actions require POST")
			break
		}
		kind := map[string]string{"hubs": "hub", "locations": "location", "rooms": "room"}[parts[4]]
		if kind == "" {
			err = fmt.Errorf("unknown entity kind")
			break
		}
		if _, err = h.spatialDefinition(kind, parts[5]); err != nil {
			break
		}
		target, err = h.applyDrillAction(kind, parts[5], command)
	default:
		err = fmt.Errorf("unknown mutation route")
	}
	if err != nil {
		fail(422, "validation", err)
		return
	}
	after, next, err := snapshot()
	if err != nil {
		fail(500, "read_failed", err)
		return
	}
	after["editor_url"] = target
	after["created"] = apiCreated(before, after)
	w.Header().Set("ETag", next)
	json.NewEncoder(w).Encode(after)
}

func apiCreated(before, after map[string]any) []string {
	ids := func(v any) map[string]bool {
		raw, _ := json.Marshal(v)
		var tree any
		json.Unmarshal(raw, &tree)
		out := map[string]bool{}
		var walk func(any)
		walk = func(n any) {
			switch x := n.(type) {
			case map[string]any:
				if id, ok := x["id"].(string); ok {
					out[id] = true
				}
				for _, child := range x {
					walk(child)
				}
			case []any:
				for _, child := range x {
					walk(child)
				}
			}
		}
		walk(tree)
		return out
	}
	old := ids(before["catalogs"])
	created := []string{}
	for id := range ids(after["catalogs"]) {
		if !old[id] {
			created = append(created, id)
		}
	}
	return created
}
