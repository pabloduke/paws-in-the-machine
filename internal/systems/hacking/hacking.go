// Package hacking is the in-game hacking terminal: the shell Buddy
// works in when he logs into his deck. He never goes anywhere — he's a
// cat at a terminal, connected to the deck (his powerful box) and
// sshing outward from there, the way a person sits at a laptop logged
// into a bigger machine. Hosts, filesystems, processes, and the
// commands that poke them (ls, cd, cat, grep, cp, scan, ssh, curl, ps, kill,
// run) are all game fiction over in-memory content declared by the
// game package — the resemblance to real tools ends at the prompt.
// Implementation note: hooks on files and processes set world flags,
// which is how a hack advances the story. Markdown notes render through
// Glamour so the in-game cat output can stay readable without giving the
// filesystem real access.
package hacking

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	glamansi "github.com/charmbracelet/glamour/ansi"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

var (
	boolTrue = true

	markdownStyle = glamansi.StyleConfig{
		Heading: glamansi.StyleBlock{
			StylePrimitive: glamansi.StylePrimitive{BlockSuffix: "\n"},
		},
		H1: glamansi.StyleBlock{
			StylePrimitive: glamansi.StylePrimitive{
				BlockPrefix: "// ",
				Bold:        &boolTrue,
				Upper:       &boolTrue,
			},
		},
		H2: glamansi.StyleBlock{
			StylePrimitive: glamansi.StylePrimitive{
				BlockPrefix: "## ",
				Bold:        &boolTrue,
			},
		},
		H3: glamansi.StyleBlock{
			StylePrimitive: glamansi.StylePrimitive{
				BlockPrefix: "### ",
				Bold:        &boolTrue,
			},
		},
		Emph: glamansi.StylePrimitive{
			BlockPrefix: "*",
			BlockSuffix: "*",
			Italic:      &boolTrue,
		},
		Strong: glamansi.StylePrimitive{
			BlockPrefix: "**",
			BlockSuffix: "**",
			Bold:        &boolTrue,
		},
		List: glamansi.StyleList{
			LevelIndent: 2,
		},
		Item: glamansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: glamansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: glamansi.StyleTask{
			Ticked:   "[x] ",
			Unticked: "[ ] ",
		},
		Code: glamansi.StyleBlock{
			StylePrimitive: glamansi.StylePrimitive{
				BlockPrefix: "`",
				BlockSuffix: "`",
			},
		},
		BlockQuote: glamansi.StyleBlock{
			Indent:      uintPtr(1),
			IndentToken: stringPtr("| "),
		},
		HorizontalRule: glamansi.StylePrimitive{
			Format: "--------",
		},
	}
)

// Node is one entry in a fake filesystem: a directory (with children)
// or a file (with text). Hooks are flag names set on the world when
// the player interacts; empty means no hook.
type Node struct {
	Name     string
	Dir      bool
	Text     string  // file contents: story prose, fake configs, clues
	Children []*Node // directory entries
	TextFn   func(*engine.World) string
	RunText  string // non-empty marks the file executable via `run`
	OnRead   string // flag set when cat'ed or grep-matched
	OnCopy   string // flag set when copied onto the deck
	OnRun    string // flag set when run
	OnSend   string // flag set when sent to the contact's drop
	// PresentWhen / AbsentWhen gate the node's very existence on world
	// flags: it is absent from ls/cat/grep and the PDA mirror until
	// PresentWhen is set, and again once AbsentWhen is set. This is how
	// a mission file "arrives" in ~/notes when its handler's briefing
	// grants it — and how a bootstrap note (mission 0) vanishes once
	// it's served its purpose (docs/systems/hacking.md).
	PresentWhen string
	AbsentWhen  string
	Copied      bool // placed by cp — marks a deck-side discovery
	Touched     bool // created by touch — user-owned, no edit backup needed
	BackedUp    bool // edit backup already created
}

// File and Dir are content-authoring helpers.
func File(name, text string) *Node { return &Node{Name: name, Text: text} }
func DynamicFile(name string, fn func(*engine.World) string) *Node {
	return &Node{Name: name, TextFn: fn}
}
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

const (
	ProtocolSSH    = "ssh"
	ProtocolFTP    = "ftp"
	ProtocolTelnet = "telnet"
	ProtocolHTTP   = "http"

	// Port states are binary on purpose (user ruling 2026-07-09): a
	// port is open or closed, no `filtered` — one less word between a
	// newb and the puzzle. Hidden is authoring-only: a hidden standard
	// port scans as closed (the perfect disguise), a hidden extra port
	// is omitted.
	StateOpen   = "open"
	StateClosed = "closed"
	StateHidden = "hidden"
)

// Service is one configured port on a fake host. OpenWhen is a story
// flag that opens an otherwise-closed service.
type Service struct {
	Port     int
	Protocol string
	State    string
	OpenWhen string
	Password string
}

// Host is one fake system on the content-declared net. The deck
// itself is a host; remote ones are reached with the ssh alias.
type Host struct {
	Name     string
	Username string            // fake shell account; non-empty requires username@host for SSH
	Root     *Node             // fake filesystem root (a Dir)
	Home     string            // path of the shell's home dir, e.g. "/home/paws_in_the_machine"
	Banner   string            // printed on connect
	Password string            // fake password required before connecting; empty means open
	Require  string            // fake network route flag required before connecting
	Procs    []*Process        // fake process table
	Served   map[string]string // fake curl resources: path -> body ("/" for the bare host)
	Services []*Service        // fake ports exposed by the host
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
	w              *engine.World
	net            map[string]*Host
	deck           *Host
	host           *Host
	cwd            []string
	stack          []conn
	pending        *Host
	pendingService *Service
	pendingUser    string
}

// ExecResult is the structured result of a fake shell command.
type ExecResult struct {
	Output    string
	Done      bool
	Document  *Document
	Edit      *EditBuffer
	Messenger bool // the messenger command: UI toggles the panel
}

// Document is a text file read by cat that the UI can render in
// the terminal reader panel.
type Document struct {
	Path string
	Text string
	Kind DocumentKind
}

type DocumentKind string

const (
	DocumentMarkdown DocumentKind = "markdown"
	DocumentText     DocumentKind = "text"
)

// EditBuffer is a text file opened by edit for the UI editor panel.
// Vim marks a buffer opened as vi/vim/nvim: same editor, but the UI
// runs it modal (normal/insert, :w/:q/:wq) instead of autosave.
type EditBuffer struct {
	Path string
	Text string
	Vim  bool
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

// IsRemote reports whether the active shell is connected beyond the deck.
func (s *Session) IsRemote() bool { return s.host != s.deck }

// Path is the current fake working directory, for the terminal status panel.
func (s *Session) Path() string { return "/" + strings.Join(s.cwd, "/") }

// login is the compatibility fallback for content without a declared deck user.
const login = "paws_in_the_machine"

// Prompt renders the shell prompt, home shown as ~ on the deck.
func (s *Session) Prompt() string {
	if s.pending != nil {
		return "password: "
	}
	path := s.Path()
	if s.host == s.deck {
		if rest, ok := strings.CutPrefix(path, s.deck.Home); ok {
			path = "~" + rest
		}
	}
	username := s.host.Username
	if username == "" {
		username = s.deck.Username
	}
	if username == "" {
		username = login
	}
	return username + "@" + s.host.Name + ":" + path + " $ "
}

// AwaitingPassword reports whether the next input line is a credential,
// allowing the UI to mask it and keep it out of command history.
func (s *Session) AwaitingPassword() bool { return s.pending != nil }

var shellCommands = []string{
	"cat", "cd", "cp", "curl", "edit", "exit", "grep", "help", "kill",
	"logout", "ls", "messenger", "mkdir", "nvim", "ps", "pwd", "run",
	"scan", "send", "ssh", "touch", "vi", "vim",
}

// Complete expands the active token in line and returns every matching token.
// The UI inserts the longest common prefix and can display ambiguous matches.
func (s *Session) Complete(line string) (string, []string) {
	if s.AwaitingPassword() {
		return line, nil
	}
	cut := strings.LastIndexAny(line, " \t")
	before, active := "", line
	if cut >= 0 {
		before, active = line[:cut+1], line[cut+1:]
	}
	prior := strings.Fields(before)
	var matches []string
	if len(prior) == 0 {
		matches = s.commandMatches(active)
	} else {
		expanded := s.expandAliases([]string{prior[0]})
		cmd := prior[0]
		if len(expanded) > 0 {
			cmd = expanded[0]
		}
		argIndex := len(prior) - 1
		switch cmd {
		case "scan", "curl":
			if argIndex == 0 {
				matches = s.hostMatches(active)
			}
		case "ssh":
			if argIndex == 0 {
				matches = s.sshTargetMatches(active)
			}
		case "grep":
			// Options and the search pattern come before path operands.
			nonOptions := 0
			for _, arg := range prior[1:] {
				if !strings.HasPrefix(arg, "-") {
					nonOptions++
				}
			}
			if nonOptions >= 1 {
				matches = s.pathMatches(active)
			}
		case "ls", "cd", "cat", "edit", "vi", "vim", "nvim", "cp",
			"send", "mkdir", "touch", "run":
			matches = s.pathMatches(active)
		}
	}
	if len(matches) == 0 {
		return line, nil
	}
	completed := commonPrefix(matches)
	if len(matches) == 1 && !strings.HasSuffix(completed, "/") {
		completed += " "
	}
	return before + completed, matches
}

func (s *Session) commandMatches(prefix string) []string {
	seen := map[string]bool{}
	var out []string
	for _, cmd := range shellCommands {
		if strings.HasPrefix(cmd, prefix) && !seen[cmd] {
			seen[cmd] = true
			out = append(out, cmd)
		}
	}
	for alias := range s.aliasTable() {
		if strings.HasPrefix(alias, prefix) && !seen[alias] {
			seen[alias] = true
			out = append(out, alias)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Session) hostMatches(prefix string) []string {
	var out []string
	for name := range s.net {
		if strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Session) sshTargetMatches(prefix string) []string {
	var out []string
	for name, host := range s.net {
		target := name
		if host.Username != "" {
			target = host.Username + "@" + name
		}
		if strings.HasPrefix(target, prefix) {
			out = append(out, target)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Session) pathMatches(token string) []string {
	dirText, base := "", token
	if slash := strings.LastIndex(token, "/"); slash >= 0 {
		dirText, base = token[:slash+1], token[slash+1:]
	}
	lookup := strings.TrimSuffix(dirText, "/")
	if lookup == "" {
		if dirText == "/" {
			lookup = "/"
		} else {
			lookup = "."
		}
	}
	n := s.node(lookup)
	if n == nil || !n.Dir {
		return nil
	}
	var out []string
	for _, child := range n.Children {
		if !present(s.w, child) || !strings.HasPrefix(child.Name, base) {
			continue
		}
		if strings.HasPrefix(child.Name, ".") && !strings.HasPrefix(base, ".") {
			continue
		}
		match := dirText + child.Name
		if child.Dir {
			match += "/"
		}
		out = append(out, match)
	}
	sort.Strings(out)
	return out
}

func commonPrefix(values []string) string {
	prefix := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
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
	result := s.ExecDetailed(line)
	return result.Output, result.Done
}

// ExecDetailed runs one typed line and returns output plus optional UI
// metadata, such as a Markdown document read by cat.
func (s *Session) ExecDetailed(line string) ExecResult {
	if s.pending != nil {
		return ExecResult{Output: s.password(line)}
	}
	if out, handled := s.w.DevCommand(line); handled {
		return ExecResult{Output: out}
	}
	args := strings.Fields(line)
	if len(args) == 0 {
		return ExecResult{}
	}
	args = s.expandAliases(args)
	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "ls":
		return ExecResult{Output: s.ls(args)}
	case "cd":
		return ExecResult{Output: s.cd(args)}
	case "pwd":
		return ExecResult{Output: s.Path()}
	case "cat":
		return s.cat(args)
	case "edit":
		return s.edit("edit", args, false)
	case "vi", "vim", "nvim":
		// The gag: the Linux-equivalent column of `edit` is real. Same
		// editor panel, but the UI runs it modal.
		return s.edit(cmd, args, true)
	case "grep":
		return ExecResult{Output: s.grep(args)}
	case "cp":
		return ExecResult{Output: s.cp(args)}
	case "send":
		return ExecResult{Output: s.send(args)}
	case "mkdir":
		return ExecResult{Output: s.mkdir(args)}
	case "touch":
		return ExecResult{Output: s.touch(args)}
	case "scan":
		return ExecResult{Output: s.scan(args)}
	case "ssh":
		return ExecResult{Output: s.ssh(args)}
	case "exit", "logout":
		out, done := s.exit()
		return ExecResult{Output: out, Done: done}
	case "curl":
		return ExecResult{Output: s.curl(args)}
	case "ps":
		return ExecResult{Output: s.ps()}
	case "kill":
		return ExecResult{Output: s.kill(args)}
	case "run":
		return ExecResult{Output: s.run(args)}
	case "messenger":
		// The session only signals; the messenger data hangs off the
		// Deck component and the UI owns the panel (messenger.go).
		return ExecResult{Messenger: true}
	case "help":
		return ExecResult{Output: helpText}
	}
	return ExecResult{Output: cmd + ": command not found"}
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
	n := find(host.Root, segs)
	if n != nil && !present(s.w, n) {
		return nil // gated off: not there yet
	}
	return n
}

// present reports whether a node currently exists in the filesystem:
// absent until its PresentWhen flag is set, and absent again once its
// AbsentWhen flag is set.
func present(w *engine.World, n *Node) bool {
	if n.PresentWhen != "" && !w.Flags[n.PresentWhen] {
		return false
	}
	if n.AbsentWhen != "" && w.Flags[n.AbsentWhen] {
		return false
	}
	return true
}

func (s *Session) resolvedPath(p string) string {
	host, segs := s.locate(p)
	path := "/" + strings.Join(segs, "/")
	if host == s.deck {
		if rest, ok := strings.CutPrefix(path, s.deck.Home); ok {
			if rest == "" {
				return "~"
			}
			return "~" + rest
		}
	}
	return path
}

// aliasFileName is the player's shell config: a hidden dotfile in the
// deck home holding `alias name=command` lines. Newbs get friendlier
// verbs out of the box (list, look, search); pros edit it to add their
// own. It is session-scoped for now — edits hold within a play session
// but reset from the authored defaults on load (the deck filesystem
// rebuilds from code; persisting it wants a generic save store).
const aliasFileName = ".aliases"

func editableTextName(name string) bool {
	return name == aliasFileName ||
		strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md")
}

// aliasTable parses the deck's ~/.aliases into a name->expansion map.
// Read fresh each command (from the deck, so it follows you onto remote
// hosts) so an `edit .aliases` takes effect at once — no `source`
// needed. Blank lines and `#` comments are ignored; malformed lines are
// skipped rather than erroring, the way a shell shrugs off a bad rc line.
func (s *Session) aliasTable() map[string]string {
	node := find(s.deck.Root, append(splitPath(s.deck.Home), aliasFileName))
	if node == nil || node.Dir {
		return nil
	}
	text := node.Text
	if node.TextFn != nil {
		text = node.TextFn(s.w)
	}
	table := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, "alias ")
		if !ok {
			continue
		}
		name, value, ok := strings.Cut(rest, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		value = unquote(strings.TrimSpace(value))
		if name != "" && value != "" {
			table[name] = value
		}
	}
	return table
}

// expandAliases rewrites the leading command word through ~/.aliases the
// way a real shell does: the first token is replaced by its expansion
// (which may carry its own args, e.g. `alias ll='ls -la'`), repeated
// while the new leading word is itself an alias. A seen-set breaks
// alias loops (alias a=b / alias b=a) so expansion always terminates.
func (s *Session) expandAliases(args []string) []string {
	table := s.aliasTable()
	if len(table) == 0 {
		return args
	}
	seen := map[string]bool{}
	for len(args) > 0 {
		head := args[0]
		expansion, ok := table[head]
		if !ok || seen[head] {
			break
		}
		seen[head] = true
		args = append(strings.Fields(expansion), args[1:]...)
	}
	return args
}

// unquote strips one layer of matching surrounding quotes, so
// `alias ll='ls -la'` stores `ls -la`, not `'ls -la'`.
func unquote(v string) string {
	if len(v) >= 2 {
		if q := v[0]; (q == '\'' || q == '"') && v[len(v)-1] == q {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func documentKind(name string) (DocumentKind, bool) {
	switch {
	case strings.HasSuffix(name, ".md"):
		return DocumentMarkdown, true
	case strings.HasSuffix(name, ".txt"):
		return DocumentText, true
	default:
		return "", false
	}
}

// --- fake file commands -----------------------------------------------

func (s *Session) ls(args []string) string {
	// Dotfiles hide unless -a, the way a real shell does — which is also
	// a puzzle surface: a clue tucked in a .file is only found by the
	// player who thinks to look. Flags and the optional path can arrive
	// in either order (ls -a, ls -a /path, ls /path). The bare word
	// `hidden` also means -a, so the friendly form `list hidden` works.
	all := false
	target := "."
	for _, a := range args {
		if a == "hidden" {
			all = true
			continue
		}
		if len(a) > 1 && strings.HasPrefix(a, "-") {
			if strings.Contains(a, "a") {
				all = true
			}
			continue
		}
		target = a
	}
	n := s.node(target)
	if n == nil {
		return "ls: cannot access '" + target + "': No such file or directory"
	}
	if !n.Dir {
		return n.Name
	}
	names := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		if !present(s.w, c) {
			continue // gated off until its flag lands
		}
		if !all && strings.HasPrefix(c.Name, ".") {
			continue // hidden unless -a
		}
		name := c.Name
		if c.Dir {
			name += "/"
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return ""
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
	return s.fileText(n)
}

func (s *Session) fileText(n *Node) string {
	if n.TextFn != nil {
		return n.TextFn(s.w)
	}
	return n.Text
}

func (s *Session) cat(args []string) ExecResult {
	if len(args) == 0 {
		return ExecResult{Output: "usage: cat <file>"}
	}
	var out []string
	var doc *Document
	for _, arg := range args {
		n := s.node(arg)
		switch {
		case n == nil:
			out = append(out, "cat: "+arg+": No such file or directory")
		case n.Dir:
			out = append(out, "cat: "+arg+": Is a directory")
		default:
			text := s.read(n)
			if kind, ok := documentKind(n.Name); ok {
				doc = &Document{Path: s.resolvedPath(arg), Text: text, Kind: kind}
			}
			if strings.HasSuffix(n.Name, ".md") {
				text = renderMarkdown(text)
			}
			out = append(out, text)
		}
	}
	return ExecResult{Output: strings.Join(out, "\n"), Document: doc}
}

func (s *Session) edit(cmd string, args []string, vim bool) ExecResult {
	if len(args) != 1 {
		return ExecResult{Output: "usage: " + cmd + " <file>"}
	}
	n := s.node(args[0])
	switch {
	case n == nil:
		return ExecResult{Output: cmd + ": " + args[0] + ": No such file or directory"}
	case n.Dir:
		return ExecResult{Output: cmd + ": " + args[0] + ": Is a directory"}
	case n.TextFn != nil:
		return ExecResult{Output: cmd + ": " + args[0] + ": generated file is read-only"}
	case !editableTextName(n.Name):
		return ExecResult{Output: cmd + ": " + args[0] + ": only .txt and .md files are editable"}
	}
	path := s.resolvedPath(args[0])
	return ExecResult{
		Output: "editing " + path + " in right panel",
		Edit:   &EditBuffer{Path: path, Text: n.Text, Vim: vim},
	}
}

func renderMarkdown(text string) string {
	return RenderMarkdown(text, 0)
}

// RenderMarkdown renders Markdown for terminal display. width <= 0 uses
// Glamour's default wrapping.
func RenderMarkdown(text string, width int) string {
	var (
		out string
		err error
	)
	if width > 0 {
		var r *glamour.TermRenderer
		r, err = glamour.NewTermRenderer(
			glamour.WithStyles(markdownStyle),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			out, err = r.Render(text)
		}
	} else {
		var r *glamour.TermRenderer
		r, err = glamour.NewTermRenderer(glamour.WithStyles(markdownStyle))
		if err == nil {
			out, err = r.Render(text)
		}
	}
	if err != nil {
		return text
	}
	return strings.TrimRight(out, "\n")
}

func uintPtr(v uint) *uint { return &v }

func stringPtr(v string) *string { return &v }

func (s *Session) grep(args []string) string {
	if len(args) == 0 {
		return "usage: grep [-ir] <pattern> [path...]"
	}
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		if args[0] != "-i" && args[0] != "-r" && args[0] != "-ir" && args[0] != "-ri" {
			return "grep: unsupported option " + args[0]
		}
		args = args[1:]
	}
	if len(args) == 0 {
		return "usage: grep [-ir] <pattern> [path...]"
	}
	pat := strings.ToLower(args[0])
	paths := args[1:]
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var out []string
	multiple := len(paths) > 1
	for _, arg := range paths {
		n := s.node(arg)
		switch {
		case n == nil:
			out = append(out, "grep: "+arg+": No such file or directory")
			continue
		case n.Dir:
			hadFile := false
			walkFiles(s.w, n, arg, func(path string, file *Node) {
				hadFile = true
				out = append(out, s.grepFile(pat, path, file, true)...)
			})
			if !hadFile && multiple {
				continue
			}
			continue
		}
		out = append(out, s.grepFile(pat, arg, n, multiple)...)
	}
	return strings.Join(out, "\n") // like the real thing: silent when nothing matches
}

func walkFiles(w *engine.World, n *Node, path string, visit func(string, *Node)) {
	if !n.Dir {
		visit(path, n)
		return
	}
	for _, c := range n.Children {
		if !present(w, c) {
			continue // gated off until its flag lands
		}
		childPath := path
		if childPath == "." {
			childPath = c.Name
		} else if childPath == "/" {
			childPath += c.Name
		} else {
			childPath += "/" + c.Name
		}
		walkFiles(w, c, childPath, visit)
	}
}

func (s *Session) grepFile(pat, path string, n *Node, prefix bool) []string {
	var out []string
	for _, ln := range strings.Split(s.fileText(n), "\n") {
		if strings.Contains(strings.ToLower(ln), pat) {
			s.setFlag(n.OnRead) // a matched line counts as read
			if prefix {
				out = append(out, path+":"+ln)
			} else {
				out = append(out, ln)
			}
		}
	}
	return out
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
	placed.Touched = false
	placed.BackedUp = false
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

// send uploads a deck-resident file to the contact's drop — the
// delivery half of a mission (docs/draft.md: mission 1 closes over the
// wire). Deck-only on purpose: retrieve, then deliver — a file still
// sitting on a remote host has to be copied home first. Any file
// sends (no wall); only hooked files advance the story.
func (s *Session) send(args []string) string {
	if len(args) != 1 {
		return "usage: send <file>"
	}
	host, _ := s.locate(args[0])
	node := s.node(args[0])
	switch {
	case node == nil:
		return "send: " + args[0] + ": No such file or directory"
	case node.Dir:
		return "send: " + args[0] + ": Is a directory"
	case host != s.deck:
		return "send: can only send from the deck — copy it home first"
	}
	s.setFlag(node.OnSend)
	return "uploading " + node.Name + " → drop... done"
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
		n := File(segs[len(segs)-1], "")
		n.Touched = true
		replaceChild(parent, n)
	}
	return strings.Join(out, "\n")
}

// SaveEdit writes editor text back to a static .txt/.md file, creating
// a one-time backup before first overwrite of non-touch-created files.
func (s *Session) SaveEdit(path, text string) string {
	host, segs := s.locate(path)
	n := find(host.Root, segs)
	switch {
	case n == nil:
		return "save failed: " + path + ": No such file or directory"
	case n.Dir:
		return "save failed: " + path + ": Is a directory"
	case n.TextFn != nil:
		return "save failed: " + path + ": generated file is read-only"
	case !editableTextName(n.Name):
		return "save failed: " + path + ": only .txt and .md files are editable"
	}
	if !n.Touched && !n.BackedUp {
		parent := find(host.Root, segs[:len(segs)-1])
		if parent == nil || !parent.Dir {
			return "save failed: " + path + ": No such file or directory"
		}
		backup := File(nextBackupName(parent, n.Name), n.Text)
		replaceChild(parent, backup)
		n.BackedUp = true
	}
	n.Text = text
	return "saved " + s.resolvedPath(path)
}

func nextBackupName(parent *Node, name string) string {
	base := name + ".bak"
	if parent.child(base) == nil {
		return base
	}
	for i := 1; ; i++ {
		candidate := base + "." + strconv.Itoa(i)
		if parent.child(candidate) == nil {
			return candidate
		}
	}
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

func (h *Host) configuredServices() []*Service {
	if len(h.Services) > 0 {
		return h.Services
	}
	return []*Service{{
		Port:     22,
		Protocol: ProtocolSSH,
		State:    StateOpen,
		Password: h.Password,
	}}
}

func (s *Session) serviceState(host *Host, svc *Service) string {
	return serviceState(s.w, host, svc)
}

// serviceState resolves what a scan shows for one port right now: a
// pure function of flags, shared by the shell and the PDA sniffer.
// Host-level reachability (Require) is deliberately not checked here:
// a missing route means the whole host is down to the scan (user
// ruling 2026-07-09), not that its ports read closed — an authored
// open port stays open, the password stays the door.
func serviceState(w *engine.World, host *Host, svc *Service) string {
	state := svc.State
	if state == "" {
		state = StateOpen
	}
	if state == StateHidden {
		return StateHidden
	}
	if svc.OpenWhen != "" && !w.Flags[svc.OpenWhen] {
		return StateClosed
	}
	return state
}

func (s *Session) servicePassword(host *Host, svc *Service) string {
	if svc == nil {
		return host.Password
	}
	if svc.Password != "" {
		return svc.Password
	}
	if svc.Protocol == ProtocolSSH {
		return host.Password
	}
	return ""
}

func (s *Session) findService(host *Host, port int) *Service {
	for _, svc := range host.configuredServices() {
		if svc.Port == port {
			return svc
		}
	}
	return nil
}

func (s *Session) scan(args []string) string {
	if len(args) != 1 {
		return "usage: scan <host>"
	}
	host, ok := s.net[args[0]]
	if !ok {
		return "scan: Could not resolve hostname " + args[0]
	}
	return portReport(s.w, host)
}

// standardPorts is what every scan reports — the classic four, so even
// a bare host reads like a real machine on the wire. A host's
// configured services overlay these; a standard port with no service
// (or one still shut by its gate) reads closed.
var standardPorts = []struct {
	port     int
	protocol string
}{
	{21, ProtocolFTP},
	{22, ProtocolSSH},
	{23, ProtocolTelnet},
	{80, ProtocolHTTP},
}

// portReport renders the scan table for a host — shared by the shell
// and the PDA sniffer (PortReport). Standard ports always appear;
// extra configured ports follow in port order; hidden extras are
// omitted (hidden standard ports scan as closed — the disguise).
func portReport(w *engine.World, host *Host) string {
	// scan is pure recon: it always lists the ports and their real
	// states (user ruling 2026-07-10). A port's state is a fact of the
	// machine — 22 open on a box running SSH — never something the
	// story flips. The route flag gates *connecting* (ssh reports the
	// network unreachable), not what the scan shows.
	byPort := map[int]*Service{}
	for _, svc := range host.configuredServices() {
		byPort[svc.Port] = svc
	}

	scanned := func(svc *Service) string {
		state := serviceState(w, host, svc)
		if state == StateHidden {
			return StateClosed
		}
		return state
	}

	rows := []string{host.Name, "PORT  SERVICE  | STATE"}
	row := func(port int, protocol, state string) {
		rows = append(rows, fmt.Sprintf("%-5d %-8s | %s", port, strings.ToUpper(protocol), state))
	}
	for _, std := range standardPorts {
		if svc, ok := byPort[std.port]; ok {
			delete(byPort, std.port)
			row(std.port, svc.Protocol, scanned(svc))
			continue
		}
		row(std.port, std.protocol, StateClosed)
	}
	extras := make([]*Service, 0, len(byPort))
	for _, svc := range byPort {
		if serviceState(w, host, svc) != StateHidden {
			extras = append(extras, svc)
		}
	}
	sort.Slice(extras, func(i, j int) bool { return extras[i].Port < extras[j].Port })
	for _, svc := range extras {
		row(svc.Port, svc.Protocol, scanned(svc))
	}
	return strings.Join(rows, "\n")
}

func (s *Session) ssh(args []string) string {
	if len(args) == 0 {
		return "usage: ssh [username@]<host> [-p port]"
	}
	target := args[0]
	username, hostName, hasUsername := strings.Cut(target, "@")
	if !hasUsername {
		hostName = target
		username = ""
	} else if username == "" || hostName == "" || strings.Contains(hostName, "@") {
		return "usage: ssh [username@]<host> [-p port]"
	}
	port := 22
	args = args[1:]
	for len(args) > 0 {
		if len(args) != 2 || args[0] != "-p" {
			return "usage: ssh [username@]<host> [-p port]"
		}
		parsed, err := strconv.Atoi(args[1])
		if err != nil || parsed < 1 || parsed > 65535 {
			return "ssh: Bad port '" + args[1] + "'"
		}
		port = parsed
		args = nil
	}

	host, ok := s.net[hostName]
	if !ok {
		return "ssh: Could not resolve hostname " + hostName
	}
	if host.Username != "" && !hasUsername {
		return "(Placeholder) ssh: username required for " + host.Name
	}
	if host.Username == "" && hasUsername {
		return "(Placeholder) ssh: username not configured for " + host.Name
	}
	if host == s.host {
		return "already connected to " + host.Name
	}
	if host.Require != "" && !s.w.Flags[host.Require] {
		return fmt.Sprintf("ssh: connect to host %s port %d: Network is unreachable", host.Name, port)
	}
	svc := s.findService(host, port)
	if svc == nil {
		return fmt.Sprintf("ssh: connect to host %s port %d: Connection refused", host.Name, port)
	}
	state := s.serviceState(host, svc)
	if state == StateClosed || state == StateHidden {
		return fmt.Sprintf("ssh: connect to host %s port %d: Connection refused", host.Name, port)
	}
	if svc.Protocol != ProtocolSSH {
		return fmt.Sprintf("ssh: port %d on %s is running %s, not ssh", port, host.Name, svc.Protocol)
	}
	if s.servicePassword(host, svc) != "" {
		s.pending = host
		s.pendingService = svc
		s.pendingUser = username
		return fmt.Sprintf("password required for %s:%d", host.Name, port)
	}
	if host.Username != "" && username != host.Username {
		return "Permission denied, please try again."
	}
	return s.connect(host)
}

func (s *Session) password(input string) string {
	host := s.pending
	svc := s.pendingService
	username := s.pendingUser
	s.pending = nil
	s.pendingService = nil
	s.pendingUser = ""
	if (host.Username != "" && username != host.Username) ||
		strings.TrimSpace(input) != s.servicePassword(host, svc) {
		return "Permission denied, please try again."
	}
	return s.connect(host)
}

func (s *Session) connect(host *Host) string {
	s.stack = append(s.stack, conn{host: s.host, cwd: s.cwd})
	s.host = host
	s.cwd = splitPath(host.Home)
	if host.Banner != "" {
		return host.Banner
	}
	return ""
}

func (s *Session) exit() (string, bool) {
	s.pending = nil
	s.pendingService = nil
	s.pendingUser = ""
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
  Command              | Purpose                        | Linux Equivalent
  list [path]          | what's here                    | ls
  list hidden          | lists hidden dotfiles too      | ls -a
  read <file>          | read a file (.md/.txt: reader) | cat
  look <file>          | same as read                   | cat
  search <word> [path] | hunt through files for a word  | grep
  copy <src> <dst>     | copy (~ is the deck's home)    | cp
  edit <file>          | edit .md/.txt, autosave on tab | vim
  cd <path> / pwd      | move around / where am I       |
  scan <host>          | list a host's ports            | nmap
  connect [user@]host  | jack into a host               | ssh
  messenger            | your messages (right panel)    | talk
  send <file>          | hand a file to your contact    | scp
  run <file>           | execute something              |
  exit / logout        | hang up (or leave the deck)    |

Linux equivalent commands also work. The simple names are aliases
in ~/.aliases — a hidden file. edit .aliases to add your own:
alias name=command`

// Deck is the data-only marker component (the dialogue.Talkable
// pattern) attached to the deck entity: using it opens the Deck Shell
// screen. Engine dispatch declines every verb — the UI owns the
// interactive session.
type Deck struct {
	Net       map[string]*Host
	Host      string // local host name on the net
	Messenger Messenger
}

// Handle declines every command; see the type comment.
func (Deck) Handle(*engine.World, *engine.Entity, engine.Command) (string, bool) {
	return "", false
}

// DecksInScope returns entities the player can currently reach that
// carry a Deck — the terminals Buddy can log into from where he is.
// Mirrors hubs.List, but includes both visible decks and decks Buddy carries.
func DecksInScope(w *engine.World) []*engine.Entity {
	var out []*engine.Entity
	seen := map[*engine.Entity]bool{}
	add := func(e *engine.Entity) {
		if seen[e] {
			return
		}
		if _, ok := engine.Part[Deck](e); ok {
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
