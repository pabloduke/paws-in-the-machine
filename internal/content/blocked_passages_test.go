package content

import "testing"

func TestBlockedPassageSchema(t *testing.T) {
	c, err := DecodeCatalogs(nil)
	if err != nil || c.BlockedPassages.Version != 1 {
		t.Fatal("legacy catalog incompatible", err)
	}
	const a = "00000000-0000-4000-8000-000000000001"
	const b = "00000000-0000-4000-8000-000000000002"
	f := EmptyBlockedPassages()
	f.Passages = []BlockedPassage{{Kind: "room", AID: a, BID: b}, {Kind: "room", AID: b, BID: a}}
	if ValidateBlockedPassages(f) == nil {
		t.Fatal("reversed duplicate allowed")
	}
	c.LocationPlacements.Placements = []LocationPlacement{{LocationID: a, HubID: "parent"}, {LocationID: b, HubID: "parent", Y: 1}}
	if dir, ok := c.HorizontalNeighbors("location", a, b); !ok || dir != "north" {
		t.Fatal("north mismatch")
	}
	c.LocationPlacements.Placements[1].X = 1
	if _, ok := c.HorizontalNeighbors("location", a, b); ok {
		t.Fatal("diagonal allowed")
	}
	c.LocationPlacements.Placements[1].X = 0
	c.LocationPlacements.Placements[1].HubID = "other"
	if _, ok := c.HorizontalNeighbors("location", a, b); ok {
		t.Fatal("cross-parent allowed")
	}
}
