package content

import (
	"fmt"
	"sort"
)

// RelationshipProblems validates references even for content excluded from play.
// Missing play settings are draft-valid and are checked separately by the loader.
func (c Catalogs) RelationshipProblems() []string {
	var problems []string
	if err := ValidateBlockedGeometry(c); err != nil {
		problems = append(problems, err.Error())
	}
	if c.Quests.Version != 0 || len(c.Quests.Quests) > 0 {
		if err := c.Quests.Validate(); err != nil {
			problems = append(problems, err.Error())
		}
	}
	for _, q := range c.Quests.Quests {
		if q.Enabled {
			for _, issue := range c.QuestProblems(q) {
				problems = append(problems, fmt.Sprintf("Quest %q: %s", q.Name, issue))
			}
		}
	}
	report := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	ids := map[string]map[string]bool{}
	for _, kind := range []string{"hub", "location", "room", "npc", "world_item", "terminal", "network", "user"} {
		ids[kind] = map[string]bool{}
	}
	for _, v := range c.Hubs.Hubs {
		ids["hub"][v.ID] = true
	}
	for _, v := range c.Locations.Locations {
		ids["location"][v.ID] = true
	}
	for _, v := range c.Rooms.Rooms {
		ids["room"][v.ID] = true
	}
	for _, v := range c.NPCs.NPCs {
		ids["npc"][v.ID] = true
	}
	for _, v := range c.WorldItems.WorldItems {
		ids["world_item"][v.ID] = true
	}
	for _, v := range c.Terminals.Terminals {
		ids["terminal"][v.ID] = true
	}
	for _, v := range c.HostNetworks.HostNetworks {
		ids["network"][v.ID] = true
	}
	for _, v := range c.Users.Users {
		ids["user"][v.ID] = true
	}
	exists := func(kind, id string) bool {
		if !ids[kind][id] {
			report("Missing %s %s referenced by authored content.", kind, id)
			return false
		}
		return true
	}
	la, ra := map[string]string{}, map[string]string{}
	for _, a := range c.LocationAssignments.Assignments {
		exists("location", a.LocationID)
		exists("hub", a.HubID)
		la[a.LocationID] = a.HubID
	}
	for _, a := range c.RoomAssignments.Assignments {
		exists("room", a.RoomID)
		exists("location", a.LocationID)
		ra[a.RoomID] = a.LocationID
	}
	lp, rp := map[string]string{}, map[string]string{}
	for _, p := range c.LocationPlacements.Placements {
		exists("location", p.LocationID)
		exists("hub", p.HubID)
		if la[p.LocationID] != p.HubID {
			report("Location %s placement disagrees with its Hub assignment.", p.LocationID)
		}
		lp[p.LocationID] = p.HubID
	}
	for _, p := range c.RoomPlacements.Placements {
		exists("room", p.RoomID)
		exists("location", p.LocationID)
		if ra[p.RoomID] != p.LocationID {
			report("Room %s placement disagrees with its Location assignment.", p.RoomID)
		}
		rp[p.RoomID] = p.LocationID
	}
	for _, e := range c.LocationEntries.Entries {
		exists("location", e.LocationID)
		exists("room", e.RoomID)
		if rp[e.RoomID] != e.LocationID {
			report("Entry Room %s is not placed in Location %s.", e.RoomID, e.LocationID)
		}
	}
	for _, v := range c.Contents.Contents {
		exists(v.EntityKind, v.EntityID)
		exists(v.ParentKind, v.ParentID)
	}
	for _, v := range c.NetworkAssignments.Assignments {
		exists("terminal", v.TerminalID)
		exists("network", v.HostNetworkID)
	}
	for _, v := range c.TerminalAccess.Access {
		exists("terminal", v.TerminalID)
		exists("user", v.UserID)
	}
	for hub, loc := range c.Play.HubArrivals {
		exists("hub", hub)
		exists("location", loc)
		if lp[loc] != hub {
			report("Arrival Location %s is not placed in Hub %s.", loc, hub)
		}
	}
	if start := c.Play.Start; start != nil {
		exists(start.Kind, start.ID)
		placed := lp[start.ID] != ""
		if start.Kind == ContainerKindRoom {
			placed = rp[start.ID] != "" && lp[rp[start.ID]] != ""
		}
		if !placed {
			report("Starting %s %s must be placed with placed ancestry.", start.Kind, start.ID)
		}
	}
	sort.Strings(problems)
	return problems
}
