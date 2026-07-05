package engine

import (
	"strings"
	"testing"
)

// placedWorld: home and away rooms, an NPC whose position is a
// function of the "moved" flag, player in home.
func placedWorld() (*World, *Entity) {
	w := NewWorld()
	home := NewEntity("home", "Home")
	away := NewEntity("away", "Away")
	npc := NewEntity("npc", "the npc").With(Placed{Fn: func(w *World) string {
		if w.Flags["moved"] {
			return "away"
		}
		return "home"
	}})
	w.Root.Add(home, away)
	home.Add(w.Player, npc)
	return w, npc
}

func TestPlacementFollowsState(t *testing.T) {
	w, npc := placedWorld()
	w.CheckEvents()
	if npc.Parent.ID != "home" {
		t.Fatalf("npc should stay put while flag is false: %s", npc.Parent.ID)
	}
	w.Flags["moved"] = true
	w.CheckEvents()
	if npc.Parent.ID != "away" {
		t.Fatalf("npc should follow its function: %s", npc.Parent.ID)
	}
	w.Flags["moved"] = false
	w.CheckEvents()
	if npc.Parent.ID != "home" {
		t.Fatalf("position is a pure function of state — should return: %s", npc.Parent.ID)
	}
}

func TestPlacementOffstage(t *testing.T) {
	w := NewWorld()
	home := NewEntity("home", "Home")
	ghost := NewEntity("ghost", "a ghost").With(Placed{Fn: func(w *World) string {
		if w.Flags["haunting"] {
			return "home"
		}
		return ""
	}})
	w.Root.Add(home)
	home.Add(w.Player, ghost)
	w.CheckEvents()
	if ghost.Parent != w.Offstage {
		t.Fatalf("empty placement should go offstage: %v", ghost.Parent.ID)
	}
	if w.InScope("ghost") != nil {
		t.Fatalf("offstage entities must be out of scope")
	}
	w.Flags["haunting"] = true
	w.CheckEvents()
	if ghost.Parent.ID != "home" {
		t.Fatalf("offstage entities must be able to return: %s", ghost.Parent.ID)
	}
}

func TestHoldPlacementsDefers(t *testing.T) {
	w, npc := placedWorld()
	w.HoldPlacements = true
	w.Flags["moved"] = true
	w.CheckEvents()
	if npc.Parent.ID != "home" {
		t.Fatalf("held placements must not apply: %s", npc.Parent.ID)
	}
	w.HoldPlacements = false
	w.CheckEvents()
	if npc.Parent.ID != "away" {
		t.Fatalf("released placements should catch up: %s", npc.Parent.ID)
	}
}

func TestPlacementBadRoomReports(t *testing.T) {
	w := NewWorld()
	home := NewEntity("home", "Home")
	lost := NewEntity("lost", "lost").With(Placed{Fn: func(*World) string {
		return "nowhere"
	}})
	w.Root.Add(home)
	home.Add(w.Player, lost)
	w.CheckEvents()
	if len(w.Pending) != 1 || !strings.Contains(w.Pending[0], "(bug)") {
		t.Fatalf("broken placement should surface as a content bug: %v", w.Pending)
	}
}

func TestRulesSeePostPlacementBoard(t *testing.T) {
	w, npc := placedWorld()
	w.Rules = []When{{
		Flags: []string{"moved"},
		Once:  "noticed",
		Do: func(w *World) string {
			// The rule runs after placement: the npc is already away.
			if npc.Parent.ID != "away" {
				return "stale board"
			}
			return "fresh board"
		},
	}}
	w.Flags["moved"] = true
	w.CheckEvents()
	if len(w.Pending) != 1 || w.Pending[0] != "fresh board" {
		t.Fatalf("rules must read the post-placement board: %v", w.Pending)
	}
}
