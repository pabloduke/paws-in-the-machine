package engine

import (
	"fmt"
	"strings"
)

type Quest struct {
	ID, Name, Description, CompletionText string
	AutoStart                             bool
	Steps                                 []QuestStep
}
type QuestStep struct{ ID, Text, Kind, Target string }
type QuestStepState struct {
	ID, Text, Target, Reason string
	Complete, Satisfied      bool
}
type QuestState struct {
	ID, Name, Status string
	Steps            []QuestStepState
}

func questFlag(id, part string) string { return "quest:" + id + ":" + part }
func (w *World) questCondition(s QuestStep) (bool, string) {
	target := w.FindID(s.Target)
	if target == nil {
		return false, "Target is unavailable: " + s.Target
	}
	switch s.Kind {
	case "carry":
		if w.Carried(target) {
			return true, "Item is in inventory."
		}
		return false, "Item is not in inventory."
	case "visit":
		if w.Room() == target {
			return true, "Player is at the target."
		}
		return false, "Player is elsewhere."
	}
	return false, "Unsupported objective."
}

// QuestStates is read-only and used by both the game and editor preview.
func (w *World) QuestStates() []QuestState {
	out := []QuestState{}
	for _, q := range w.Quests {
		state := QuestState{ID: q.ID, Name: q.Name, Status: "available"}
		if w.Flags[questFlag(q.ID, "active")] {
			state.Status = "active"
		}
		if w.Flags[questFlag(q.ID, "complete")] {
			state.Status = "completed"
		}
		blocked := state.Status == "available"
		for _, s := range q.Steps {
			met, reason := w.questCondition(s)
			done := w.Flags[questFlag(q.ID, "step:"+s.ID)]
			if done {
				reason = "Completed; progress is retained. " + reason
			} else if blocked {
				reason = "Waiting for activation or an earlier step. " + reason
			}
			state.Steps = append(state.Steps, QuestStepState{s.ID, s.Text, s.Target, reason, done, met})
			if !done {
				blocked = true
			}
		}
		out = append(out, state)
	}
	return out
}
func (w *World) traceQuest(text string) {
	if !w.DevMode {
		return
	}
	w.QuestTrace = append(w.QuestTrace, text)
	if len(w.QuestTrace) > 100 {
		w.QuestTrace = w.QuestTrace[len(w.QuestTrace)-100:]
	}
}
func (w *World) EvaluateQuests() {
	for _, q := range w.Quests {
		if q.AutoStart {
			w.Flags[questFlag(q.ID, "active")] = true
		}
		if !w.Flags[questFlag(q.ID, "active")] || w.Flags[questFlag(q.ID, "complete")] {
			continue
		}
		done := len(q.Steps) > 0
		for _, s := range q.Steps {
			if w.Flags[questFlag(q.ID, "step:"+s.ID)] {
				continue
			}
			met, reason := w.questCondition(s)
			w.traceQuest(fmt.Sprintf("%s / %s: %s", q.Name, s.Text, reason))
			if !met {
				done = false
				break
			}
			w.Flags[questFlag(q.ID, "step:"+s.ID)] = true
			w.traceQuest("Step completed: " + s.ID)
		}
		if done {
			w.Flags[questFlag(q.ID, "complete")] = true
			text := q.CompletionText
			if text == "" {
				text = "Quest complete: " + q.Name
			}
			w.Pending = append(w.Pending, text)
			w.traceQuest(text)
		}
	}
}
func (w *World) QuestText(debug bool) string {
	var b strings.Builder
	for _, q := range w.QuestStates() {
		fmt.Fprintf(&b, "%s — %s", q.Name, q.Status)
		if debug {
			fmt.Fprintf(&b, " [%s]", q.ID)
		}
		b.WriteString("\n")
		for _, s := range q.Steps {
			mark := "[ ]"
			if s.Complete {
				mark = "[x]"
			}
			fmt.Fprintf(&b, "  %s %s\n", mark, s.Text)
			if debug {
				fmt.Fprintf(&b, "    %s: %s\n", s.Target, s.Reason)
			}
		}
	}
	if !debug {
		for _, q := range w.QuestStates() {
			if q.Status == "available" {
				fmt.Fprintf(&b, "Start %s: quests start %s\n", q.Name, q.ID)
			}
		}
	}
	if b.Len() == 0 {
		return "No authored quests."
	}
	return strings.TrimSpace(b.String())
}
func (w *World) DevLabel() string {
	label := ""
	if w.DevMode {
		label = "DEV"
	}
	if w.DevModified {
		if label != "" {
			label += " · "
		}
		label += "MODIFIED"
	}
	return label
}

// Developer commands are diagnostic session controls, not player actions.
func (w *World) DevCommand(line string) (string, bool) {
	args := strings.Fields(line)
	if len(args) >= 2 && args[0] == "sudo" && args[1] == "devmode" {
		if len(args) != 3 {
			return "Use sudo devmode --meow or sudo devmode --off.", true
		}
		switch args[2] {
		case "--meow":
			w.DevMode = true
			return "Developer mode enabled.", true
		case "--off":
			w.DevMode = false
			return "Developer mode disabled.", true
		}
		return "Unknown developer mode option.", true
	}
	if len(args) == 0 || args[0] != "quest-debug" {
		return "", false
	}
	if !w.DevMode {
		return "Developer mode is disabled.", true
	}
	if len(args) == 1 {
		return w.QuestText(true), true
	}
	if len(args) == 2 && args[1] == "trace" {
		return strings.Join(w.QuestTrace, "\n"), true
	}
	if len(args) == 3 && (args[1] == "reset" || args[1] == "complete") {
		for _, q := range w.Quests {
			if q.ID == args[2] {
				w.DevModified = true
				if args[1] == "reset" {
					for k := range w.Flags {
						if strings.HasPrefix(k, questFlag(q.ID, "")) {
							delete(w.Flags, k)
						}
					}
					w.traceQuest("Forced reset: " + q.ID)
					return "Quest reset. Inventory and position unchanged; satisfied objectives will complete at the next checkpoint.", true
				}
				w.Flags[questFlag(q.ID, "active")] = true
				for _, s := range q.Steps {
					w.Flags[questFlag(q.ID, "step:"+s.ID)] = true
				}
				w.Flags[questFlag(q.ID, "complete")] = true
				w.traceQuest("Forced completion: " + q.ID)
				return "Quest force-completed; session MODIFIED.", true
			}
		}
		return "Unknown quest ID.", true
	}
	return "quest-debug | quest-debug trace | quest-debug reset <id> | quest-debug complete <id>", true
}
