package content

import "testing"

func TestLocationsRoundTripAndStrictValidation(t *testing.T) {
	file := LocationsFile{Version: LocationsVersion, Locations: []Location{{ID: testUUID, Name: "Location", Description: "Description"}}}
	data, err := EncodeLocations(file)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLocations(data)
	if err != nil || len(decoded.Locations) != 1 || decoded.Locations[0] != file.Locations[0] {
		t.Fatalf("round trip = %+v, %v", decoded, err)
	}
	for _, data := range [][]byte{
		[]byte(`{"version":1}`),
		[]byte(`{"version":1,"locations":[],"extra":true}`),
		[]byte(`{"version":1,"locations":[]} {}`),
	} {
		if _, err := DecodeLocations(data); err == nil {
			t.Fatalf("invalid JSON accepted: %s", data)
		}
	}
}

func TestLocationPlacementsRoundTripAndValidation(t *testing.T) {
	hubID := "123e4567-e89b-42d3-a456-426614174001"
	otherLocation := "123e4567-e89b-42d3-a456-426614174002"
	file := LocationPlacementsFile{Version: 1, Placements: []LocationPlacement{{LocationID: testUUID, HubID: hubID, X: 9, Y: 4}}}
	data, err := EncodeLocationPlacements(file)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLocationPlacements(data)
	if err != nil || len(decoded.Placements) != 1 || decoded.Placements[0] != file.Placements[0] {
		t.Fatalf("round trip = %+v, %v", decoded, err)
	}
	for name, invalid := range map[string]LocationPlacementsFile{
		"version":            {Version: 2, Placements: []LocationPlacement{}},
		"location UUID":      {Version: 1, Placements: []LocationPlacement{{LocationID: "bad", HubID: hubID}}},
		"hub UUID":           {Version: 1, Placements: []LocationPlacement{{LocationID: testUUID, HubID: "bad"}}},
		"negative":           {Version: 1, Placements: []LocationPlacement{{LocationID: testUUID, HubID: hubID, X: -1}}},
		"z dimension":        {Version: 1, Placements: []LocationPlacement{{LocationID: testUUID, HubID: hubID, Z: 1}}},
		"duplicate location": {Version: 1, Placements: []LocationPlacement{{LocationID: testUUID, HubID: hubID}, {LocationID: testUUID, HubID: hubID, X: 1}}},
		"occupied cell":      {Version: 1, Placements: []LocationPlacement{{LocationID: testUUID, HubID: hubID}, {LocationID: otherLocation, HubID: hubID}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateLocationPlacements(invalid); err == nil {
				t.Fatal("invalid placements accepted")
			}
		})
	}
}

func TestLocationEntriesRoundTripAndValidation(t *testing.T) {
	roomID := "123e4567-e89b-42d3-a456-426614174001"
	file := LocationEntriesFile{Version: 1, Entries: []LocationEntry{{LocationID: testUUID, RoomID: roomID}}}
	data, err := EncodeLocationEntries(file)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLocationEntries(data)
	if err != nil || len(decoded.Entries) != 1 || decoded.Entries[0] != file.Entries[0] {
		t.Fatalf("round trip = %+v, %v", decoded, err)
	}
	for name, invalid := range map[string]LocationEntriesFile{
		"version":            {Version: 2, Entries: []LocationEntry{}},
		"location UUID":      {Version: 1, Entries: []LocationEntry{{LocationID: "bad", RoomID: roomID}}},
		"room UUID":          {Version: 1, Entries: []LocationEntry{{LocationID: testUUID, RoomID: "bad"}}},
		"duplicate location": {Version: 1, Entries: []LocationEntry{{LocationID: testUUID, RoomID: roomID}, {LocationID: testUUID, RoomID: roomID}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateLocationEntries(invalid); err == nil {
				t.Fatal("invalid entries accepted")
			}
		})
	}
}

func TestSpatialAssignmentsRoundTripAndValidation(t *testing.T) {
	parentID := "123e4567-e89b-42d3-a456-426614174001"
	locationFile := LocationAssignmentsFile{Version: 1, Assignments: []LocationAssignment{{LocationID: testUUID, HubID: parentID}}}
	data, err := EncodeLocationAssignments(locationFile)
	if err != nil {
		t.Fatal(err)
	}
	decodedLocations, err := DecodeLocationAssignments(data)
	if err != nil || len(decodedLocations.Assignments) != 1 || decodedLocations.Assignments[0] != locationFile.Assignments[0] {
		t.Fatalf("location assignment round trip = %+v, %v", decodedLocations, err)
	}
	if err := ValidateLocationAssignments(LocationAssignmentsFile{Version: 1, Assignments: []LocationAssignment{{LocationID: testUUID, HubID: parentID}, {LocationID: testUUID, HubID: parentID}}}); err == nil {
		t.Fatal("duplicate location assignment accepted")
	}

	roomFile := RoomAssignmentsFile{Version: 1, Assignments: []RoomAssignment{{RoomID: testUUID, LocationID: parentID}}}
	data, err = EncodeRoomAssignments(roomFile)
	if err != nil {
		t.Fatal(err)
	}
	decodedRooms, err := DecodeRoomAssignments(data)
	if err != nil || len(decodedRooms.Assignments) != 1 || decodedRooms.Assignments[0] != roomFile.Assignments[0] {
		t.Fatalf("room assignment round trip = %+v, %v", decodedRooms, err)
	}
	if err := ValidateRoomAssignments(RoomAssignmentsFile{Version: 1, Assignments: []RoomAssignment{{RoomID: testUUID, LocationID: parentID}, {RoomID: testUUID, LocationID: parentID}}}); err == nil {
		t.Fatal("duplicate room assignment accepted")
	}

	for _, invalid := range [][]byte{
		[]byte(`{"version":1}`),
		[]byte(`{"version":1,"assignments":[],"extra":true}`),
		[]byte(`{"version":1,"assignments":[]} {}`),
	} {
		if _, err := DecodeLocationAssignments(invalid); err == nil {
			t.Fatalf("invalid location assignments accepted: %s", invalid)
		}
		if _, err := DecodeRoomAssignments(invalid); err == nil {
			t.Fatalf("invalid room assignments accepted: %s", invalid)
		}
	}
}
