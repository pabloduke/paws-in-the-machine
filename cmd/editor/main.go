// Command editor serves the local browser-based game editor.
//
// Run it from the repository root:
//
//	go run ./cmd/editor
package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:8080", "editor listen address")
	contentDirFlag := flag.String("content-dir", "", "edit one content directory and disable world switching")
	worldsDirFlag := flag.String("worlds-dir", "", "directory holding one subdirectory per world (defaults to <repo>/worlds)")
	flag.Parse()
	contentDir, err := resolveContentDir(*contentDirFlag)
	if err != nil {
		log.Fatal("editor: ", err)
	}

	// -content-dir pins the editor to exactly one directory, which is
	// what tests, scripts, and one-off inspections want. Without it the
	// editor manages worlds, with the repository's own content offered
	// as "main".
	var handler http.Handler
	if *contentDirFlag != "" {
		handler = newWorldHandler(contentDir, nil)
		log.Printf("game editor content: %s (single directory; world switching disabled)", contentDir)
	} else {
		worldsDir := *worldsDirFlag
		if worldsDir == "" {
			worldsDir = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(contentDir))), "worlds")
		}
		worldsDir, err = filepath.Abs(worldsDir)
		if err != nil {
			log.Fatal("editor: ", err)
		}
		registry, regErr := newWorldRegistry(worldsDir, contentDir)
		if regErr != nil {
			log.Fatal("editor: ", regErr)
		}
		handler = registry
		log.Printf("game editor worlds: %s", worldsDir)
		log.Printf("game editor content: %s (world \"main\")", contentDir)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("game editor: http://%s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("editor: ", err)
	}
}
