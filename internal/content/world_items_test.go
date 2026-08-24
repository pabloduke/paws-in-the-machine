package content_test

import (
	"strings"
	"testing"

	gamecontent "github.com/pabloduke/paws-in-the-machine/internal/content"
)

const testUUID = "123e4567-e89b-42d3-a456-426614174000"

func validFile() gamecontent.WorldItemsFile {
	return gamecontent.WorldItemsFile{
		Version: gamecontent.WorldItemsVersion,
		WorldItems: []gamecontent.WorldItem{{
			ID:               testUUID,
			Name:             "item",
			Kind:             gamecontent.WorldItemTakeable,
			ShortDescription: "short",
			FullDescription:  "full",
		}},
	}
}

func TestWorldItemsRoundTripIsByteIdentical(t *testing.T) {
	first, err := gamecontent.EncodeWorldItems(validFile())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := gamecontent.DecodeWorldItems(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := gamecontent.EncodeWorldItems(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("round trip changed bytes:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !strings.HasSuffix(string(first), "\n") {
		t.Fatal("encoded content must end in a newline")
	}
}

func TestWorldItemsRejectInvalidContent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gamecontent.WorldItemsFile)
	}{
		{"version", func(f *gamecontent.WorldItemsFile) { f.Version = 99 }},
		{"uuid", func(f *gamecontent.WorldItemsFile) { f.WorldItems[0].ID = "not-a-uuid" }},
		{"name", func(f *gamecontent.WorldItemsFile) { f.WorldItems[0].Name = "  " }},
		{"kind", func(f *gamecontent.WorldItemsFile) { f.WorldItems[0].Kind = "container" }},
		{"short description", func(f *gamecontent.WorldItemsFile) { f.WorldItems[0].ShortDescription = "" }},
		{"full description", func(f *gamecontent.WorldItemsFile) { f.WorldItems[0].FullDescription = "" }},
		{"duplicate UUID", func(f *gamecontent.WorldItemsFile) {
			f.WorldItems = append(f.WorldItems, f.WorldItems[0])
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := validFile()
			tt.mutate(&file)
			if _, err := gamecontent.EncodeWorldItems(file); err == nil {
				t.Fatal("invalid content encoded without error")
			}
		})
	}
}

func TestWorldItemsRejectUnknownAndTrailingJSON(t *testing.T) {
	for _, data := range []string{
		`{"version":1,"world_items":[],"surprise":true}`,
		`{"version":1,"world_items":[]} {}`,
		`{"version":1}`,
	} {
		if _, err := gamecontent.DecodeWorldItems([]byte(data)); err == nil {
			t.Fatalf("invalid JSON accepted: %s", data)
		}
	}
}

func TestEmptyWorldItemsEncodesAsAnArray(t *testing.T) {
	data, err := gamecontent.EncodeWorldItems(gamecontent.EmptyWorldItems())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"world_items": []`) {
		t.Fatalf("empty catalog did not encode as an array:\n%s", data)
	}
}
