package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// A world is one content directory: a complete, independent set of the
// authored catalogs. Worlds exist so a designer can try a variation
// without disturbing what already works.
//
// The editor edits exactly one world at a time. Loading another builds
// a fresh handler over that directory and swaps it in, so no store is
// ever pointed at a different directory while it is in use, and no
// existing handler code has to know worlds exist at all.
//
// There is no separate save step: every screen writes its catalog
// immediately. Copy therefore duplicates whatever is on disk right now,
// and loading a world is a deliberate act, never a side effect of
// copying one.

// mainWorld is the repository's own content, always offered so the real
// game content is reachable from the same screen as everything else
// instead of being a directory you have to remember.
const mainWorld = "main"

var errWorldNotFound = errors.New("world not found")
var errWorldExists = errors.New("world already exists")

type worldInfo struct {
	Name    string
	Dir     string
	Current bool
	IsMain  bool
}

type worldRegistry struct {
	mu      sync.Mutex
	root    string // directory holding one subdirectory per world
	mainDir string // the repository's internal/game/content
	current atomic.Pointer[loadedWorld]
}

type loadedWorld struct {
	name    string
	dir     string
	handler http.Handler
}

func newWorldRegistry(root, mainDir string) (*worldRegistry, error) {
	registry := &worldRegistry{root: root, mainDir: mainDir}
	if err := registry.Load(mainWorld); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *worldRegistry) Current() *loadedWorld { return r.current.Load() }

// ServeHTTP forwards to the currently loaded world. A request already
// in flight keeps the handler it started with, so a swap never tears a
// response in half.
func (r *worldRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	current := r.Current()
	if current == nil {
		http.Error(w, "no world is loaded", http.StatusInternalServerError)
		return
	}
	current.handler.ServeHTTP(w, req)
}

// validWorldName keeps a name usable as a single directory entry. This
// is the only thing standing between a submitted name and the
// filesystem, so it is deliberately strict rather than clever.
func validWorldName(name string) bool {
	if name == "" || len(name) > 64 || name == mainWorld {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func (r *worldRegistry) dirFor(name string) (string, error) {
	if name == mainWorld {
		return r.mainDir, nil
	}
	if !validWorldName(name) {
		return "", fmt.Errorf("world names use letters, digits, hyphens, and underscores")
	}
	return filepath.Join(r.root, name), nil
}

// List returns main followed by every world directory, alphabetically.
func (r *worldRegistry) List() ([]worldInfo, error) {
	current := r.Current()
	currentName := ""
	if current != nil {
		currentName = current.name
	}
	worlds := []worldInfo{{
		Name: mainWorld, Dir: r.mainDir, IsMain: true, Current: currentName == mainWorld,
	}}
	entries, err := os.ReadDir(r.root)
	if errors.Is(err, os.ErrNotExist) {
		return worlds, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read worlds directory: %w", err)
	}
	var others []worldInfo
	for _, entry := range entries {
		if !entry.IsDir() || !validWorldName(entry.Name()) {
			continue
		}
		others = append(others, worldInfo{
			Name: entry.Name(), Dir: filepath.Join(r.root, entry.Name()),
			Current: entry.Name() == currentName,
		})
	}
	sort.SliceStable(others, byNameLower(func(i int) string { return others[i].Name }, len(others)))
	return append(worlds, others...), nil
}

func (r *worldRegistry) Load(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dir, err := r.dirFor(name)
	if err != nil {
		return err
	}
	if name != mainWorld {
		if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
			return errWorldNotFound
		}
	}
	r.current.Store(&loadedWorld{name: name, dir: dir, handler: newWorldHandler(dir, r)})
	return nil
}

// Create makes an empty world. Catalogs appear on their first save, so
// an empty directory is a complete, valid, empty world.
func (r *worldRegistry) Create(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dir, err := r.dirFor(name)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		return errWorldExists
	}
	return os.MkdirAll(dir, 0o755)
}

// Copy duplicates a world's catalogs under a new name. Recovery copies
// are not carried over: a .bak belongs to the world that wrote it, and
// copying one would offer a restore point the new world never had.
func (r *worldRegistry) Copy(from, to string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fromDir, err := r.dirFor(from)
	if err != nil {
		return err
	}
	toDir, err := r.dirFor(to)
	if err != nil {
		return err
	}
	if info, statErr := os.Stat(fromDir); statErr != nil || !info.IsDir() {
		return errWorldNotFound
	}
	if _, statErr := os.Stat(toDir); statErr == nil {
		return errWorldExists
	}
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", from, err)
	}
	if err := os.MkdirAll(toDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", to, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		if err := copyFile(filepath.Join(fromDir, name), filepath.Join(toDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("copy %s: %w", filepath.Base(from), err)
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("copy %s: %w", filepath.Base(from), err)
	}
	data, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("copy %s: %w", filepath.Base(from), err)
	}
	return writeAtomic(to, data, info.Mode().Perm())
}
