package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

type questPage struct {
	Quests                           []content.Quest
	Quest                            content.Quest
	Step                             content.QuestStep
	Targets                          []questTarget
	Revision, Error, Notice, Preview string
	Problems                         []string
}
type questTarget struct{ Value, Name string }

func (h *editorHandler) questCatalogs() (content.Catalogs, string, error) {
	c, err := game.ReadCatalogs(filepath.Dir(h.rooms.path))
	if err != nil {
		return c, "", err
	}
	b, err := json.Marshal(c)
	return c, fmt.Sprintf("\"%x\"", sha256.Sum256(b)), err
}
func (h *editorHandler) writeQuests(c content.Catalogs) error {
	if err := c.Quests.Validate(); err != nil {
		return err
	}
	for _, q := range c.Quests.Quests {
		if q.Enabled {
			if p := c.QuestProblems(q); len(p) > 0 {
				return fmt.Errorf("%s", strings.Join(p, " "))
			}
		}
	}
	b, err := json.MarshalIndent(c.Quests, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := filepath.Join(filepath.Dir(h.rooms.path), "quests.json")
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if old, e := os.ReadFile(path); e == nil {
		if err = writeAtomic(path+".bak", old, 0644); err != nil {
			return err
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	return writeAtomic(path, b, 0644)
}
func prepareQuest(q *content.Quest) error {
	var err error
	if q.ID == "" {
		q.ID, err = newUUIDv4()
		if err != nil {
			return err
		}
	}
	for i := range q.Steps {
		if q.Steps[i].ID == "" {
			q.Steps[i].ID, err = newUUIDv4()
			if err != nil {
				return err
			}
		}
	}
	if q.Steps == nil {
		q.Steps = []content.QuestStep{}
	}
	return nil
}

// saveQuest is the sole mutation path for forms and JSON callers.
func (h *editorHandler) saveQuest(c content.Catalogs, q content.Quest, remove, create bool) (content.Quest, error) {
	if err := prepareQuest(&q); err != nil {
		return q, err
	}
	at := -1
	for i, v := range c.Quests.Quests {
		if v.ID == q.ID {
			at = i
		}
	}
	if create && at >= 0 {
		return q, fmt.Errorf("Quest already exists.")
	}
	if (!create || remove) && at < 0 {
		return q, os.ErrNotExist
	}
	if remove {
		c.Quests.Quests = append(c.Quests.Quests[:at], c.Quests.Quests[at+1:]...)
	} else if at >= 0 {
		c.Quests.Quests[at] = q
	} else {
		c.Quests.Quests = append(c.Quests.Quests, q)
	}
	return q, h.writeQuests(c)
}
func (h *editorHandler) questReferenceProblem(kind, id string) error {
	c, _, err := h.questCatalogs()
	if err != nil {
		return err
	}
	for _, q := range c.Quests.Quests {
		for _, s := range q.Steps {
			if s.Target.Kind == kind && s.Target.ID == id {
				return fmt.Errorf("Referenced by quest %q. Remove its objective reference first.", q.Name)
			}
		}
	}
	return nil
}
func (h *editorHandler) serveQuestAPI(w http.ResponseWriter, r *http.Request, parts []string, revision string) {
	fail := func(status int, err error) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "quest_error", "message": err.Error()}})
	}
	if len(parts) < 5 || len(parts) > 6 {
		fail(404, os.ErrNotExist)
		return
	}
	c, _, err := h.questCatalogs()
	if err != nil {
		fail(500, err)
		return
	}
	id := ""
	if len(parts) == 6 {
		id = parts[5]
	}
	var q content.Quest
	found := false
	for _, v := range c.Quests.Quests {
		if v.ID == id {
			q = v
			found = true
		}
	}
	if r.Method == http.MethodGet {
		if id == "" {
			json.NewEncoder(w).Encode(c.Quests.Quests)
		} else if found {
			json.NewEncoder(w).Encode(q)
		} else {
			fail(404, os.ErrNotExist)
		}
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
		fail(405, fmt.Errorf("Use GET, POST, PUT, DELETE."))
		return
	}
	if (r.Header.Get("Origin") != "" && !sameOrigin(r)) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		fail(403, fmt.Errorf("Cross-origin writes forbidden."))
		return
	}
	if r.Header.Get("If-Match") == "" {
		fail(428, fmt.Errorf("If-Match required."))
		return
	}
	if r.Header.Get("If-Match") != revision {
		fail(409, fmt.Errorf("World changed; reload before saving."))
		return
	}
	create := r.Method == http.MethodPost
	if create && id != "" || !create && id == "" {
		fail(405, fmt.Errorf("POST creates; PUT and DELETE require a quest ID."))
		return
	}
	if !create && !found {
		fail(404, os.ErrNotExist)
		return
	}
	if r.Method != http.MethodDelete {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			fail(415, fmt.Errorf("Send application/json."))
			return
		}
		var submitted content.Quest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&submitted); err != nil {
			fail(400, err)
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			fail(400, fmt.Errorf("Expected one JSON object."))
			return
		}
		if !create && submitted.ID != "" && submitted.ID != id {
			fail(400, fmt.Errorf("Quest ID cannot change."))
			return
		}
		q = submitted
		if !create {
			q.ID = id
		}
	}
	q, err = h.saveQuest(c, q, r.Method == http.MethodDelete, create)
	if err != nil {
		fail(422, err)
		return
	}
	_, rev, err := h.questCatalogs()
	if err != nil {
		fail(500, err)
		return
	}
	w.Header().Set("ETag", rev)
	if create {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(q)
}
func (h *editorHandler) serveQuests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	c, rev, err := h.questCatalogs()
	p := questPage{Quests: c.Quests.Quests, Revision: rev}
	id := strings.TrimPrefix(r.URL.Path, "/content/quests")
	id = strings.Trim(id, "/")
	if id == "new" {
		id = ""
	}
	if strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	found := id == ""
	for _, q := range c.Quests.Quests {
		if q.ID == id {
			p.Quest = q
			found = true
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		p.Error = err.Error()
	}
	if r.Method == http.MethodPost && err == nil {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err = r.ParseForm(); err == nil && r.Form.Get("revision") != rev {
			err = fmt.Errorf("World changed; reload before saving.")
		}
		action := r.Form.Get("action")
		if err == nil && action == "preview" {
			p.Preview = previewQuest(c, p.Quest, r.Form["satisfied"])
		} else if err == nil {
			q := p.Quest
			q.Steps = append([]content.QuestStep(nil), q.Steps...)
			switch action {
			case "save":
				q.Name = r.Form.Get("name")
				q.Description = r.Form.Get("description")
				q.CompletionText = r.Form.Get("completion_text")
				q.Enabled = r.Form.Get("enabled") == "on"
				q.AutoStart = r.Form.Get("auto_start") == "on"
			case "delete":
			case "save-step":
				s := content.QuestStep{ID: r.Form.Get("step_id"), Text: r.Form.Get("text"), Kind: r.Form.Get("kind")}
				s.Target.Kind, s.Target.ID, _ = strings.Cut(r.Form.Get("target"), ":")
				p.Step = s
				if s.ID == "" {
					s.ID, err = newUUIDv4()
					q.Steps = append(q.Steps, s)
				} else {
					at := -1
					for i, v := range q.Steps {
						if v.ID == s.ID {
							at = i
						}
					}
					if at < 0 {
						err = fmt.Errorf("Unknown step.")
					} else {
						q.Steps[at] = s
					}
				}
			case "up", "down", "delete-step":
				at := -1
				for i, s := range q.Steps {
					if s.ID == r.Form.Get("step_id") {
						at = i
					}
				}
				if at < 0 {
					err = fmt.Errorf("Unknown step.")
				} else if action == "delete-step" {
					q.Steps = append(q.Steps[:at], q.Steps[at+1:]...)
				} else {
					dest := at - 1
					if action == "down" {
						dest = at + 1
					}
					if dest >= 0 && dest < len(q.Steps) {
						q.Steps[at], q.Steps[dest] = q.Steps[dest], q.Steps[at]
					}
				}
			default:
				err = fmt.Errorf("Unknown quest action.")
			}
			if err == nil {
				q, err = h.saveQuest(c, q, action == "delete", id == "")
			}
			if err == nil {
				target := "/content/quests/" + q.ID
				if action == "delete" {
					target = "/content/quests"
				}
				http.Redirect(w, r, target, http.StatusSeeOther)
				return
			}
			if action == "save" {
				if id == "" {
					q.ID = ""
				}
				p.Quest = q
			}
		}
		if err != nil {
			p.Error = err.Error()
		}
	}
	for _, i := range c.WorldItems.WorldItems {
		if string(i.Kind) == "takeable" {
			p.Targets = append(p.Targets, questTarget{"world_item:" + i.ID, "Item: " + i.Name})
		}
	}
	for _, v := range c.Locations.Locations {
		p.Targets = append(p.Targets, questTarget{"location:" + v.ID, "Location: " + v.Name})
	}
	for _, v := range c.Rooms.Rooms {
		p.Targets = append(p.Targets, questTarget{"room:" + v.ID, "Room: " + v.Name})
	}
	for _, s := range p.Quest.Steps {
		if s.ID == r.URL.Query().Get("step") && !(r.Method == http.MethodPost && r.Form.Get("action") == "save-step" && p.Error != "") {
			p.Step = s
		}
	}
	if p.Quest.ID != "" {
		p.Problems = c.QuestProblems(p.Quest)
	}
	data := pageData{Title: "Quests", Content: "quests", HeaderTabs: headerTabs("quests"), QuestPage: p}
	h.renderPage(w, r, data)
}
func previewQuest(c content.Catalogs, q content.Quest, satisfied []string) string {
	if problems := c.QuestProblems(q); len(problems) > 0 {
		return strings.Join(problems, "\n")
	}
	// A disposable world supplies hypothetical facts to the same evaluator.
	w := engine.NewWorld()
	room := engine.NewEntity("preview-start", "Preview")
	w.Root.Add(room)
	room.Add(w.Player)
	rq := engine.Quest{ID: q.ID, Name: q.Name, AutoStart: true, CompletionText: q.CompletionText}
	met := map[string]bool{}
	for _, id := range satisfied {
		met[id] = true
	}
	targets := map[string]*engine.Entity{}
	for _, s := range q.Steps {
		key := s.Target.Kind + ":" + s.Target.ID
		target := targets[key]
		if target == nil {
			target = engine.NewEntity(key, key)
			w.Root.Add(target)
			targets[key] = target
		}
		if met[s.ID] {
			if s.Kind == "carry" {
				w.Player.Add(target)
			} else {
				target.Add(w.Player)
			}
		}
		rq.Steps = append(rq.Steps, engine.QuestStep{ID: s.ID, Text: s.Text, Kind: s.Kind, Target: key})
	}
	w.Quests = []engine.Quest{rq}
	w.EvaluateQuests()
	return w.QuestText(true)
}

func (h *editorHandler) relatedQuests(data *pageData) {
	kind, id := data.Drill.Kind, data.Drill.ID
	if data.ItemForm.ID != "" {
		kind, id = "world_item", data.ItemForm.ID
	}
	if data.EntityKey == "rooms" {
		kind, id = "room", data.DescribedForm.ID
	}
	if data.EntityKey == "locations" {
		kind, id = "location", data.DescribedForm.ID
	}
	if id == "" {
		return
	}
	c, _, err := h.questCatalogs()
	if err != nil {
		return
	}
	for _, q := range c.Quests.Quests {
		for _, s := range q.Steps {
			if s.Target.Kind == kind && s.Target.ID == id {
				data.RelatedQuests = append(data.RelatedQuests, crumb{q.Name, "/content/quests/" + q.ID})
				break
			}
		}
	}
}
