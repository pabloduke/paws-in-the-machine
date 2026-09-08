package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newWorldsTestRegistry(t *testing.T) (*worldRegistry, string, string) {
	t.Helper()
	root := t.TempDir()
	worldsDir := filepath.Join(root, "worlds")
	mainDir := filepath.Join(root, "content")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry, err := newWorldRegistry(worldsDir, mainDir)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	return registry, worldsDir, mainDir
}

func TestWorldsStartOnMainAndListAlphabetically(t *testing.T) {
	registry, worldsDir, mainDir := newWorldsTestRegistry(t)
	if current := registry.Current(); current == nil || current.name != mainWorld || current.dir != mainDir {
		t.Fatalf("registry did not start on main: %+v", current)
	}
	for _, name := range []string{"zeta", "alpha"} {
		if err := registry.Create(name); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	// A stray file and an unusable directory name are not worlds.
	os.WriteFile(filepath.Join(worldsDir, "notes.txt"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(worldsDir, "has space"), 0o755)

	worlds, err := registry.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var names []string
	for _, world := range worlds {
		names = append(names, world.Name)
	}
	if strings.Join(names, ",") != "main,alpha,zeta" {
		t.Fatalf("worlds = %v, want main first then alphabetical", names)
	}
	if !worlds[0].Current || !worlds[0].IsMain {
		t.Fatalf("main is not marked current: %+v", worlds[0])
	}
}

func TestWorldNamesCannotEscapeTheWorldsDirectory(t *testing.T) {
	registry, _, _ := newWorldsTestRegistry(t)
	for _, name := range []string{
		"", "..", "../escape", "a/b", "with space", "dot.name",
		strings.Repeat("x", 65), mainWorld,
	} {
		if validWorldName(name) {
			t.Errorf("validWorldName(%q) = true, want false", name)
		}
		if err := registry.Create(name); err == nil {
			t.Errorf("Create(%q) was accepted", name)
		}
	}
	// Nothing may have been written outside the worlds directory.
	if _, err := os.Stat(filepath.Join(registry.root, "..", "escape")); err == nil {
		t.Fatal("a world was created outside the worlds directory")
	}
}

func TestCopyDuplicatesCatalogsButNotRecoveryCopies(t *testing.T) {
	registry, worldsDir, mainDir := newWorldsTestRegistry(t)
	if err := os.WriteFile(filepath.Join(mainDir, "rooms.json"), []byte(`{"version":1,"rooms":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A recovery copy belongs to the world that wrote it.
	if err := os.WriteFile(filepath.Join(mainDir, "rooms.json.bak"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := registry.Copy(mainWorld, "variant"); err != nil {
		t.Fatalf("copy: %v", err)
	}
	copied, err := os.ReadFile(filepath.Join(worldsDir, "variant", "rooms.json"))
	if err != nil {
		t.Fatalf("catalog was not copied: %v", err)
	}
	if string(copied) != `{"version":1,"rooms":[]}`+"\n" {
		t.Fatalf("copied catalog differs: %q", copied)
	}
	if _, err := os.Stat(filepath.Join(worldsDir, "variant", "rooms.json.bak")); !os.IsNotExist(err) {
		t.Fatal("a recovery copy was carried into the new world")
	}
	if err := registry.Copy(mainWorld, "variant"); err != errWorldExists {
		t.Fatalf("copying over an existing world = %v, want errWorldExists", err)
	}
	// The source must be untouched.
	if _, err := os.Stat(filepath.Join(mainDir, "rooms.json")); err != nil {
		t.Fatalf("copy disturbed the source world: %v", err)
	}
}

func TestCopiedWorldIsIndependentOfItsSource(t *testing.T) {
	registry, worldsDir, mainDir := newWorldsTestRegistry(t)
	hubs := newHubStore(mainDir)
	if _, err := hubs.Create("Original Hub"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Copy(mainWorld, "variant"); err != nil {
		t.Fatalf("copy: %v", err)
	}
	// Editing the copy must not reach back into main.
	variantHubs := newHubStore(filepath.Join(worldsDir, "variant"))
	if _, err := variantHubs.Create("Variant Only"); err != nil {
		t.Fatal(err)
	}
	mainList, _ := hubs.List()
	if len(mainList) != 1 || mainList[0].Name != "Original Hub" {
		t.Fatalf("editing the copy changed main: %+v", mainList)
	}
	variantList, _ := variantHubs.List()
	if len(variantList) != 2 {
		t.Fatalf("the copy did not inherit main's content: %+v", variantList)
	}
}

func TestLoadingAWorldSwitchesWhatTheEditorEdits(t *testing.T) {
	registry, worldsDir, mainDir := newWorldsTestRegistry(t)
	if _, err := newHubStore(mainDir).Create("Main Hub"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Create("variant"); err != nil {
		t.Fatal(err)
	}
	if _, err := newHubStore(filepath.Join(worldsDir, "variant")).Create("Variant Hub"); err != nil {
		t.Fatal(err)
	}

	body := getRequest(t, registry, "/content/hubs", nil).Body.String()
	if !strings.Contains(body, "Main Hub") || strings.Contains(body, "Variant Hub") {
		t.Fatal("the editor did not start in main")
	}
	if !strings.Contains(body, "world: main") {
		t.Fatal("the header does not say which world is loaded")
	}

	rec := postForm(t, registry, "/worlds/load", url.Values{"name": {"variant"}}, "http://example.com", false)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/content/hubs" {
		t.Fatalf("load = %d %q, want a redirect to Hub selection", rec.Code, rec.Header().Get("Location"))
	}
	body = getRequest(t, registry, "/content/hubs", nil).Body.String()
	if !strings.Contains(body, "Variant Hub") || strings.Contains(body, "Main Hub") {
		t.Fatal("loading a world did not switch what the editor edits")
	}
	if !strings.Contains(body, "world: variant") {
		t.Fatal("the header still names the old world")
	}
}

func TestLoadingAnUnknownWorldIsRefusedAndKeepsTheCurrentOne(t *testing.T) {
	registry, _, _ := newWorldsTestRegistry(t)
	rec := postForm(t, registry, "/worlds/load", url.Values{"name": {"nope"}}, "http://example.com", false)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "There is no world named nope.") {
		t.Fatalf("unknown world = %d %s", rec.Code, rec.Body.String())
	}
	if current := registry.Current(); current == nil || current.name != mainWorld {
		t.Fatalf("a failed load changed the current world: %+v", current)
	}
}

func TestWorldsScreenCopiesAndCreatesThroughHTTP(t *testing.T) {
	registry, worldsDir, _ := newWorldsTestRegistry(t)

	rec := postForm(t, registry, "/worlds/copy",
		url.Values{"from": {mainWorld}, "name": {"variant-a"}}, "http://example.com", false)
	if !strings.Contains(rec.Body.String(), "Copied main to variant-a") {
		t.Fatalf("copy failed: %s", rec.Body.String())
	}
	// Copying does not load: switching worlds stays a deliberate act.
	if current := registry.Current(); current.name != mainWorld {
		t.Fatalf("copy switched worlds: %+v", current)
	}
	if _, err := os.Stat(filepath.Join(worldsDir, "variant-a")); err != nil {
		t.Fatalf("copy did not create the directory: %v", err)
	}

	rec = postForm(t, registry, "/worlds/new", url.Values{"name": {"sketch"}}, "http://example.com", false)
	if !strings.Contains(rec.Body.String(), "Created the empty world sketch") {
		t.Fatalf("create failed: %s", rec.Body.String())
	}

	rec = postForm(t, registry, "/worlds/new", url.Values{"name": {"bad name"}}, "http://example.com", false)
	if !strings.Contains(rec.Body.String(), "Use letters, digits, hyphens, and underscores") {
		t.Fatalf("invalid name accepted: %s", rec.Body.String())
	}
	rec = postForm(t, registry, "/worlds/new", url.Values{"name": {mainWorld}}, "http://example.com", false)
	if !strings.Contains(rec.Body.String(), "cannot be reused as a name") {
		t.Fatalf("main was accepted as a new world name: %s", rec.Body.String())
	}
}

func TestWorldWritesRequireSameOrigin(t *testing.T) {
	registry, _, _ := newWorldsTestRegistry(t)
	rec := postForm(t, registry, "/worlds/copy",
		url.Values{"from": {mainWorld}, "name": {"variant"}}, "", false)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin world write = %d, want 403", rec.Code)
	}
}

// A handler built for one directory has no registry and must say so
// rather than offering controls that cannot work.
func TestSingleDirectoryEditorReportsWorldsUnavailable(t *testing.T) {
	handler, _ := testEditorHandler(t)
	body := getRequest(t, handler, "/worlds", nil).Body.String()
	if !strings.Contains(body, "nothing to switch between") {
		t.Fatalf("single-directory editor did not explain: %s", body)
	}
	if strings.Contains(body, "world: ") {
		t.Fatal("a single-directory editor should not claim a world name")
	}
}
