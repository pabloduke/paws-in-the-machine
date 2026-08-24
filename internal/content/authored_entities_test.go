package content

import "testing"

const testUUID = "123e4567-e89b-42d3-a456-426614174000"

func TestRoomsAndNPCsRoundTrip(t *testing.T) {
	rooms := RoomsFile{Version: RoomsVersion, Rooms: []Room{{ID: testUUID, Name: "Room", Description: "Description"}}}
	roomData, err := EncodeRooms(rooms)
	if err != nil {
		t.Fatal(err)
	}
	decodedRooms, err := DecodeRooms(roomData)
	if err != nil || len(decodedRooms.Rooms) != 1 || decodedRooms.Rooms[0] != rooms.Rooms[0] {
		t.Fatalf("room round trip = %+v, %v", decodedRooms, err)
	}

	npcs := NPCsFile{Version: NPCsVersion, NPCs: []NPC{{ID: testUUID, Name: "NPC", Description: "Description"}}}
	npcData, err := EncodeNPCs(npcs)
	if err != nil {
		t.Fatal(err)
	}
	decodedNPCs, err := DecodeNPCs(npcData)
	if err != nil || len(decodedNPCs.NPCs) != 1 || decodedNPCs.NPCs[0] != npcs.NPCs[0] {
		t.Fatalf("NPC round trip = %+v, %v", decodedNPCs, err)
	}
}

func TestDescribedEntitySchemasRejectInvalidContent(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"room version", ValidateRooms(RoomsFile{Version: 2, Rooms: []Room{}})},
		{"room UUID", ValidateRooms(RoomsFile{Version: RoomsVersion, Rooms: []Room{{ID: "bad", Name: "Room", Description: "Description"}}})},
		{"room name", ValidateRooms(RoomsFile{Version: RoomsVersion, Rooms: []Room{{ID: testUUID, Name: " ", Description: "Description"}}})},
		{"room description", ValidateRooms(RoomsFile{Version: RoomsVersion, Rooms: []Room{{ID: testUUID, Name: "Room", Description: " "}}})},
		{"duplicate room UUID", ValidateRooms(RoomsFile{Version: RoomsVersion, Rooms: []Room{{ID: testUUID, Name: "One", Description: "Description"}, {ID: testUUID, Name: "Two", Description: "Description"}}})},
		{"NPC version", ValidateNPCs(NPCsFile{Version: 2, NPCs: []NPC{}})},
		{"NPC name", ValidateNPCs(NPCsFile{Version: NPCsVersion, NPCs: []NPC{{ID: testUUID, Name: "", Description: "Description"}}})},
		{"NPC description", ValidateNPCs(NPCsFile{Version: NPCsVersion, NPCs: []NPC{{ID: testUUID, Name: "NPC", Description: ""}}})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("invalid content was accepted")
			}
		})
	}
}

func TestDescribedEntityDecodersAreStrict(t *testing.T) {
	for name, decode := range map[string]func([]byte) error{
		"rooms missing array": func(data []byte) error { _, err := DecodeRooms(data); return err },
		"NPCs unknown field":  func(data []byte) error { _, err := DecodeNPCs(data); return err },
		"rooms trailing JSON": func(data []byte) error { _, err := DecodeRooms(data); return err },
	} {
		var data []byte
		switch name {
		case "rooms missing array":
			data = []byte(`{"version":1}`)
		case "NPCs unknown field":
			data = []byte(`{"version":1,"npcs":[],"extra":true}`)
		default:
			data = []byte(`{"version":1,"rooms":[]} {}`)
		}
		if err := decode(data); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestTerminalsRoundTripAndValidation(t *testing.T) {
	file := TerminalsFile{Version: TerminalsVersion, Terminals: []Terminal{{ID: testUUID, Username: "paws_in_the_machine", HostName: "undernet.relay"}}}
	data, err := EncodeTerminals(file)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeTerminals(data)
	if err != nil || len(decoded.Terminals) != 1 || decoded.Terminals[0] != file.Terminals[0] {
		t.Fatalf("terminal round trip = %+v, %v", decoded, err)
	}

	for _, username := range []string{"paws", "paws_in_the_machine", "_service", "user-2"} {
		if !ValidTerminalUsername(username) {
			t.Errorf("valid username %q rejected", username)
		}
	}
	for _, username := range []string{"", "Paws", "2paws", "paws cat"} {
		if ValidTerminalUsername(username) {
			t.Errorf("invalid username %q accepted", username)
		}
	}
	for _, host := range []string{"deck", "undernet.relay", "sun-farm", "host_2"} {
		if !ValidTerminalHostName(host) {
			t.Errorf("valid host %q rejected", host)
		}
	}
	for _, host := range []string{"", "Deck", ".deck", "deck.", "deck host"} {
		if ValidTerminalHostName(host) {
			t.Errorf("invalid host %q accepted", host)
		}
	}
}

func TestTerminalsAllowDuplicateHostNamesUntilNetworkAssignment(t *testing.T) {
	secondUUID := "123e4567-e89b-42d3-a456-426614174001"
	file := TerminalsFile{Version: TerminalsVersion, Terminals: []Terminal{
		{ID: testUUID, Username: "one", HostName: "deck"},
		{ID: secondUUID, Username: "two", HostName: "deck"},
	}}
	if err := ValidateTerminals(file); err != nil {
		t.Fatalf("unassigned duplicate host name was rejected: %v", err)
	}
}

func TestTerminalsDecoderIsStrict(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"version":1}`),
		[]byte(`{"version":1,"terminals":[],"extra":true}`),
		[]byte(`{"version":1,"terminals":[]} {}`),
	} {
		if _, err := DecodeTerminals(data); err == nil {
			t.Fatalf("invalid terminal JSON accepted: %s", data)
		}
	}
}
