package content

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Quest definitions are immutable authoring data; progress lives in game flags.
type Quest struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Enabled        bool        `json:"enabled"`
	AutoStart      bool        `json:"auto_start"`
	CompletionText string      `json:"completion_text"`
	Steps          []QuestStep `json:"steps"`
}
type QuestStep struct {
	ID     string  `json:"id"`
	Text   string  `json:"text"`
	Kind   string  `json:"kind"`
	Target CellRef `json:"target"`
}
type QuestsFile struct {
	Version int     `json:"version"`
	Quests  []Quest `json:"quests"`
}

func EmptyQuests() QuestsFile { return QuestsFile{Version: 1, Quests: []Quest{}} }
func DecodeQuests(b []byte) (QuestsFile, error) {
	f := EmptyQuests()
	if err := json.Unmarshal(b, &f); err != nil {
		return f, err
	}
	return f, f.Validate()
}
func (f QuestsFile) Validate() error {
	if f.Version != 1 {
		return fmt.Errorf("unsupported quests version %d", f.Version)
	}
	validID := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	ids := map[string]bool{}
	for _, q := range f.Quests {
		if !validID.MatchString(q.ID) || ids[q.ID] {
			return fmt.Errorf("invalid or duplicate quest ID %q", q.ID)
		}
		ids[q.ID] = true
		steps := map[string]bool{}
		for _, s := range q.Steps {
			if !validID.MatchString(s.ID) || steps[s.ID] {
				return fmt.Errorf("invalid or duplicate step ID %q", s.ID)
			}
			steps[s.ID] = true
		}
	}
	return nil
}

// QuestProblems includes incomplete drafts for editor diagnostics. Only enabled
// quests block assembly; drafts remain editable even with unresolved references.
func (c Catalogs) QuestProblems(q Quest) []string {
	var out []string
	if strings.TrimSpace(q.Name) == "" {
		out = append(out, "Name is required.")
	}
	if len(q.Steps) == 0 {
		out = append(out, "At least one objective is required.")
	}
	for _, s := range q.Steps {
		issue := ""
		if strings.TrimSpace(s.Text) == "" {
			out = append(out, fmt.Sprintf("Step %s: objective text is required.", s.ID))
		}
		switch s.Kind {
		case "carry":
			found := false
			for _, i := range c.WorldItems.WorldItems {
				if i.ID == s.Target.ID && s.Target.Kind == "world_item" {
					found = true
					if string(i.Kind) != "takeable" {
						issue = "item must be takeable"
					}
				}
			}
			if !found {
				issue = "choose an existing item"
			}
		case "visit":
			found := false
			for _, i := range c.Locations.Locations {
				if s.Target.Kind == "location" && i.ID == s.Target.ID {
					found = true
				}
			}
			for _, i := range c.Rooms.Rooms {
				if s.Target.Kind == "room" && i.ID == s.Target.ID {
					found = true
				}
			}
			if !found {
				issue = "choose an existing Location or Room"
			}
		default:
			issue = "choose carry or visit"
		}
		if issue != "" {
			out = append(out, fmt.Sprintf("Step %s: %s.", s.ID, issue))
		}
	}
	return out
}
