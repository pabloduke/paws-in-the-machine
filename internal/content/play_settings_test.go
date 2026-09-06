package content

import (
	"strings"
	"testing"
)

func TestPlaySettingsSchema(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	settings := EmptyPlaySettings()
	settings.Start = &CellRef{Kind: "room", ID: id}
	settings.HubArrivals[id] = id
	b, err := EncodePlaySettings(settings)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePlaySettings(b)
	if err != nil || decoded.Start == nil || *decoded.Start != *settings.Start || decoded.HubArrivals[id] != id {
		t.Fatal("round trip failed", err)
	}
	for _, text := range []string{
		`{"version":2,"hub_arrivals":{}}`,
		`{"version":1,"hub_arrivals":null}`,
		`{"version":1,"hub_arrivals":{},"extra":true}`,
		`{"version":1,"hub_arrivals":{},"start":{"kind":"hub","id":"` + id + `"}}`,
		`{"version":1,"hub_arrivals":{"bad":"` + id + `"}}`,
		`{"version":1,"hub_arrivals":{}} {}`,
	} {
		if _, err := DecodePlaySettings([]byte(text)); err == nil {
			t.Fatalf("accepted invalid settings %s", text)
		}
	}
}

func TestCatalogDecoderRejectsDuplicateCells(t *testing.T) {
	id1 := "00000000-0000-4000-8000-000000000001"
	id2 := "00000000-0000-4000-8000-000000000002"
	data := `{"version":1,"placements":[{"location_id":"` + id1 + `","hub_id":"` + id1 + `","x":0,"y":0,"z":0,"w":0},{"location_id":"` + id2 + `","hub_id":"` + id1 + `","x":0,"y":0,"z":0,"w":0}]}`
	if _, err := DecodeCatalogs(map[string][]byte{"location_placements.json": []byte(data)}); err == nil || !strings.Contains(err.Error(), "location_placements.json") {
		t.Fatal("duplicate cells accepted", err)
	}
}
