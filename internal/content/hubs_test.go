package content_test

import (
	"strings"
	"testing"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

func validHubsFile() gamecontent.HubsFile {
	return gamecontent.HubsFile{
		Version: gamecontent.HubsVersion,
		Hubs: []gamecontent.Hub{{
			ID:   testUUID,
			Name: "hub",
		}},
	}
}

func TestHubsRoundTripIsByteIdentical(t *testing.T) {
	first, err := gamecontent.EncodeHubs(validHubsFile())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := gamecontent.DecodeHubs(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := gamecontent.EncodeHubs(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("round trip changed bytes:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestHubsRejectInvalidContent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gamecontent.HubsFile)
	}{
		{"version", func(f *gamecontent.HubsFile) { f.Version = 99 }},
		{"uuid", func(f *gamecontent.HubsFile) { f.Hubs[0].ID = "invalid" }},
		{"name", func(f *gamecontent.HubsFile) { f.Hubs[0].Name = "  " }},
		{"duplicate UUID", func(f *gamecontent.HubsFile) { f.Hubs = append(f.Hubs, f.Hubs[0]) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := validHubsFile()
			tt.mutate(&file)
			if _, err := gamecontent.EncodeHubs(file); err == nil {
				t.Fatal("invalid hub content encoded without error")
			}
		})
	}
}

func TestHubsRejectUnknownMissingAndTrailingJSON(t *testing.T) {
	for _, data := range []string{
		`{"version":1,"hubs":[],"surprise":true}`,
		`{"version":1,"hubs":[]} {}`,
		`{"version":1}`,
	} {
		if _, err := gamecontent.DecodeHubs([]byte(data)); err == nil {
			t.Fatalf("invalid JSON accepted: %s", data)
		}
	}
}

func TestEmptyHubsEncodesAsAnArray(t *testing.T) {
	data, err := gamecontent.EncodeHubs(gamecontent.EmptyHubs())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"hubs": []`) {
		t.Fatalf("empty hub catalog did not encode as an array:\n%s", data)
	}
}
