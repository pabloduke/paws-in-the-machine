package dialogue

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Talkable marks an entity as an NPC or terminal with a dialogue graph.
type Talkable struct {
	Start string
	Nodes map[string]Node
}

func (t Talkable) Handle(w *engine.World, self *engine.Entity, cmd engine.Command) (string, bool) {
	if cmd.Verb != "talk" {
		return "", false
	}
	s, err := Start(w, self)
	if err != nil {
		return err.Error(), true
	}
	return s.Render(), true
}

// Node is one line of NPC text plus the player choices available from it.
type Node struct {
	Text    string
	Choices []Choice
}

// Choice is a player response. Requirements hide unavailable choices;
// Effects apply before moving to Next or ending the conversation.
type Choice struct {
	Text     string
	Require  []Requirement
	Effects  []Effect
	Next     string
	End      bool
	OnceFlag string
}

// Requirement controls whether a choice is visible.
type Requirement interface {
	Allowed(*engine.World) bool
}

// RequirementFunc adapts a function to Requirement.
type RequirementFunc func(*engine.World) bool

func (f RequirementFunc) Allowed(w *engine.World) bool { return f(w) }

// Effect mutates world state when a choice is selected. Returning text
// appends an extra line before the next node or conversation end.
type Effect interface {
	Apply(*engine.World) string
}

// EffectFunc adapts a function to Effect.
type EffectFunc func(*engine.World) string

func (f EffectFunc) Apply(w *engine.World) string { return f(w) }

// Session is the active conversation state.
type Session struct {
	w      *engine.World
	npc    *engine.Entity
	graph  Talkable
	nodeID string
	done   bool
}

// Start opens a dialogue session for a Talkable entity.
func Start(w *engine.World, npc *engine.Entity) (*Session, error) {
	graph, ok := engine.Part[Talkable](npc)
	if !ok {
		return nil, fmt.Errorf("%s has nothing to say.", engine.DisplayName(w, npc))
	}
	start := graph.Start
	if start == "" {
		start = "start"
	}
	if _, ok := graph.Nodes[start]; !ok {
		return nil, fmt.Errorf("(bug) dialogue %s has no start node %q", npc.ID, start)
	}
	return &Session{w: w, npc: npc, graph: graph, nodeID: start}, nil
}

// Done reports whether the conversation has ended.
func (s *Session) Done() bool { return s == nil || s.done }

// Choices returns the currently visible choices.
func (s *Session) Choices() []Choice {
	if s.Done() {
		return nil
	}
	node, ok := s.graph.Nodes[s.nodeID]
	if !ok {
		return nil
	}
	var out []Choice
	for _, c := range node.Choices {
		if c.OnceFlag != "" && s.w.Flags[c.OnceFlag] {
			continue
		}
		if requirementsAllow(s.w, c.Require) {
			out = append(out, c)
		}
	}
	return out
}

// Render prints the current NPC line and numbered visible choices.
func (s *Session) Render() string {
	if s.Done() {
		return ""
	}
	node, ok := s.graph.Nodes[s.nodeID]
	if !ok {
		s.done = true
		return "(bug) missing dialogue node " + strconv.Quote(s.nodeID)
	}
	var b strings.Builder
	b.WriteString(engine.DisplayName(s.w, s.npc) + ":\n")
	b.WriteString(node.Text)
	choices := s.Choices()
	if len(choices) == 0 {
		b.WriteString("\n\n[No responses available.]")
		return b.String()
	}
	for i, c := range choices {
		b.WriteString(fmt.Sprintf("\n%d. %s", i+1, c.Text))
	}
	return b.String()
}

// Choose selects a visible choice by 1-based index and returns the text
// produced by effects plus the next rendered node, if any.
func (s *Session) Choose(n int) string {
	if s.Done() {
		return ""
	}
	choices := s.Choices()
	if n < 1 || n > len(choices) {
		return "Choose one of the numbered responses."
	}
	choice := choices[n-1]
	if choice.OnceFlag != "" {
		s.w.Flags[choice.OnceFlag] = true
	}

	var lines []string
	for _, effect := range choice.Effects {
		if out := effect.Apply(s.w); out != "" {
			lines = append(lines, out)
		}
	}
	if choice.End {
		s.done = true
		return strings.Join(lines, "\n\n")
	}
	if choice.Next == "" {
		s.done = true
		return strings.Join(lines, "\n\n")
	}
	if _, ok := s.graph.Nodes[choice.Next]; !ok {
		s.done = true
		lines = append(lines, "(bug) missing dialogue node "+strconv.Quote(choice.Next))
		return strings.Join(lines, "\n\n")
	}
	s.nodeID = choice.Next
	if rendered := s.Render(); rendered != "" {
		lines = append(lines, rendered)
	}
	return strings.Join(lines, "\n\n")
}

func requirementsAllow(w *engine.World, reqs []Requirement) bool {
	for _, r := range reqs {
		if !r.Allowed(w) {
			return false
		}
	}
	return true
}

// Flag requires a story flag to be set.
func Flag(name string) Requirement {
	return RequirementFunc(func(w *engine.World) bool { return w.Flags[name] })
}

// MissingFlag requires a story flag to be absent.
func MissingFlag(name string) Requirement {
	return RequirementFunc(func(w *engine.World) bool { return !w.Flags[name] })
}

// HasItem requires Buddy to carry an entity by ID.
func HasItem(id string) Requirement {
	return RequirementFunc(func(w *engine.World) bool {
		e := w.FindID(id)
		return e != nil && w.Carried(e)
	})
}

// StatAtLeast requires one of Buddy's visible stats to meet a threshold.
func StatAtLeast(name string, value int) Requirement {
	return RequirementFunc(func(w *engine.World) bool {
		switch strings.ToLower(name) {
		case "stealth":
			return w.Stats.Stealth >= value
		case "agility":
			return w.Stats.Agility >= value
		case "charm":
			return w.Stats.Charm >= value
		default:
			return false
		}
	})
}

// SetFlag sets a story flag.
func SetFlag(name string) Effect {
	return EffectFunc(func(w *engine.World) string {
		w.Flags[name] = true
		return ""
	})
}

// ClearFlag clears a story flag.
func ClearFlag(name string) Effect {
	return EffectFunc(func(w *engine.World) string {
		delete(w.Flags, name)
		return ""
	})
}

// Say emits extra prose for a selected choice.
func Say(text string) Effect {
	return EffectFunc(func(*engine.World) string { return text })
}

// AwardOnce grants XP once, guarded by a flag.
func AwardOnce(flag string, xp int) Effect {
	return EffectFunc(func(w *engine.World) string {
		if w.Flags[flag] {
			return ""
		}
		w.Flags[flag] = true
		return engine.AwardXP(w, xp)
	})
}
