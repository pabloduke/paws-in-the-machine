package main

import (
	"fmt"
	"sort"
	"strings"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

// One assembled read-only view of every authored spatial relationship.
// Both the Overview tree and the cell-contents screens need the same
// joins — Hub owns Location owns Room, plus what sits in each cell — so
// they are computed once here instead of twice in the handlers.

type worldThing struct {
	ID, Kind, Name string
}

type worldRoomNode struct {
	ID, Name    string
	Placed      bool
	X, Y        int
	IsEntry     bool
	Contents    []worldThing
	Description string
}

type worldLocationNode struct {
	ID, Name    string
	Placed      bool
	X, Y        int
	Rooms       []worldRoomNode
	Contents    []worldThing
	EntryRoomID string
	Description string
}

type worldHubNode struct {
	ID, Name  string
	Locations []worldLocationNode
}

// container is one player-standable cell that can hold things.
type container struct {
	Kind string // gamecontent.ContainerKind*
	ID   string
	Name string // the cell's own name
	Path string // human-readable ancestry, e.g. "Plaza › Arcade › Back Room"
}

func (c container) Value() string { return c.Kind + ":" + c.ID }

type worldSnapshot struct {
	Hubs []worldHubNode
	// Locations with no Hub. They keep their own Rooms: an unassigned
	// Location is a valid resting state, and its interior must stay
	// visible and authorable while the building has nowhere to stand.
	Unrooted []worldLocationNode
	// Rooms with no Location at all.
	OrphanRooms []worldThing
	Containers  []container
	// Entities that exist but are not inside any cell, by kind.
	Loose map[string][]worldThing
	// Contents of every cell, keyed by "parentKind:parentID". Callers
	// read this directly rather than walking the tree, so a cell is
	// never unreachable because of where it sits.
	ContentsOf map[string][]worldThing
	// Every thing of each kind that is in some cell, by kind.
	Placed map[string][]worldThing
	// Placement of each thing, keyed by "kind:id", for status labels.
	Placement map[string]string
}

// AllLocations walks every Location node in the snapshot, rooted or not.
func (s worldSnapshot) AllLocations(visit func(hubName string, location worldLocationNode)) {
	for _, hub := range s.Hubs {
		for _, location := range hub.Locations {
			visit(hub.Name, location)
		}
	}
	for _, location := range s.Unrooted {
		visit("", location)
	}
}

func parseContainerValue(value string) (kind, id string, ok bool) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", "", false
	}
	switch parts[0] {
	case gamecontent.ContainerKindLocation, gamecontent.ContainerKindRoom:
		return parts[0], parts[1], true
	}
	return "", "", false
}

func byNameLower(name func(int) string, n int) func(i, j int) bool {
	return func(i, j int) bool { return strings.ToLower(name(i)) < strings.ToLower(name(j)) }
}

// worldSnapshot joins every catalog and relation into one tree. A store
// error is returned rather than partially-built state, so callers show
// one honest failure instead of a half-empty world.
func (h *editorHandler) worldSnapshot() (worldSnapshot, error) {
	snapshot := worldSnapshot{
		Loose: map[string][]worldThing{}, Placement: map[string]string{},
		ContentsOf: map[string][]worldThing{}, Placed: map[string][]worldThing{},
	}

	hubs, err := h.hubs.List()
	if err != nil {
		return snapshot, err
	}
	locations, err := h.locations.List()
	if err != nil {
		return snapshot, err
	}
	rooms, err := h.rooms.List()
	if err != nil {
		return snapshot, err
	}
	items, err := h.items.List()
	if err != nil {
		return snapshot, err
	}
	npcs, err := h.npcs.List()
	if err != nil {
		return snapshot, err
	}
	terminals, err := h.terminals.List()
	if err != nil {
		return snapshot, err
	}
	locationAssignments, err := h.locationAssignments.List()
	if err != nil {
		return snapshot, err
	}
	roomAssignments, err := h.roomAssignments.List()
	if err != nil {
		return snapshot, err
	}
	locationPlacements, err := h.locationPlacements.List()
	if err != nil {
		return snapshot, err
	}
	roomPlacements, err := h.roomPlacements.List()
	if err != nil {
		return snapshot, err
	}
	entries, err := h.locationEntries.List()
	if err != nil {
		return snapshot, err
	}
	contents, err := h.contents.List()
	if err != nil {
		return snapshot, err
	}

	hubOfLocation := map[string]string{}
	for _, a := range locationAssignments {
		hubOfLocation[a.LocationID] = a.HubID
	}
	locationOfRoom := map[string]string{}
	for _, a := range roomAssignments {
		locationOfRoom[a.RoomID] = a.LocationID
	}
	locationCoords := map[string]gamecontent.LocationPlacement{}
	for _, p := range locationPlacements {
		locationCoords[p.LocationID] = p
	}
	roomCoords := map[string]gamecontent.RoomPlacement{}
	for _, p := range roomPlacements {
		roomCoords[p.RoomID] = p
	}
	entryOfLocation := map[string]string{}
	for _, e := range entries {
		entryOfLocation[e.LocationID] = e.RoomID
	}

	// Everything that can sit in a cell, in one list keyed by kind.
	things := map[string]map[string]worldThing{
		gamecontent.ContentKindWorldItem: {},
		gamecontent.ContentKindNPC:       {},
		gamecontent.ContentKindTerminal:  {},
	}
	for _, v := range items {
		things[gamecontent.ContentKindWorldItem][v.ID] = worldThing{ID: v.ID, Kind: gamecontent.ContentKindWorldItem, Name: v.Name}
	}
	for _, v := range npcs {
		things[gamecontent.ContentKindNPC][v.ID] = worldThing{ID: v.ID, Kind: gamecontent.ContentKindNPC, Name: v.Name}
	}
	for _, v := range terminals {
		things[gamecontent.ContentKindTerminal][v.ID] = worldThing{ID: v.ID, Kind: gamecontent.ContentKindTerminal, Name: v.HostName}
	}

	contentsOfCell := map[string][]worldThing{}
	placedThings := map[string]bool{}
	for _, record := range contents {
		thing, ok := things[record.EntityKind][record.EntityID]
		if !ok {
			// A dangling reference; the validation report names it.
			continue
		}
		cell := record.ParentKind + ":" + record.ParentID
		contentsOfCell[cell] = append(contentsOfCell[cell], thing)
		placedThings[contentKey(record.EntityKind, record.EntityID)] = true
	}
	for _, list := range contentsOfCell {
		sort.SliceStable(list, byNameLower(func(i int) string { return list[i].Name }, len(list)))
	}

	roomsOfLocation := map[string][]gamecontent.Room{}
	for _, room := range rooms {
		if parent, ok := locationOfRoom[room.ID]; ok {
			roomsOfLocation[parent] = append(roomsOfLocation[parent], room)
		} else {
			snapshot.OrphanRooms = append(snapshot.OrphanRooms, worldThing{ID: room.ID, Kind: "room", Name: room.Name})
		}
	}

	buildRoom := func(room gamecontent.Room, entryRoomID string) worldRoomNode {
		node := worldRoomNode{ID: room.ID, Name: room.Name, Description: room.Description, IsEntry: room.ID == entryRoomID}
		if coords, ok := roomCoords[room.ID]; ok {
			node.Placed, node.X, node.Y = true, coords.X, coords.Y
		}
		node.Contents = contentsOfCell[gamecontent.ContainerKindRoom+":"+room.ID]
		return node
	}

	// Every Location becomes a node with its Rooms attached, whether or
	// not it has a Hub. Ancestry decides where the node is filed, never
	// whether it exists.
	buildLocation := func(location gamecontent.Location) worldLocationNode {
		node := worldLocationNode{
			ID: location.ID, Name: location.Name, Description: location.Description,
			EntryRoomID: entryOfLocation[location.ID],
		}
		if coords, ok := locationCoords[location.ID]; ok {
			node.Placed, node.X, node.Y = true, coords.X, coords.Y
		}
		node.Contents = contentsOfCell[gamecontent.ContainerKindLocation+":"+location.ID]
		owned := roomsOfLocation[location.ID]
		sort.SliceStable(owned, byNameLower(func(i int) string { return owned[i].Name }, len(owned)))
		for _, room := range owned {
			node.Rooms = append(node.Rooms, buildRoom(room, node.EntryRoomID))
		}
		return node
	}

	locationNodes := map[string]worldLocationNode{}
	locationsOfHub := map[string][]worldLocationNode{}
	for _, location := range locations {
		node := buildLocation(location)
		locationNodes[location.ID] = node
		if parent, ok := hubOfLocation[location.ID]; ok {
			locationsOfHub[parent] = append(locationsOfHub[parent], node)
		} else {
			snapshot.Unrooted = append(snapshot.Unrooted, node)
		}
	}

	sort.SliceStable(hubs, byNameLower(func(i int) string { return hubs[i].Name }, len(hubs)))
	for _, hub := range hubs {
		hubNode := worldHubNode{ID: hub.ID, Name: hub.Name}
		owned := locationsOfHub[hub.ID]
		sort.SliceStable(owned, byNameLower(func(i int) string { return owned[i].Name }, len(owned)))
		hubNode.Locations = owned
		snapshot.Hubs = append(snapshot.Hubs, hubNode)
	}
	sort.SliceStable(snapshot.Unrooted, byNameLower(func(i int) string { return snapshot.Unrooted[i].Name }, len(snapshot.Unrooted)))
	sort.SliceStable(snapshot.OrphanRooms, byNameLower(func(i int) string { return snapshot.OrphanRooms[i].Name }, len(snapshot.OrphanRooms)))

	// Containers, in tree order, so the picker reads like the world.
	// A Location with no Hub is labelled but still offered, and it
	// brings its own Rooms with it.
	addLocationContainers := func(prefix string, location worldLocationNode) {
		snapshot.Containers = append(snapshot.Containers, container{
			Kind: gamecontent.ContainerKindLocation, ID: location.ID, Name: location.Name,
			Path: prefix + location.Name,
		})
		for _, room := range location.Rooms {
			snapshot.Containers = append(snapshot.Containers, container{
				Kind: gamecontent.ContainerKindRoom, ID: room.ID, Name: room.Name,
				Path: prefix + location.Name + " › " + room.Name,
			})
		}
	}
	for _, hub := range snapshot.Hubs {
		for _, location := range hub.Locations {
			addLocationContainers(hub.Name+" › ", location)
		}
	}
	for _, location := range snapshot.Unrooted {
		addLocationContainers("(no Hub) › ", location)
	}
	for _, room := range snapshot.OrphanRooms {
		snapshot.Containers = append(snapshot.Containers, container{
			Kind: gamecontent.ContainerKindRoom, ID: room.ID, Name: room.Name,
			Path: "(no Location) › " + room.Name,
		})
	}

	pathOfCell := map[string]string{}
	for _, c := range snapshot.Containers {
		pathOfCell[c.Value()] = c.Path
	}
	for _, record := range contents {
		snapshot.Placement[contentKey(record.EntityKind, record.EntityID)] = pathOfCell[record.ParentKind+":"+record.ParentID]
	}
	snapshot.ContentsOf = contentsOfCell
	for kind, byID := range things {
		loose := make([]worldThing, 0, len(byID))
		placed := make([]worldThing, 0, len(byID))
		for _, thing := range byID {
			if placedThings[contentKey(kind, thing.ID)] {
				placed = append(placed, thing)
			} else {
				loose = append(loose, thing)
			}
		}
		sort.SliceStable(loose, byNameLower(func(i int) string { return loose[i].Name }, len(loose)))
		sort.SliceStable(placed, byNameLower(func(i int) string { return placed[i].Name }, len(placed)))
		snapshot.Loose[kind] = loose
		snapshot.Placed[kind] = placed
	}
	sort.SliceStable(snapshot.OrphanRooms, byNameLower(func(i int) string { return snapshot.OrphanRooms[i].Name }, len(snapshot.OrphanRooms)))
	return snapshot, nil
}

func coordinateLabel(placed bool, x, y int) string {
	if !placed {
		return "Unplaced"
	}
	return fmt.Sprintf("%d,%d", x, y)
}
