// Package hacking is the in-game hacking terminal: the shell Buddy
// works in when he logs into his deck. He never goes anywhere — he's a
// cat at a terminal, connected to the deck (his powerful box) and
// sshing outward from there, the way a person sits at a laptop logged
// into a bigger machine. Hosts, filesystems, processes, and the
// commands that poke them (ls, cd, cat, grep, cp, ssh, curl, ps, kill,
// run) are all game fiction over in-memory content declared by the
// game package — the resemblance to real tools ends at the prompt.
// Implementation note: the package imports only the engine and stdlib
// string helpers; hooks on files and processes set world flags, which
// is how a hack advances the story.
package hacking

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Node is one entry in a fake filesystem: a directory (with children)
// or a file (with text). Hooks are flag names set on the world when
// the player interacts; empty means no hook.
type Node struct {
	Name     string
	Dir      bool
	Text     string  // file contents: story prose, fake configs, clues
	Children []*Node // directory entries
	RunText  string  // non-empty marks the file executable via `run`
	OnRead   string  // flag set when cat'ed or grep-matched
	OnCopy   string  // flag set when copied onto the deck
	OnRun    string  // flag set when run
	Copied   bool    // placed by cp — feeds the DISCOVERIES panel
}

// File and Dir are content-authoring helpers.
func File(name, text string) *Node { return &Node{Name: name, Text: text} }
func Dir(name string, children ...*Node) *Node {
	return &Node{Name: name, Dir: true, Children: children}
}

func (n *Node) child(name string) *Node {
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// clone deep-copies a node so cp never aliases content trees.
func (n *Node) clone() *Node {
	out := *n
	out.Children = nil
	for _, c := range n.Children {
		out.Children = append(out.Children, c.clone())
	}
	return &out
}

// Process is a fake process table entry on a fake host.
type Process struct {
	PID     int
	Name    string
	Stopped bool
	OnKill  string // flag set when killed
}

// Host is one fake system on the content-declared net. The deck
// itself is a host; remote ones are reached with the ssh alias.
type Host struct {
	Name   string
	Root   *Node             // fake filesystem root (a Dir)
	Home   string            // path of the shell's home dir, e.g. "/home/paws_in_the_machine"
	Banner string            // printed on connect
	Procs  []*Process        // fake process table
	Served map[string]string // fake curl resources: path -> body ("/" for the bare host)
}

// conn is one held connection on the ssh stack.
type conn struct {
	host *Host
	cwd  []string
}

// Session is the state behind the terminal screen: which fake host
// the terminal shows, the working directory, and the connection stack.
// Buddy never moves; the session is the thing that travels.
type Session struct {
	w     *engine.World
	net   map[string]*Host
	deck  *Host
	host  *Host
	cwd   []string
	stack []conn
}

// NewSession opens the shell on the deck's local host.
func NewSession(w *engine.World, net map[string]*Host, deckHost string) (*Session, error) {
	deck, ok := net[deckHost]
	if !ok {
		return nil, fmt.Errorf("(bug) deck shell net has no host %q", deckHost)
	}
	return &Session{
		w:    w,
		net:  net,
		deck: deck,
		host: deck,
		cwd:  splitPath(deck.Home),
	}, nil
}

// HostName is the current fake host, for the terminal title bar.
func (s *Session) HostName() string { return s.host.Name }

// Path is the current fake working directory, for the quest panel.
func (s *Session) Path() string { return "/" + strings.Join(s.cwd, "/") }

// login is Buddy's handle — the name the net knows him by.
const login = "paws_in_the_machine"

// Prompt renders the shell prompt, home shown as ~ on the deck.
func (s *Session) Prompt() string {
	path := s.Path()
	if s.host == s.deck {
		if rest, ok := strings.CutPrefix(path, s.deck.Home); ok {
			path = "~" + rest
		}
	}
	return login + "@" + s.host.Name + ":" + path + " $ "
}

// Discoveries lists files that have been copied onto the deck.
func (s *Session) Discoveries() []string {
	var out []string
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Copied && !n.Dir {
			out = append(out, n.Name)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(s.deck.Root)
	sort.Strings(out)
	return out
}

// Exec runs one typed line against the session. done reports that the
// session ended — `exit`/`logout` popped all the way back off the deck.
func (s *Session) Exec(line string) (out string, done bool) {
	args := strings.Fields(line)
	if len(args) == 0 {
		return "", false
	}
	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "ls":
		return s.ls(args), false
	case "cd":
		return s.cd(args), false
	case "pwd":
		return s.Path(), false
	case "cat":
		return s.cat(args), false
	case "grep":
		return s.grep(args), false
	case "cp":
		return s.cp(args), false
	case "mkdir":
		return s.mkdir(args), false
	case "touch":
		return s.touch(args), false
	case "ssh":
		return s.ssh(args), false
	case "exit", "logout":
		return s.exit()
	case "curl":
		return s.curl(args), false
	case "ps":
		return s.ps(), false
	case "kill":
		return s.kill(args), false
	case "run":
		return s.run(args), false
	case "help":
		return helpText, false
	}
	return cmd + ": command not found", false
}

// End reports the narration the UI shows when the terminal is closed
// outright (the Esc affordance), from wherever the session is.
func (s *Session) End() string {
	if s.host != s.deck {
		return "Connection to " + s.host.Name + " closed by local host."
	}
	return "logout"
}

// --- path plumbing ----------------------------------------------------

func splitPath(p string) []string {
	var out []string
	for _, seg := range strings.Split(p, "/") {
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}

// normalize resolves "." and ".." (clamped at root).
func normalize(segs []string) []string {
	var out []string
	for _, seg := range segs {
		switch seg {
		case ".":
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, seg)
		}
	}
	return out
}

// locate resolves a typed path to a fake host and path segments.
// `~` always addresses the deck's home, from any host — that is how
// cp works across the net without extra tooling.
func (s *Session) locate(p string) (*Host, []string) {
	switch {
	case p == "~":
		return s.deck, splitPath(s.deck.Home)
	case strings.HasPrefix(p, "~/"):
		return s.deck, normalize(append(splitPath(s.deck.Home), splitPath(p[2:])...))
	case strings.HasPrefix(p, "/"):
		return s.host, normalize(splitPath(p))
	default:
		return s.host, normalize(append(append([]string{}, s.cwd...), splitPath(p)...))
	}
}

// find walks a fake filesystem to a node; nil when absent.
func find(root *Node, segs []string) *Node {
	n := root
	for _, seg := range segs {
		if n == nil || !n.Dir {
			return nil
		}
		n = n.child(seg)
	}
	return n
}

func (s *Session) node(p string) *Node {
	host, segs := s.locate(p)
	return find(host.Root, segs)
}

// --- fake file commands -----------------------------------------------

func (s *Session) ls(args []string) string {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}
	n := s.node(target)
	if n == nil {
		return "ls: cannot access '" + target + "': No such file or directory"
	}
	if !n.Dir {
		return n.Name
	}
	if len(n.Children) == 0 {
		return ""
	}
	names := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		name := c.Name
		if c.Dir {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, "  ")
}

func (s *Session) cd(args []string) string {
	if len(args) == 0 {
		s.cwd = splitPath(s.host.Home)
		return ""
	}
	host, segs := s.locate(args[0])
	if host != s.host {
		return "cd: " + args[0] + ": is on " + host.Name + " (try ssh)"
	}
	n := find(host.Root, segs)
	if n == nil {
		return "cd: " + args[0] + ": No such file or directory"
	}
	if !n.Dir {
		return "cd: " + args[0] + ": Not a directory"
	}
	s.cwd = segs
	return ""
}

// read fires a file's OnRead hook and returns its text.
func (s *Session) read(n *Node) string {
	s.setFlag(n.OnRead)
	return n.Text
}

func (s *Session) cat(args []string) string {
	if len(args) == 0 {
		return "usage: cat <file>"
	}
	var out []string
	for _, arg := range args {
		n := s.node(arg)
		switch {
		case n == nil:
			out = append(out, "cat: "+arg+": No such file or directory")
		case n.Dir:
			out = append(out, "cat: "+arg+": Is a directory")
		default:
			out = append(out, s.read(n))
		}
	}
	return strings.Join(out, "\n")
}

func (s *Session) grep(args []string) string {
	if len(args) < 2 {
		return "usage: grep <pattern> <file...>"
	}
	pat := args[0]
	files := args[1:]
	var out []string
	for _, arg := range files {
		n := s.node(arg)
		switch {
		case n == nil:
			out = append(out, "grep: "+arg+": No such file or directory")
			continue
		case n.Dir:
			out = append(out, "grep: "+arg+": Is a directory")
			continue
		}
		for _, ln := range strings.Split(n.Text, "\n") {
			if strings.Contains(strings.ToLower(ln), strings.ToLower(pat)) {
				s.setFlag(n.OnRead) // a matched line counts as read
				if len(files) > 1 {
					out = append(out, arg+":"+ln)
				} else {
					out = append(out, ln)
				}
			}
		}
	}
	return strings.Join(out, "\n") // like the real thing: silent when nothing matches
}

func (s *Session) cp(args []string) string {
	if len(args) != 2 {
		return "usage: cp <src> <dst>"
	}
	src := s.node(args[0])
	switch {
	case src == nil:
		return "cp: cannot stat '" + args[0] + "': No such file or directory"
	case src.Dir:
		return "cp: -r not supported on this deck"
	}

	dstHost, dstSegs := s.locate(args[1])
	placed := src.clone()
	if dst := find(dstHost.Root, dstSegs); dst != nil && dst.Dir {
		// copy into the directory under the source name
		replaceChild(dst, placed)
	} else {
		if len(dstSegs) == 0 {
			return "cp: cannot create '" + args[1] + "': Is a directory"
		}
		parent := find(dstHost.Root, dstSegs[:len(dstSegs)-1])
		if parent == nil || !parent.Dir {
			return "cp: cannot create '" + args[1] + "': No such file or directory"
		}
		placed.Name = dstSegs[len(dstSegs)-1]
		replaceChild(parent, placed)
	}
	if dstHost == s.deck {
		placed.Copied = true
		s.setFlag(src.OnCopy)
	}
	return "" // like the real thing: silent on success
}

func (s *Session) mkdir(args []string) string {
	if len(args) == 0 {
		return "usage: mkdir <dir...>"
	}
	var out []string
	for _, arg := range args {
		host, segs := s.locate(arg)
		if len(segs) == 0 {
			out = append(out, "mkdir: cannot create directory '"+arg+"': File exists")
			continue
		}
		if find(host.Root, segs) != nil {
			out = append(out, "mkdir: cannot create directory '"+arg+"': File exists")
			continue
		}
		parent := find(host.Root, segs[:len(segs)-1])
		if parent == nil || !parent.Dir {
			out = append(out, "mkdir: cannot create directory '"+arg+"': No such file or directory")
			continue
		}
		replaceChild(parent, Dir(segs[len(segs)-1]))
	}
	return strings.Join(out, "\n")
}

func (s *Session) touch(args []string) string {
	if len(args) == 0 {
		return "usage: touch <file...>"
	}
	var out []string
	for _, arg := range args {
		host, segs := s.locate(arg)
		if len(segs) == 0 {
			out = append(out, "touch: cannot touch '"+arg+"': Is a directory")
			continue
		}
		if existing := find(host.Root, segs); existing != nil {
			if existing.Dir {
				out = append(out, "touch: cannot touch '"+arg+"': Is a directory")
			}
			continue
		}
		parent := find(host.Root, segs[:len(segs)-1])
		if parent == nil || !parent.Dir {
			out = append(out, "touch: cannot touch '"+arg+"': No such file or directory")
			continue
		}
		replaceChild(parent, File(segs[len(segs)-1], ""))
	}
	return strings.Join(out, "\n")
}

// replaceChild inserts c into dir, overwriting a same-named entry.
func replaceChild(dir *Node, c *Node) {
	for i, existing := range dir.Children {
		if existing.Name == c.Name {
			dir.Children[i] = c
			return
		}
	}
	dir.Children = append(dir.Children, c)
}

// --- fake net commands ------------------------------------------------

func (s *Session) ssh(args []string) string {
	if len(args) == 0 {
		return "usage: ssh <host>"
	}
	host, ok := s.net[args[0]]
	if !ok {
		return "ssh: Could not resolve hostname " + args[0]
	}
	if host == s.host {
		return "already connected to " + host.Name
	}
	s.stack = append(s.stack, conn{host: s.host, cwd: s.cwd})
	s.host = host
	s.cwd = splitPath(host.Home)
	if host.Banner != "" {
		return host.Banner
	}
	return ""
}

func (s *Session) exit() (string, bool) {
	if len(s.stack) == 0 {
		return "logout", true // closing the deck session drops back to the room
	}
	closed := s.host.Name
	top := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	s.host, s.cwd = top.host, top.cwd
	return "Connection to " + closed + " closed.", false
}

func (s *Session) curl(args []string) string {
	if len(args) == 0 {
		return "usage: curl <host>[/path]"
	}
	name, path, hasPath := strings.Cut(args[0], "/")
	host, ok := s.net[name]
	if !ok || host.Served == nil {
		return "curl: (6) Could not resolve host: " + name
	}
	key := "/"
	if hasPath && path != "" {
		key = "/" + path
	}
	if body, ok := host.Served[key]; ok {
		return body
	}
	return "curl: (22) The requested URL returned error: 404"
}

// --- fake process commands --------------------------------------------

func (s *Session) ps() string {
	var b strings.Builder
	b.WriteString("  PID CMD          STAT")
	for _, p := range s.host.Procs {
		stat := "S"
		if p.Stopped {
			stat = "Z"
		}
		b.WriteString(fmt.Sprintf("\n%5d %-12s %s", p.PID, p.Name, stat))
	}
	return b.String()
}

func (s *Session) kill(args []string) string {
	if len(args) == 0 {
		return "usage: kill <pid>"
	}
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return "kill: " + args[0] + ": arguments must be process ids"
	}
	for _, p := range s.host.Procs {
		if p.PID == pid {
			if p.Stopped {
				return "kill: (" + args[0] + ") - No such process"
			}
			p.Stopped = true
			s.setFlag(p.OnKill)
			return "" // like the real thing: silent; ps shows the corpse
		}
	}
	return "kill: (" + args[0] + ") - No such process"
}

func (s *Session) run(args []string) string {
	if len(args) == 0 {
		return "usage: run <file>"
	}
	n := s.node(args[0])
	switch {
	case n == nil:
		return "run: " + args[0] + ": No such file or directory"
	case n.Dir || n.RunText == "":
		return "run: " + args[0] + ": Permission denied"
	}
	s.setFlag(n.OnRun)
	return n.RunText
}

// setFlag applies a hook: a game flag and nothing else.
func (s *Session) setFlag(name string) {
	if name != "" {
		s.w.Flags[name] = true
	}
}

const helpText = `deck shell:
  ls [path]           list directory
  cd <path> / pwd     move around / where am I
  cat <file>          read a file
  grep <pat> <files>  search file lines
  cp <src> <dst>      copy (~ is always the deck's home)
  mkdir <dir>         create fake directories
  touch <file>        create empty fake files
  ssh <host>          connect to a host
  curl <host>[/path]  poke a host without logging in
  ps / kill <pid>     list / stop processes
  run <file>          execute something
  exit / logout       close the connection (or the deck, to leave)`

// Deck is the data-only marker component (the dialogue.Talkable
// pattern) attached to the deck entity: using it opens the Deck Shell
// screen. Engine dispatch declines every verb — the UI owns the
// interactive session.
type Deck struct {
	Net        map[string]*Host
	Host       string // local host name on the net
	Objectives []Objective
}

// Handle declines every command; see the type comment.
func (Deck) Handle(*engine.World, *engine.Entity, engine.Command) (string, bool) {
	return "", false
}

// Objective is one quest-panel hint: Text shows while Flag is unset.
type Objective struct {
	Flag string
	Text string
}

// DecksInScope returns entities the player can currently reach that
// carry a Deck — the terminals Buddy can log into from where he is.
// Mirrors hubs.List, but scope-based: you must be at the deck.
func DecksInScope(w *engine.World) []*engine.Entity {
	var out []*engine.Entity
	for _, e := range w.Visible() {
		if _, ok := engine.Part[Deck](e); ok {
			out = append(out, e)
		}
	}
	return out
}

// CurrentObjective picks the first unmet hint, or a placeholder.
func (d Deck) CurrentObjective(w *engine.World) string {
	for _, o := range d.Objectives {
		if !w.Flags[o.Flag] {
			return o.Text
		}
	}
	return "signal searching..."
}
