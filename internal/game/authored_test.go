package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

const (
	ah  = "00000000-0000-4000-8000-000000000001"
	al  = "00000000-0000-4000-8000-000000000002"
	al2 = "00000000-0000-4000-8000-000000000003"
	ar  = "00000000-0000-4000-8000-000000000004"
	ar2 = "00000000-0000-4000-8000-000000000005"
	ai  = "00000000-0000-4000-8000-000000000006"
	af  = "00000000-0000-4000-8000-000000000007"
	asc = "00000000-0000-4000-8000-000000000008"
	an  = "00000000-0000-4000-8000-000000000009"
	at  = "00000000-0000-4000-8000-000000000010"
)

func authoredFixture() content.Catalogs {
	c, _ := content.DecodeCatalogs(nil)
	c.Hubs.Hubs = []content.Hub{{ID: ah, Name: "Test Hub"}}
	c.Locations.Locations = []content.Location{{ID: al, Name: "Test Exterior", Description: "(Placeholder)"}, {ID: al2, Name: "Adjacent Exterior", Description: "(Placeholder)"}}
	c.Rooms.Rooms = []content.Room{{ID: ar, Name: "Test Interior", Description: "(Placeholder)"}, {ID: ar2, Name: "Adjacent Interior", Description: "(Placeholder)"}}
	c.LocationAssignments.Assignments = []content.LocationAssignment{{LocationID: al, HubID: ah}, {LocationID: al2, HubID: ah}}
	c.RoomAssignments.Assignments = []content.RoomAssignment{{RoomID: ar, LocationID: al}, {RoomID: ar2, LocationID: al}}
	c.LocationPlacements.Placements = []content.LocationPlacement{{LocationID: al, HubID: ah}, {LocationID: al2, HubID: ah, X: 1}}
	c.RoomPlacements.Placements = []content.RoomPlacement{{RoomID: ar, LocationID: al}, {RoomID: ar2, LocationID: al, Y: 1}}
	c.LocationEntries.Entries = []content.LocationEntry{{LocationID: al, RoomID: ar}}
	c.Play = content.PlaySettings{Version: 1, Start: &content.CellRef{Kind: "location", ID: al}, HubArrivals: map[string]string{ah: al2}}
	c.WorldItems.WorldItems = []content.WorldItem{
		{ID: ai, Name: "Test Item", Kind: content.WorldItemTakeable, ShortDescription: "short item text", FullDescription: "full item text"},
		{ID: af, Name: "Fixed Item", Kind: content.WorldItemFixed, ShortDescription: "short fixed text", FullDescription: "full fixed text"},
		{ID: asc, Name: "Scenery Item", Kind: content.WorldItemScenery, ShortDescription: "short scenery text", FullDescription: "full scenery text"},
		// Catalog-local UUID reuse must never collide with a cell.
		{ID: al, Name: "Exterior Item", Kind: content.WorldItemTakeable, ShortDescription: "short exterior text", FullDescription: "full exterior text"},
	}
	c.NPCs.NPCs = []content.NPC{{ID: an, Name: "Test NPC", Description: "(Placeholder)"}}
	c.Terminals.Terminals = []content.Terminal{{ID: at, HostName: "test-host"}}
	for _, id := range []string{ai, af, asc} {
		c.Contents.Contents = append(c.Contents.Contents, content.Content{EntityID: id, EntityKind: "world_item", ParentID: ar, ParentKind: "room"})
	}
	c.Contents.Contents = append(c.Contents.Contents,
		content.Content{EntityID: al, EntityKind: "world_item", ParentID: al, ParentKind: "location"},
		content.Content{EntityID: an, EntityKind: "npc", ParentID: ar, ParentKind: "room"},
		content.Content{EntityID: at, EntityKind: "terminal", ParentID: ar, ParentKind: "room"})
	return c
}
func TestAuthoredWorldExploration(t *testing.T) {
	result := AssembleAuthored(authoredFixture())
	if !result.Ready() {
		t.Fatal(result.DiagnosticText())
	}
	w := result.World
	eng := engine.New(w)
	if w.Room().ID != authoredID("location", al) {
		t.Fatal("wrong start")
	}
	if w.InScope("test item") != nil || w.InScope("test interior") != nil {
		t.Fatal("interior leaked into exterior scope")
	}
	if w.InScope("exterior item") == nil {
		t.Fatal("capitalized item is not targetable")
	}
	if !strings.Contains(eng.Execute("take Test Item"), "don't see") {
		t.Fatal("could take interior item from outside")
	}
	for _, step := range []struct{ dir, id string }{{"east", al2}, {"west", al}} {
		engine.Go(w, step.dir)
		if w.Room().ID != authoredID("location", step.id) {
			t.Fatalf("bad %s exit", step.dir)
		}
	}
	engine.Go(w, "down")
	if w.Room().ID != authoredID("room", ar) || hubs.Current(w).ID != authoredID("hub", ah) {
		t.Fatal("interior descent or ancestry failed")
	}
	if w.InScope("exterior item") != nil {
		t.Fatal("exterior leaked into interior")
	}
	engine.Go(w, "north")
	if w.Room().ID != authoredID("room", ar2) {
		t.Fatal("interior adjacency failed")
	}
	engine.Go(w, "south")
	if got := eng.Execute("examine Test Item"); got != "full item text" {
		t.Fatalf("examine: %q", got)
	}
	item := w.FindID(authoredID("world_item", ai))
	if got := engine.Listing(w, item); got != "Test Item — short item text" {
		t.Fatalf("listing: %q", got)
	}
	obvious := map[string]bool{}
	for _, e := range w.Obvious() {
		obvious[e.ID] = true
	}
	if !obvious[item.ID] || !obvious[authoredID("world_item", af)] || obvious[authoredID("world_item", asc)] || !obvious[authoredID("npc", an)] {
		t.Fatal("incorrect visibility")
	}
	eng.Execute("take Test Item")
	if !w.Carried(item) {
		t.Fatal("take failed")
	}
	eng.Execute("drop Test Item")
	if item.Parent != w.Room() {
		t.Fatal("drop failed")
	}
	for _, name := range []string{"Fixed Item", "Scenery Item"} {
		eng.Execute("take " + name)
	}
	if w.Carried(w.FindID(authoredID("world_item", af))) || w.Carried(w.FindID(authoredID("world_item", asc))) {
		t.Fatal("nonportable item taken")
	}
	if got := eng.Execute("examine Test NPC"); got != "(Placeholder)" {
		t.Fatal(got)
	}
	if got := eng.Execute("use test-host"); !strings.Contains(got, "(Placeholder)") {
		t.Fatal(got)
	}
	engine.Go(w, "up")
	if w.Room().ID != authoredID("location", al) {
		t.Fatal("return gluing failed")
	}
	if got := hubs.Travel(w, authoredID("hub", ah)); got != "" || w.Room().ID != authoredID("location", al2) {
		t.Fatal("arrival failed", got)
	}
	if len(w.Rules) != 0 || len(w.Journal) != 0 || w.FindID("deck") != nil {
		t.Fatal("built-in content leaked into authored world")
	}
	fresh := AssembleAuthored(authoredFixture()).World
	if fresh.Room().ID != authoredID("location", al) || fresh.FindID(item.ID).Parent.ID != authoredID("room", ar) {
		t.Fatal("session mutation leaked")
	}
}

func TestAuthoredDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		change func(*content.Catalogs)
		ready  bool
		want   string
	}{
		{"missing start", func(c *content.Catalogs) { c.Play.Start = nil }, false, "starting"},
		{"missing arrival", func(c *content.Catalogs) { delete(c.Play.HubArrivals, ah) }, false, "arrival"},
		{"bad start", func(c *content.Catalogs) { c.Play.Start.ID = ai }, false, "Starting"},
		{"bad arrival", func(c *content.Catalogs) { c.Play.HubArrivals[ah] = ar }, false, "Arrival"},
		{"wrong owner", func(c *content.Catalogs) { c.LocationAssignments.Assignments = nil }, false, "disagrees"},
		{"missing item", func(c *content.Catalogs) { c.WorldItems.WorldItems = nil }, false, "Missing world_item"},
		{"bad interior entry", func(c *content.Catalogs) { c.LocationEntries.Entries[0].RoomID = al }, false, "Entry Room"},
		{"disconnected", func(c *content.Catalogs) { c.RoomPlacements.Placements[1].Y = 3 }, true, "unreachable"},
		{"no interior entry", func(c *content.Catalogs) { c.LocationEntries.Entries = nil }, true, "no entry Room"},
		{"unplaced interior", func(c *content.Catalogs) { c.LocationEntries.Entries = nil; c.RoomPlacements.Placements = nil }, true, "Excluded Room"},
		{"empty hub", func(c *content.Catalogs) {
			c.Hubs.Hubs = append(c.Hubs.Hubs, content.Hub{ID: ai, Name: "Empty Test Hub"})
		}, true, "Excluded empty Hub"},
		{"room start", func(c *content.Catalogs) { c.Play.Start = &content.CellRef{Kind: "room", ID: ar2} }, true, "Terminal filesystems"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := authoredFixture()
			tt.change(&c)
			r := AssembleAuthored(c)
			if r.Ready() != tt.ready || !strings.Contains(r.DiagnosticText(), tt.want) {
				t.Fatalf("ready=%v: %s", r.Ready(), r.DiagnosticText())
			}
		})
	}
}
func TestAuthoredDirectoryFailures(t *testing.T) {
	dir := t.TempDir()
	if r := LoadAuthored(dir); r.Ready() || !strings.Contains(r.DiagnosticText(), "starting") {
		t.Fatal(r)
	}
	for _, text := range []string{"{", `{"version":99,"hubs":[]}`, `{"version":1,"hubs":[],"unknown":true}`} {
		if err := os.WriteFile(filepath.Join(dir, "hubs.json"), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if r := LoadAuthored(dir); r.Ready() || !strings.Contains(r.DiagnosticText(), "hubs.json") {
			t.Fatal(r)
		}
	}
	if r := LoadAuthored(filepath.Join(dir, "missing")); r.Ready() {
		t.Fatal("missing directory accepted")
	}
}
