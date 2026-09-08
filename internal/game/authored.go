package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/charts"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// Diagnostic describes authoring errors (blocking) or draft limitations (warnings).
type Diagnostic struct{ Severity, Message string }
type AuthoredResult struct {
	World       *engine.World
	Diagnostics []Diagnostic
}

func (r AuthoredResult) Ready() bool { return r.World != nil }

// ReadCatalogs reads one saved snapshot without mutating editor files. Unknown
// files, including legacy charts.json, are not runtime input for authored worlds.
func ReadCatalogs(dir string) (content.Catalogs, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return content.Catalogs{}, err
	}
	if !info.IsDir() {
		return content.Catalogs{}, fmt.Errorf("content path is not a directory: %s", dir)
	}
	files := map[string][]byte{}
	for _, name := range content.CatalogNames {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return content.Catalogs{}, fmt.Errorf("%s: %w", name, err)
		}
		files[name] = b
	}
	return content.DecodeCatalogs(files)
}

func LoadAuthored(dir string) AuthoredResult {
	c, err := ReadCatalogs(dir)
	if err != nil {
		return AuthoredResult{Diagnostics: []Diagnostic{{"error", err.Error()}}}
	}
	return AssembleAuthored(c)
}

func authoredID(kind, id string) string { return kind + ":" + id }

// AssembleAuthored consumes validated catalogs. It never augments an authored
// world with built-in missions, characters, terminals, or event rules.
func AssembleAuthored(c content.Catalogs) AuthoredResult {
	result := AuthoredResult{}
	report := func(severity, message string) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{severity, message})
	}
	for _, problem := range c.RelationshipProblems() {
		report("error", problem)
	}
	if c.Play.Start == nil {
		report("error", "Choose a starting Location or Room in World Overview.")
	}
	lp, rp := map[string]content.LocationPlacement{}, map[string]content.RoomPlacement{}
	hubHasCells := map[string]bool{}
	for _, p := range c.LocationPlacements.Placements {
		lp[p.LocationID] = p
		hubHasCells[p.HubID] = true
	}
	for _, p := range c.RoomPlacements.Placements {
		rp[p.RoomID] = p
	}
	for _, h := range c.Hubs.Hubs {
		if !hubHasCells[h.ID] {
			report("warning", fmt.Sprintf("Excluded empty Hub %q.", h.Name))
		} else if c.Play.HubArrivals[h.ID] == "" {
			report("error", fmt.Sprintf("Choose an arrival Location for Hub %q.", h.Name))
		}
	}
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			return result
		}
	}
	w := engine.NewWorld()
	w.Stats = engine.Stats{Stealth: 10, Agility: 12, Charm: 8}
	entities := map[string]*engine.Entity{}
	chartCells := map[string]map[charts.Coord]string{}
	cellChart := map[string]string{}
	addCell := func(kind, id, name, description, parent, chart string, x, y, z int) {
		key := authoredID(kind, id)
		e := engine.NewEntity(key, name).With(engine.RoomBoundary{}, engine.Description{Text: description})
		entities[parent].Add(e)
		entities[key] = e
		if chartCells[chart] == nil {
			chartCells[chart] = map[charts.Coord]string{}
		}
		at := charts.Coord{X: x, Y: y, Z: z}
		chartCells[chart][at] = key
		cellChart[key] = chart
	}
	for _, h := range c.Hubs.Hubs {
		if !hubHasCells[h.ID] {
			continue
		}
		e := engine.NewEntity(authoredID("hub", h.ID), h.Name).With(hubs.Hub{Entry: authoredID("location", c.Play.HubArrivals[h.ID])})
		entities[e.ID] = e
		w.Root.Add(e)
	}
	for _, l := range c.Locations.Locations {
		p, ok := lp[l.ID]
		if !ok {
			report("warning", fmt.Sprintf("Excluded unplaced Location %q.", l.Name))
			continue
		}
		addCell("location", l.ID, l.Name, l.Description, authoredID("hub", p.HubID), authoredID("hub", p.HubID), p.X, p.Y, p.Z)
	}
	for _, r := range c.Rooms.Rooms {
		p, ok := rp[r.ID]
		if !ok || entities[authoredID("location", p.LocationID)] == nil {
			report("warning", fmt.Sprintf("Excluded Room %q: it or its ancestry is unplaced.", r.Name))
			continue
		}
		addCell("room", r.ID, r.Name, r.Description, authoredID("location", p.LocationID), authoredID("location", p.LocationID), p.X, p.Y, p.Z)
	}
	chartIDs := make([]string, 0, len(chartCells))
	for id := range chartCells {
		chartIDs = append(chartIDs, id)
	}
	sort.Strings(chartIDs)
	cs := make([]*charts.Chart, 0, len(chartIDs))
	for _, id := range chartIDs {
		cs = append(cs, charts.New(id, chartCells[id]))
	}
	weave := charts.NewWeave(cs...)
	entryFor := map[string]string{}
	for _, e := range c.LocationEntries.Entries {
		entryFor[e.LocationID] = e.RoomID
	}
	for _, l := range c.Locations.Locations {
		if len(chartCells[authoredID("location", l.ID)]) > 0 && entryFor[l.ID] == "" {
			report("warning", fmt.Sprintf("Location %q has interior Rooms but no entry Room.", l.Name))
		}
	}
	for _, bug := range weave.Apply(w) {
		report("error", bug)
	}
	// Authored worlds use explicit vertical links, with separate interior boundaries.
	for key := range cellChart {
		ex, _ := engine.Part[engine.Exits](entities[key])
		delete(ex.Dirs, "up")
		delete(ex.Dirs, "down")
		if ex.Dirs == nil {
			ex.Dirs = map[string]string{}
			for i, p := range entities[key].Parts {
				if _, ok := p.(engine.Exits); ok {
					entities[key].Parts[i] = ex
				}
			}
		}
	}
	if err := content.ValidateVerticalGeometry(c); err != nil {
		report("error", err.Error())
		return result
	}
	link := func(from, dir, to, back string) {
		a, b := entities[from], entities[to]
		if a == nil || b == nil {
			return
		}
		ex, _ := engine.Part[engine.Exits](a)
		ex.Dirs[dir] = to
		ex, _ = engine.Part[engine.Exits](b)
		ex.Dirs[back] = from
	}
	for _, v := range c.VerticalConnections.Connections {
		link(authoredID(v.Kind, v.LowerID), "up", authoredID(v.Kind, v.UpperID), "down")
	}
	for _, v := range c.LocationEntries.Entries {
		link(authoredID("location", v.LocationID), "in", authoredID("room", v.RoomID), "out")
	}
	placements := map[string]content.Content{}
	for _, v := range c.Contents.Contents {
		placements[authoredID(v.EntityKind, v.EntityID)] = v
	}
	placeThing := func(kind, id, name string, parts ...engine.Component) {
		key := authoredID(kind, id)
		p, ok := placements[key]
		parent := entities[authoredID(p.ParentKind, p.ParentID)]
		if !ok || parent == nil {
			report("warning", fmt.Sprintf("Excluded %s %q: no playable cell placement.", kind, name))
			return
		}
		parent.Add(engine.NewEntity(key, name).With(parts...))
	}
	for _, item := range c.WorldItems.WorldItems {
		parts := []engine.Component{engine.Description{Text: item.FullDescription}, engine.ShortDescription{Text: item.ShortDescription}}
		switch item.Kind {
		case content.WorldItemTakeable:
			parts = append(parts, engine.Portable{})
		case content.WorldItemFixed:
			parts = append(parts, engine.Notable{})
		}
		placeThing("world_item", item.ID, item.Name, parts...)
	}
	for _, npc := range c.NPCs.NPCs {
		placeThing("npc", npc.ID, npc.Name, engine.Notable{}, engine.Description{Text: npc.Description})
	}
	for _, terminal := range c.Terminals.Terminals {
		message := "(Placeholder) Terminal functionality is unavailable in this playtest."
		placeThing("terminal", terminal.ID, terminal.HostName, engine.Notable{}, engine.Description{Text: message}, engine.On{Verb: "use", Do: func(*engine.World) string { return message }})
	}
	if len(c.Terminals.Terminals) > 0 || len(c.HostNetworks.HostNetworks) > 0 || len(c.Users.Users) > 0 {
		report("warning", "Terminal filesystems, accounts, and network navigation are not playable in this integration.")
	}
	start := entities[authoredID(c.Play.Start.Kind, c.Play.Start.ID)]
	if start == nil {
		report("error", "Starting cell is not playable.")
		return result
	}
	start.Add(w.Player)
	// Connectivity is advisory: the start and all Hub arrivals are legitimate
	// access points. Do not invent passages to disconnected authored cells.
	reached := map[string]bool{}
	queue := []string{start.ID}
	for _, hub := range hubs.List(w) {
		part, _ := engine.Part[hubs.Hub](hub)
		queue = append(queue, part.Entry)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if reached[id] {
			continue
		}
		reached[id] = true
		if e := entities[id]; e != nil {
			if ex, ok := engine.Part[engine.Exits](e); ok {
				for _, dest := range ex.Dirs {
					queue = append(queue, dest)
				}
			}
		}
	}
	for _, id := range chartIDs {
		cells := chartCells[id]
		keys := make([]string, 0, len(cells))
		for _, key := range cells {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !reached[key] {
				report("warning", fmt.Sprintf("Cell %q is unreachable from the start and Hub arrivals.", entities[key].Name))
			}
		}
	}
	for _, d := range result.Diagnostics {
		if d.Severity == "error" {
			return result
		}
	}
	for _, q := range c.Quests.Quests {
		if !q.Enabled {
			continue
		}
		rq := engine.Quest{ID: q.ID, Name: q.Name, Description: q.Description, CompletionText: q.CompletionText, AutoStart: q.AutoStart}
		for _, s := range q.Steps {
			target := authoredID(s.Target.Kind, s.Target.ID)
			e := w.FindID(target)
			if e == nil {
				report("warning", fmt.Sprintf("Quest %q target %s is excluded from play.", q.Name, target))
			} else {
				cell := e
				if s.Kind == "carry" {
					cell = e.Parent
				}
				if cell == nil || !reached[cell.ID] {
					report("warning", fmt.Sprintf("Quest %q target %s is unreachable.", q.Name, target))
				}
			}
			rq.Steps = append(rq.Steps, engine.QuestStep{ID: s.ID, Text: s.Text, Kind: s.Kind, Target: target})
		}
		w.Quests = append(w.Quests, rq)
	}
	w.EvaluateQuests()
	result.World = w
	return result
}

func (r AuthoredResult) DiagnosticText() string {
	var lines []string
	for _, d := range r.Diagnostics {
		lines = append(lines, d.Severity+": "+d.Message)
	}
	return strings.Join(lines, "\n")
}
