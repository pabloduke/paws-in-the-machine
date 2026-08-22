package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/charts"
)

// entityRecord is the editor's deliberately small view of assembled game
// content. The editor needs stable identity for references; it does not need
// ownership of, or write access to, runtime entities.
type entityRecord struct {
	ID       string
	Name     string
	ParentID string
}

// entityCatalog is injected into the model so editor tests and future content
// sources do not have to construct the complete game world.
type entityCatalog interface {
	Lookup(id string) (entityRecord, bool)
}

type catalog map[string]entityRecord

func (c catalog) Lookup(id string) (entityRecord, bool) {
	record, ok := c[id]
	return record, ok
}

// assembledGameCatalog takes a read-only snapshot of entity identity from the
// same assembled world the game runs. If content declares an ID twice, there
// is no stable reference target, so the editor refuses to start.
func assembledGameCatalog() (catalog, error) {
	return catalogWorld(game.NewWorld())
}

func catalogWorld(w *engine.World) (catalog, error) {
	out := catalog{}
	var duplicate string
	w.Root.Walk(func(e *engine.Entity) bool {
		if _, exists := out[e.ID]; exists {
			duplicate = e.ID
			return false
		}
		parentID := ""
		if e.Parent != nil {
			parentID = e.Parent.ID
		}
		out[e.ID] = entityRecord{ID: e.ID, Name: e.Name, ParentID: parentID}
		return true
	})
	if duplicate != "" {
		return nil, fmt.Errorf("assembled game declares duplicate entity ID %q", duplicate)
	}
	return out, nil
}

type placement struct {
	chart string
	at    charts.Coord
	id    string
}

func weavePlacements(w *charts.Weave) []placement {
	var out []placement
	for _, chart := range w.Charts() {
		for at, id := range chart.Cells {
			out = append(out, placement{chart: chart.ID, at: at, id: id})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].id != out[j].id {
			return out[i].id < out[j].id
		}
		if out[i].chart != out[j].chart {
			return out[i].chart < out[j].chart
		}
		a, b := out[i].at, out[j].at
		if a.X != b.X {
			return a.X < b.X
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		return a.W < b.W
	})
	return out
}

func placementText(p placement) string {
	return p.chart + " " + coordText(p.at)
}

// validateWeave checks the reference invariants the editor can prove from the
// assembled game: every placed ID exists, and one entity occupies at most one
// chart coordinate. Both are hard errors because charts.Apply targets runtime
// entities by ID.
func validateWeave(w *charts.Weave, entities entityCatalog) []string {
	placements := weavePlacements(w)
	byID := make(map[string][]placement)
	var problems []string
	for _, p := range placements {
		byID[p.id] = append(byID[p.id], p)
		if _, ok := entities.Lookup(p.id); !ok {
			problems = append(problems, fmt.Sprintf(
				"unknown entity %q at %s", p.id, placementText(p)))
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ps := byID[id]
		if len(ps) < 2 {
			continue
		}
		locations := make([]string, len(ps))
		for i, p := range ps {
			locations[i] = placementText(p)
		}
		problems = append(problems, fmt.Sprintf(
			"entity %q is placed more than once: %s", id,
			strings.Join(locations, ", ")))
	}
	return problems
}

func validatePlacement(w *charts.Weave, entities entityCatalog, chartID string,
	at charts.Coord, id string) string {
	if _, ok := entities.Lookup(id); !ok {
		return fmt.Sprintf("unknown entity %q", id)
	}
	for _, p := range weavePlacements(w) {
		if p.id == id && (p.chart != chartID || p.at != at) {
			return fmt.Sprintf("entity %q is already placed at %s", id,
				placementText(p))
		}
	}
	return ""
}
