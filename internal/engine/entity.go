// Package engine is a small reusable text-adventure framework built on
// two ideas:
//
//   - Composite: the world is one tree of entities. Rooms contain items,
//     containers contain items, the player is an entity whose contents
//     are the inventory.
//   - Composable: behavior is attached to entities as components. The
//     engine dispatches each parsed command to the target's components
//     in attachment order; the first to handle it wins, otherwise the
//     engine default for that verb runs.
//
// The engine knows nothing about any particular game. Content packages
// construct the entity tree declaratively.
package engine

import "strings"

// Entity is the one node type in the world tree. Anything the player
// can refer to — a room, an item, a container, an NPC, the player — is
// an entity distinguished only by its components and its place in the
// tree.
type Entity struct {
	ID      string
	Name    string
	Aliases []string

	Parent   *Entity
	Contents []*Entity

	Parts []Component
}

// NewEntity creates an entity. Extra names become aliases.
func NewEntity(id, name string, aliases ...string) *Entity {
	return &Entity{ID: id, Name: name, Aliases: aliases}
}

// With attaches components and returns the entity, for declarative
// construction. Attachment order is dispatch order.
func (e *Entity) With(parts ...Component) *Entity {
	e.Parts = append(e.Parts, parts...)
	return e
}

// Add places children inside this entity, reparenting as needed.
func (e *Entity) Add(children ...*Entity) *Entity {
	for _, c := range children {
		if c.Parent != nil {
			c.Parent.removeChild(c)
		}
		c.Parent = e
		e.Contents = append(e.Contents, c)
	}
	return e
}

func (e *Entity) removeChild(c *Entity) {
	out := e.Contents[:0]
	for _, v := range e.Contents {
		if v != c {
			out = append(out, v)
		}
	}
	e.Contents = out
}

// Matches reports whether name (already lowercased) refers to this
// entity.
// The display name matches with or without its leading article, so
// whatever the UI shows ("a data-shard") is always typable: the parser
// strips articles from input, this strips them from the name, and the
// two meet in the middle.
func (e *Entity) Matches(name string) bool {
	if name == e.ID || name == e.Name || name == stripArticle(e.Name) {
		return true
	}
	for _, a := range e.Aliases {
		if name == a {
			return true
		}
	}
	return false
}

func stripArticle(s string) string {
	for _, article := range []string{"a ", "an ", "the "} {
		if rest, ok := strings.CutPrefix(s, article); ok {
			return rest
		}
	}
	return s
}

// Walk visits the subtree rooted at e depth-first. Returning false from
// fn stops the walk.
func (e *Entity) Walk(fn func(*Entity) bool) bool {
	if !fn(e) {
		return false
	}
	for _, c := range e.Contents {
		if !c.Walk(fn) {
			return false
		}
	}
	return true
}
