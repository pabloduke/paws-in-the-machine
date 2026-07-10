package hacking

import (
	"sort"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// PDA is the data-only marker component for Buddy's carried pocket
// slab: a read-only mirror of the deck's storage plus a passive port
// sniffer. It is deliberately not a Deck — nothing can be hacked,
// written, or logged into from it; the UI shows a menu, never a
// shell. Seeing is portable; touching happens at home.
type PDA struct {
	Net  map[string]*Host
	Host string // the deck host whose files the PDA mirrors
}

// Handle declines every command; the UI owns the menu (the
// dialogue.Talkable pattern).
func (PDA) Handle(*engine.World, *engine.Entity, engine.Command) (string, bool) {
	return "", false
}

// PDAsInScope returns entities the player can currently reach that
// carry a PDA — mirrors DecksInScope (visible plus carried), so the
// UI can offer the slab wherever Buddy is.
func PDAsInScope(w *engine.World) []*engine.Entity {
	var out []*engine.Entity
	seen := map[*engine.Entity]bool{}
	add := func(e *engine.Entity) {
		if seen[e] {
			return
		}
		if _, ok := engine.Part[PDA](e); ok {
			seen[e] = true
			out = append(out, e)
		}
	}
	for _, e := range w.Visible() {
		add(e)
	}
	for _, e := range w.Player.Contents {
		e.Walk(func(c *engine.Entity) bool {
			add(c)
			return true
		})
	}
	return out
}

// TextFile is one readable document on the mirrored host: a display
// path plus the node behind it. Listing never fires hooks; Read does.
type TextFile struct {
	Path string
	Kind DocumentKind

	node *Node
}

// Read fires the file's OnRead hook (exactly like cat) and returns
// the document for the reader.
func (f TextFile) Read(w *engine.World) Document {
	if f.node.OnRead != "" {
		w.Flags[f.node.OnRead] = true
	}
	text := f.node.Text
	if f.node.TextFn != nil {
		text = f.node.TextFn(w)
	}
	return Document{Path: f.Path, Text: text, Kind: f.Kind}
}

// TextFiles lists every .md/.txt on the host, sorted by path, with
// the host's home shown as ~ the way the shell prints it. Flag-gated
// files (PresentWhen) are skipped until their flag lands, so a mission
// file mirrors onto the PDA exactly when it appears on the deck.
func TextFiles(w *engine.World, host *Host) []TextFile {
	var out []TextFile
	var walk func(n *Node, path string)
	walk = func(n *Node, path string) {
		for _, c := range n.Children {
			if !present(w, c) {
				continue
			}
			p := path + "/" + c.Name
			if c.Dir {
				walk(c, p)
				continue
			}
			if kind, ok := documentKind(c.Name); ok {
				out = append(out, TextFile{Path: homePath(host, p), Kind: kind, node: c})
			}
		}
	}
	walk(host.Root, "")
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// homePath abbreviates a host-absolute path under the host's home
// directory to ~, matching the shell's own display.
func homePath(host *Host, path string) string {
	if rest, ok := strings.CutPrefix(path, host.Home); ok {
		if rest == "" {
			return "~"
		}
		return "~" + rest
	}
	return path
}

// KnownHosts lists the scannable hosts on the net — everything except
// the local host — sorted by name.
func KnownHosts(net map[string]*Host, localHost string) []string {
	var out []string
	for name := range net {
		if name != localHost {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// PortReport is the scan output for a host, sessionless: what the
// PDA's sniffer shows. Identical to the shell's `scan <host>`.
func PortReport(w *engine.World, net map[string]*Host, hostName string) string {
	host, ok := net[hostName]
	if !ok {
		return "scan: Could not resolve hostname " + hostName
	}
	return portReport(w, host)
}
