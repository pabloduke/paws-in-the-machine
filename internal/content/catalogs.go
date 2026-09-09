package content

import "fmt"

// Catalogs is an immutable input snapshot shared by tooling and runtime assembly.
type Catalogs struct {
	BlockedPassages     BlockedPassagesFile
	Quests              QuestsFile
	VerticalConnections VerticalConnectionsFile
	Hubs                HubsFile
	Locations           LocationsFile
	Rooms               RoomsFile
	NPCs                NPCsFile
	WorldItems          WorldItemsFile
	Terminals           TerminalsFile
	HostNetworks        HostNetworksFile
	Users               UsersFile
	LocationAssignments LocationAssignmentsFile
	RoomAssignments     RoomAssignmentsFile
	LocationPlacements  LocationPlacementsFile
	RoomPlacements      RoomPlacementsFile
	LocationEntries     LocationEntriesFile
	Contents            ContentsFile
	NetworkAssignments  NetworkAssignmentsFile
	TerminalAccess      TerminalAccessFile
	Play                PlaySettings
}

var CatalogNames = []string{
	"blocked_passages.json",
	"quests.json",
	"vertical_connections.json",
	"hubs.json",
	"locations.json",
	"rooms.json",
	"npcs.json",
	"world_items.json",
	"terminals.json",
	"host_networks.json",
	"users.json",
	"location_assignments.json",
	"room_assignments.json",
	"location_placements.json",
	"room_placements.json",
	"location_entry_rooms.json",
	"contents.json",
	"network_assignments.json",
	"terminal_access.json",
	"play_settings.json",
}

// DecodeCatalogs treats absent catalogs as empty, as the editor does.
func DecodeCatalogs(files map[string][]byte) (Catalogs, error) {
	var c Catalogs
	var err error
	c.BlockedPassages = EmptyBlockedPassages()
	if data, ok := files["blocked_passages.json"]; ok {
		c.BlockedPassages, err = DecodeBlockedPassages(data)
		if err != nil {
			return c, fmt.Errorf("blocked_passages.json: %w", err)
		}
	}
	c.Quests = EmptyQuests()
	if data, ok := files["quests.json"]; ok {
		c.Quests, err = DecodeQuests(data)
		if err != nil {
			return c, fmt.Errorf("quests.json: %w", err)
		}
	}
	c.VerticalConnections = EmptyVerticalConnections()
	if data, ok := files["vertical_connections.json"]; ok {
		c.VerticalConnections, err = DecodeVerticalConnections(data)
		if err != nil {
			return c, err
		}
	}
	c.Hubs = EmptyHubs()
	if data, ok := files["hubs.json"]; ok {
		c.Hubs, err = DecodeHubs(data)
		if err != nil {
			return c, fmt.Errorf("hubs.json: %w", err)
		}
	}
	c.Locations = EmptyLocations()
	if data, ok := files["locations.json"]; ok {
		c.Locations, err = DecodeLocations(data)
		if err != nil {
			return c, fmt.Errorf("locations.json: %w", err)
		}
	}
	c.Rooms = EmptyRooms()
	if data, ok := files["rooms.json"]; ok {
		c.Rooms, err = DecodeRooms(data)
		if err != nil {
			return c, fmt.Errorf("rooms.json: %w", err)
		}
	}
	c.NPCs = EmptyNPCs()
	if data, ok := files["npcs.json"]; ok {
		c.NPCs, err = DecodeNPCs(data)
		if err != nil {
			return c, fmt.Errorf("npcs.json: %w", err)
		}
	}
	c.WorldItems = EmptyWorldItems()
	if data, ok := files["world_items.json"]; ok {
		c.WorldItems, err = DecodeWorldItems(data)
		if err != nil {
			return c, fmt.Errorf("world_items.json: %w", err)
		}
	}
	c.Terminals = EmptyTerminals()
	if data, ok := files["terminals.json"]; ok {
		c.Terminals, err = DecodeTerminals(data)
		if err != nil {
			return c, fmt.Errorf("terminals.json: %w", err)
		}
	}
	c.HostNetworks = EmptyHostNetworks()
	if data, ok := files["host_networks.json"]; ok {
		c.HostNetworks, err = DecodeHostNetworks(data)
		if err != nil {
			return c, fmt.Errorf("host_networks.json: %w", err)
		}
	}
	c.Users = EmptyUsers()
	if data, ok := files["users.json"]; ok {
		c.Users, err = DecodeUsers(data)
		if err != nil {
			return c, fmt.Errorf("users.json: %w", err)
		}
	}
	c.LocationAssignments = EmptyLocationAssignments()
	if data, ok := files["location_assignments.json"]; ok {
		c.LocationAssignments, err = DecodeLocationAssignments(data)
		if err != nil {
			return c, fmt.Errorf("location_assignments.json: %w", err)
		}
	}
	c.RoomAssignments = EmptyRoomAssignments()
	if data, ok := files["room_assignments.json"]; ok {
		c.RoomAssignments, err = DecodeRoomAssignments(data)
		if err != nil {
			return c, fmt.Errorf("room_assignments.json: %w", err)
		}
	}
	c.LocationPlacements = EmptyLocationPlacements()
	if data, ok := files["location_placements.json"]; ok {
		c.LocationPlacements, err = DecodeLocationPlacements(data)
		if err != nil {
			return c, fmt.Errorf("location_placements.json: %w", err)
		}
	}
	c.RoomPlacements = EmptyRoomPlacements()
	if data, ok := files["room_placements.json"]; ok {
		c.RoomPlacements, err = DecodeRoomPlacements(data)
		if err != nil {
			return c, fmt.Errorf("room_placements.json: %w", err)
		}
	}
	c.LocationEntries = EmptyLocationEntries()
	if data, ok := files["location_entry_rooms.json"]; ok {
		c.LocationEntries, err = DecodeLocationEntries(data)
		if err != nil {
			return c, fmt.Errorf("location_entry_rooms.json: %w", err)
		}
	}
	c.Contents = EmptyContents()
	if data, ok := files["contents.json"]; ok {
		c.Contents, err = DecodeContents(data)
		if err != nil {
			return c, fmt.Errorf("contents.json: %w", err)
		}
	}
	c.NetworkAssignments = EmptyNetworkAssignments()
	if data, ok := files["network_assignments.json"]; ok {
		c.NetworkAssignments, err = DecodeNetworkAssignments(data)
		if err != nil {
			return c, fmt.Errorf("network_assignments.json: %w", err)
		}
	}
	c.TerminalAccess = EmptyTerminalAccess()
	if data, ok := files["terminal_access.json"]; ok {
		c.TerminalAccess, err = DecodeTerminalAccess(data)
		if err != nil {
			return c, fmt.Errorf("terminal_access.json: %w", err)
		}
	}
	c.Play = EmptyPlaySettings()
	if data, ok := files["play_settings.json"]; ok {
		c.Play, err = DecodePlaySettings(data)
	}
	return c, err
}
