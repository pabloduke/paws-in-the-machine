package dialogue

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Talkable marks an entity as an NPC or terminal with a dialogue graph.
// It is a marker + data component: conversations are interactive, so
// they are driven through Start and the Session (the UI's talk
// intercept, or a headless caller using Start/Choose directly) — never
// through engine command dispatch, which is one-shot by design.
type Talkable struct {
	Start string
	Nodes map[string]Node
}

// Handle claims only "talk", and only to refuse honestly: the engine's
// default would say the NPC has nothing to say, which is a lie when a
// dialogue graph is attached. Conversations are interactive by ruling
// (the ME1/FNV modal), so the headless path points at the real driver
// instead of pretending to be one. Every other verb declines.
func (Talkable) Handle(_ *engine.World, self *engine.Entity, cmd engine.Command) (string, bool) {
	if cmd.Verb != "talk" {
		return "", false
	}
	return fmt.Sprintf("(%s has things to say — conversations are interactive; drive them with dialogue.Start, the way the UI's talk intercept does.)",
		engine.Capitalize(self.Name)), true
}

// Node is one line of NPC text plus the player choices available from it.
// TextFn, when set, overrides Text and derives the line from world state
// — the greeting that runs warm or cold on what an NPC remembers about
// Buddy (NPC memory, #14; docs/systems/dialogue.md). It mirrors
// engine.Description{Fn}: memory is ordinary flags, read at render time.
type Node struct {
	Text    string
	TextFn  func(*engine.World) string
	Choices []Choice
}

// line resolves a node's NPC text: TextFn wins when present, so a
// memory-aware greeting is a pure function of flags at render time.
func (n Node) line(w *engine.World) string {
	if n.TextFn != nil {
		return n.TextFn(w)
	}
	return n.Text
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

// Requirement controls whether a choice is visible. Stat requirements
// (StatCheck) are the exception: they never hide a choice — they render
// it with a derived tag and lock it until the stat qualifies.
type Requirement interface {
	Allowed(*engine.World) bool
}

// StatCheck is a visible, lockable stat gate: the choice renders with a
// "[Charm 8]"-style tag and stays locked until the stat meets Min.
// Locks re-evaluate every render, so training retroactively unlocks.
type StatCheck struct {
	Stat string
	Min  int
}

func (c StatCheck) Allowed(w *engine.World) bool {
	return statValue(w, c.Stat) >= c.Min
}

// Tag renders the FNV-style bracket tag, e.g. "[Charm 8]". A stat name
// the engine doesn't know renders loudly as a bug tag instead of
// silently locking the choice forever.
func (c StatCheck) Tag() string {
	if !engine.IsStat(c.Stat) {
		return fmt.Sprintf("[bug: unknown stat %q]", c.Stat)
	}
	return fmt.Sprintf("[%s %d]", engine.Capitalize(c.Stat), c.Min)
}

// Option is one rendered choice: the underlying Choice plus its derived
// stat tag and current lock state.
type Option struct {
	Choice Choice
	Tag    string // "[Charm 8]" etc.; empty when the choice has no stat gate
	Locked bool
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

// Speaker returns the NPC's display name for the active conversation.
func (s *Session) Speaker() string {
	if s.Done() {
		return ""
	}
	return engine.DisplayName(s.w, s.npc)
}

// Text returns the current node's NPC line, without choices.
func (s *Session) Text() string {
	if s.Done() {
		return ""
	}
	node, ok := s.graph.Nodes[s.nodeID]
	if !ok {
		return ""
	}
	return node.line(s.w)
}

// Options returns the currently visible choices with their lock state.
// Knowledge gates (flags, items, once-flags) hide a choice; stat gates
// keep it visible, tagged, and locked until the stat qualifies.
func (s *Session) Options() []Option {
	if s.Done() {
		return nil
	}
	node, ok := s.graph.Nodes[s.nodeID]
	if !ok {
		return nil
	}
	var out []Option
	for _, c := range node.Choices {
		if c.OnceFlag != "" && s.w.Flags[c.OnceFlag] {
			continue
		}
		opt := Option{Choice: c}
		hidden := false
		var tags []string
		for _, r := range c.Require {
			if check, ok := r.(StatCheck); ok {
				tags = append(tags, check.Tag())
				if !check.Allowed(s.w) {
					opt.Locked = true
				}
				continue
			}
			if !r.Allowed(s.w) {
				hidden = true
				break
			}
		}
		if hidden {
			continue
		}
		opt.Tag = strings.Join(tags, " ")
		out = append(out, opt)
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
	b.WriteString(node.line(s.w))
	options := s.Options()
	if len(options) == 0 {
		b.WriteString("\n\n[No responses available.]")
		return b.String()
	}
	for i, o := range options {
		b.WriteString(fmt.Sprintf("\n%d. ", i+1))
		if o.Tag != "" {
			b.WriteString(o.Tag + " ")
		}
		b.WriteString(o.Choice.Text)
		if o.Locked {
			b.WriteString(" ✗")
		}
	}
	return b.String()
}

// Pick selects a visible choice by 1-based index, applies its effects,
// and advances or ends the conversation. It returns only the effect
// prose (or a refusal for locked choices) — callers that render the
// next node live, like the UI panel, read Speaker/Text/Options after.
func (s *Session) Pick(n int) string {
	out := s.pick(n)
	// A pick completes a player action (docs/systems/events.md); its
	// effects may have set flags the event rules react to.
	s.w.CheckEvents()
	return out
}

func (s *Session) pick(n int) string {
	if s.Done() {
		return ""
	}
	options := s.Options()
	if n < 1 || n > len(options) {
		return "Choose one of the numbered responses."
	}
	if options[n-1].Locked {
		return lockedMessage(s.w, options[n-1].Choice)
	}
	choice := options[n-1].Choice
	if choice.OnceFlag != "" {
		s.w.Flags[choice.OnceFlag] = true
	}

	var lines []string
	for _, effect := range choice.Effects {
		if out := effect.Apply(s.w); out != "" {
			lines = append(lines, out)
		}
	}
	if choice.End || choice.Next == "" {
		s.done = true
		return strings.Join(lines, "\n\n")
	}
	if _, ok := s.graph.Nodes[choice.Next]; !ok {
		s.done = true
		lines = append(lines, "(bug) missing dialogue node "+strconv.Quote(choice.Next))
		return strings.Join(lines, "\n\n")
	}
	s.nodeID = choice.Next
	return strings.Join(lines, "\n\n")
}

// Choose is Pick plus a render of the next node — the one-call flow for
// headless callers and tests.
func (s *Session) Choose(n int) string {
	out := s.Pick(n)
	if s.Done() {
		return out
	}
	if rendered := s.Render(); rendered != "" {
		if out != "" {
			return out + "\n\n" + rendered
		}
		return rendered
	}
	return out
}

// Flag requires a story flag to be set.
func Flag(name string) Requirement {
	return RequirementFunc(func(w *engine.World) bool { return w.Flags[name] })
}

// MissingFlag requires a story flag to be absent.
func MissingFlag(name string) Requirement {
	return RequirementFunc(func(w *engine.World) bool { return !w.Flags[name] })
}

// HasItem requires Buddy to carry an entity by ID. It scans only the
// inventory subtree — carried is a fact about the player's contents,
// so the rest of the world never needs walking.
func HasItem(id string) Requirement {
	return RequirementFunc(func(w *engine.World) bool {
		found := false
		w.Player.Walk(func(e *engine.Entity) bool {
			if e != w.Player && e.ID == id {
				found = true
				return false
			}
			return true
		})
		return found
	})
}

// StatAtLeast gates a choice on one of Buddy's visible stats. Unlike the
// other requirements it does not hide the choice: it renders tagged
// ("[Charm 8]") and locked until the stat meets the threshold.
func StatAtLeast(name string, value int) Requirement {
	return StatCheck{Stat: strings.ToLower(name), Min: value}
}

// statValue reads a stat through the engine's single name mapping;
// unknown names read as -1, which Tag surfaces as a bug tag.
func statValue(w *engine.World, name string) int {
	if p := w.Stats.ByName(name); p != nil {
		return *p
	}
	return -1
}

// lockedMessage explains a refused locked choice, e.g.
// "Your Charm isn't up to that yet. (Charm 5/8)".
func lockedMessage(w *engine.World, c Choice) string {
	for _, r := range c.Require {
		check, ok := r.(StatCheck)
		if !ok || check.Allowed(w) {
			continue
		}
		name := engine.Capitalize(check.Stat)
		return fmt.Sprintf("Your %s isn't up to that yet. (%s %d/%d)",
			name, name, statValue(w, check.Stat), check.Min)
	}
	return "You can't do that yet."
}

// SetFlag sets a story flag.
func SetFlag(name string) Effect {
	return EffectFunc(func(w *engine.World) string {
		w.Flags[name] = true
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
