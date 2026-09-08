package main

import (
	"fmt"
	"github.com/pabloduke/paws-in-the-machine/internal/content"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"path/filepath"
)

func (h *editorHandler) verticalStore() *relationStore[content.VerticalConnection] {
	return &relationStore[content.VerticalConnection]{path: filepath.Join(filepath.Dir(h.rooms.path), "vertical_connections.json"), label: "vertical connections", key: func(v content.VerticalConnection) string { return v.Kind + ":" + v.LowerID }, decode: func(b []byte) ([]content.VerticalConnection, error) {
		f, e := content.DecodeVerticalConnections(b)
		return f.Connections, e
	}, encode: func(v []content.VerticalConnection) ([]byte, error) {
		return content.EncodeVerticalConnections(content.VerticalConnectionsFile{Version: 1, Connections: v})
	}}
}
func (h *editorHandler) guardVertical(kind, id string) error {
	vs, err := h.verticalStore().List()
	if err != nil {
		return err
	}
	for _, v := range vs {
		if v.Kind == kind && (v.LowerID == id || v.UpperID == id) {
			return fmt.Errorf("Remove this cell's vertical connections before moving, unplacing, or deleting it.")
		}
	}
	return nil
}
func (h *editorHandler) connectVertical(kind, lower, upper string) error {
	c, err := game.ReadCatalogs(filepath.Dir(h.rooms.path))
	if err != nil {
		return err
	}
	v := content.VerticalConnection{Kind: kind, LowerID: lower, UpperID: upper}
	c.VerticalConnections.Connections = append(c.VerticalConnections.Connections, v)
	if err = content.ValidateVerticalGeometry(c); err != nil {
		return err
	}
	_, err = h.verticalStore().Put(v)
	return err
}
