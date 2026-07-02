package hubs_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

func cityWorld() *engine.World {
	w := engine.NewWorld()

	home := engine.NewEntity("home", "Home Hub").With(hubs.Hub{Entry: "den"})
	den := engine.NewEntity("den", "The Den").With(engine.Exits{})
	cellar := engine.NewEntity("cellar", "The Cellar").With(engine.Exits{})

	plaza := engine.NewEntity("plaza", "Plaza Hub").With(hubs.Hub{Entry: "square"})
	square := engine.NewEntity("square", "The Square").With(engine.Exits{})

	// A non-hub child of root must not appear in List.
	limbo := engine.NewEntity("limbo", "Limbo")

	w.Root.Add(home, plaza, limbo)
	home.Add(den, cellar)
	plaza.Add(square)
	cellar.Add(w.Player) // nested: Current must climb to the hub
	return w
}

func TestListAndCurrent(t *testing.T) {
	w := cityWorld()

	list := hubs.List(w)
	if len(list) != 2 || list[0].ID != "home" || list[1].ID != "plaza" {
		t.Fatalf("unexpected hub list: %v", list)
	}
	if cur := hubs.Current(w); cur == nil || cur.ID != "home" {
		t.Fatalf("expected current hub home, got %v", cur)
	}

	// Worlds without hubs (engine test fixtures) degrade gracefully.
	bare := engine.NewWorld()
	room := engine.NewEntity("room", "Room")
	bare.Root.Add(room)
	room.Add(bare.Player)
	if got := hubs.List(bare); len(got) != 0 {
		t.Fatalf("expected no hubs, got %v", got)
	}
	if got := hubs.Current(bare); got != nil {
		t.Fatalf("expected nil current hub, got %v", got)
	}
}

func TestTravel(t *testing.T) {
	w := cityWorld()

	out := hubs.Travel(w, "plaza")
	if !strings.Contains(out, "The Square") {
		t.Fatalf("travel should land in and describe the square: %q", out)
	}
	if w.Room().ID != "square" {
		t.Fatalf("player should be in square, is in %s", w.Room().ID)
	}
	if cur := hubs.Current(w); cur.ID != "plaza" {
		t.Fatalf("current hub should be plaza, got %s", cur.ID)
	}

	// Traveling home lands at the entry room, not where we left from.
	hubs.Travel(w, "home")
	if w.Room().ID != "den" {
		t.Fatalf("travel home should land in den, is in %s", w.Room().ID)
	}

	if out := hubs.Travel(w, "nowhere"); !strings.Contains(out, "unknown hub") {
		t.Fatalf("unknown hub should report a content bug: %q", out)
	}
}
