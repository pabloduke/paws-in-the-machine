package content

import "testing"

func TestRoomPlacementsRoundTripAndValidation(t *testing.T) {
	hubID := "123e4567-e89b-42d3-a456-426614174001"
	file := RoomPlacementsFile{Version: RoomPlacementsVersion, Placements: []RoomPlacement{{RoomID: testUUID, LocationID: hubID, X: 9, Y: 4}}}
	data, err := EncodeRoomPlacements(file)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRoomPlacements(data)
	if err != nil || len(decoded.Placements) != 1 || decoded.Placements[0] != file.Placements[0] {
		t.Fatalf("round trip = %+v, %v", decoded, err)
	}
	otherRoom := "123e4567-e89b-42d3-a456-426614174002"
	for name, invalid := range map[string]RoomPlacementsFile{
		"version":        {Version: 2, Placements: []RoomPlacement{}},
		"room UUID":      {Version: 1, Placements: []RoomPlacement{{RoomID: "bad", LocationID: hubID}}},
		"location UUID":  {Version: 1, Placements: []RoomPlacement{{RoomID: testUUID, LocationID: "bad"}}},
		"negative":       {Version: 1, Placements: []RoomPlacement{{RoomID: testUUID, LocationID: hubID, X: -1}}},
		"z dimension":    {Version: 1, Placements: []RoomPlacement{{RoomID: testUUID, LocationID: hubID, Z: 1}}},
		"duplicate room": {Version: 1, Placements: []RoomPlacement{{RoomID: testUUID, LocationID: hubID}, {RoomID: testUUID, LocationID: hubID, X: 1}}},
		"occupied cell":  {Version: 1, Placements: []RoomPlacement{{RoomID: testUUID, LocationID: hubID}, {RoomID: otherRoom, LocationID: hubID}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateRoomPlacements(invalid); err == nil {
				t.Fatal("invalid placements accepted")
			}
		})
	}
}

func TestRoomPlacementsDecoderIsStrict(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("{\"version\":1}"),
		[]byte("{\"version\":1,\"placements\":[],\"extra\":true}"),
		[]byte("{\"version\":1,\"placements\":[]} {}"),
	} {
		if _, err := DecodeRoomPlacements(data); err == nil {
			t.Fatalf("invalid JSON accepted: %s", data)
		}
	}
}
