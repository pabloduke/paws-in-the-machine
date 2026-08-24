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
	"time"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:8080", "editor listen address")
	contentDirFlag := flag.String("content-dir", "", "game content directory (defaults to <repo>/internal/game/content)")
	flag.Parse()
	contentDir, err := resolveContentDir(*contentDirFlag)
	if err != nil {
		log.Fatal("editor: ", err)
	}

	server := &http.Server{
		Addr: *addr,
		Handler: newEditorHandler(
			newWorldItemStore(contentDir),
			newHubStore(contentDir),
			newRoomStore(contentDir),
			newNPCStore(contentDir),
			newTerminalStore(contentDir),
			newHostNetworkStore(contentDir),
			newNetworkAssignmentStore(contentDir),
			newUserStore(contentDir),
			newTerminalAccessStore(contentDir),
			newRoomPlacementStore(contentDir),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("game editor: http://%s", *addr)
	log.Printf("game editor content: %s", contentDir)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("editor: ", err)
	}
}
