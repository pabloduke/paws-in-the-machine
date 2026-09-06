package main

import (
	"net/http"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const overviewBasePath = "/overview"

type treeThing struct {
	Name, Kind, KindLabel string
}

type treeRoom struct {
	ID, Name, Status string
	IsEntry          bool
	Contents         []treeThing
}

type treeLocation struct {
	ID, Name, Status string
	HasEntry         bool
	Contents         []treeThing
	Rooms            []treeRoom
}

type treeHub struct {
	ID, Name  string
	Locations []treeLocation
}

type looseGroup struct {
	Label, Path string
	Names       []string
}

type overviewPage struct {
	Hubs       []treeHub
	Unrooted   []treeLocation
	HubCount   int
	Problems   []string
	Advisories []looseGroup
	Loose      []looseGroup
	Empty      bool
}

func (h *editorHandler) serveOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	data := basePage("overview", "overview", "World Overview", "overview")
	data.Play = h.playPage("")
	page := overviewPage{}
	snapshot, err := h.worldSnapshot()
	if err != nil {
		data.StoreError = err.Error()
		data.Overview = page
		h.renderPage(w, r, data)
		return
	}

	page.HubCount = len(snapshot.Hubs)
	page.Empty = len(snapshot.Hubs) == 0 &&
		len(snapshot.Unrooted) == 0 && len(snapshot.OrphanRooms) == 0
	for _, hub := range snapshot.Hubs {
		node := treeHub{ID: hub.ID, Name: hub.Name}
		for _, location := range hub.Locations {
			node.Locations = append(node.Locations, treeLocationNode(location))
		}
		page.Hubs = append(page.Hubs, node)
	}
	// Locations with no Hub are drawn as their own subtree rather than
	// flattened to a name, so their Rooms and contents stay visible.
	for _, location := range snapshot.Unrooted {
		page.Unrooted = append(page.Unrooted, treeLocationNode(location))
	}

	page.Problems, err = h.allProblems()
	if err != nil {
		data.StoreError = err.Error()
		data.Overview = page
		h.renderPage(w, r, data)
		return
	}

	// Advisories are not errors. An unassigned or unplaced entity is a
	// valid resting state (docs/EDITOR.md); these lists exist so a
	// designer can see what is still waiting for them.
	page.Advisories = advisoryGroups(snapshot)
	for _, kind := range []string{gamecontent.ContentKindWorldItem, gamecontent.ContentKindNPC, gamecontent.ContentKindTerminal} {
		loose := snapshot.Loose[kind]
		if len(loose) == 0 {
			continue
		}
		group := looseGroup{Label: contentKindLabel(kind) + "s in no cell", Path: placePathForKind(kind)}
		for _, thing := range loose {
			group.Names = append(group.Names, thing.Name)
		}
		page.Loose = append(page.Loose, group)
	}

	data.Overview = page
	h.renderPage(w, r, data)
}

func treeLocationNode(location worldLocationNode) treeLocation {
	node := treeLocation{
		ID: location.ID, Name: location.Name,
		Status:   coordinateLabel(location.Placed, location.X, location.Y),
		HasEntry: location.EntryRoomID != "",
		Contents: treeThings(location.Contents),
	}
	for _, room := range location.Rooms {
		node.Rooms = append(node.Rooms, treeRoom{
			ID: room.ID, Name: room.Name,
			Status:   coordinateLabel(room.Placed, room.X, room.Y),
			IsEntry:  room.IsEntry,
			Contents: treeThings(room.Contents),
		})
	}
	return node
}

func treeThings(contents []worldThing) []treeThing {
	out := make([]treeThing, 0, len(contents))
	for _, thing := range contents {
		out = append(out, treeThing{Name: thing.Name, Kind: thing.Kind, KindLabel: contentKindLabel(thing.Kind)})
	}
	return out
}

func placePathForKind(kind string) string {
	switch kind {
	case gamecontent.ContentKindWorldItem:
		return worldItemContentsBasePath
	case gamecontent.ContentKindNPC:
		return npcContentsBasePath
	case gamecontent.ContentKindTerminal:
		return terminalContentsBasePath
	}
	return overviewBasePath
}

func advisoryGroups(snapshot worldSnapshot) []looseGroup {
	var groups []looseGroup
	if len(snapshot.Unrooted) > 0 {
		group := looseGroup{Label: "Locations in no Hub", Path: "/content/hubs"}
		for _, location := range snapshot.Unrooted {
			group.Names = append(group.Names, location.Name)
		}
		groups = append(groups, group)
	}
	if len(snapshot.OrphanRooms) > 0 {
		group := looseGroup{Label: "Rooms in no Location", Path: "/content/locations"}
		for _, thing := range snapshot.OrphanRooms {
			group.Names = append(group.Names, thing.Name)
		}
		groups = append(groups, group)
	}
	unplacedLocations := looseGroup{Label: "Locations assigned but not placed on a grid", Path: locationPlacementBasePath}
	unplacedRooms := looseGroup{Label: "Rooms assigned but not placed on a grid", Path: roomPlacementBasePath}
	noEntry := looseGroup{Label: "Locations with Rooms but no entry Room", Path: "/content/locations"}
	snapshot.AllLocations(func(hubName string, location worldLocationNode) {
		// A Location with no Hub is already reported as unrooted; it is
		// not additionally "assigned but unplaced".
		if hubName != "" && !location.Placed {
			unplacedLocations.Names = append(unplacedLocations.Names, location.Name)
		}
		if len(location.Rooms) > 0 && location.EntryRoomID == "" {
			noEntry.Names = append(noEntry.Names, location.Name)
		}
		for _, room := range location.Rooms {
			if !room.Placed {
				unplacedRooms.Names = append(unplacedRooms.Names, room.Name)
			}
		}
	})
	for _, group := range []looseGroup{unplacedLocations, unplacedRooms, noEntry} {
		if len(group.Names) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

// allProblems runs every relationship check and returns each problem it
// finds. The write path uses the fail-fast validators instead.
func (h *editorHandler) allProblems() ([]string, error) {
	problems, err := h.spatialProblems()
	if err != nil {
		return nil, err
	}
	contentProblems, err := h.contentsProblems()
	if err != nil {
		return nil, err
	}
	problems = append(problems, contentProblems...)
	networkProblems, err := h.networkProblems()
	if err != nil {
		return nil, err
	}
	problems = append(problems, networkProblems...)
	authProblems, err := h.authProblems()
	if err != nil {
		return nil, err
	}
	return append(problems, authProblems...), nil
}
