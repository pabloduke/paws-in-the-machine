package hacking_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func testNet() map[string]*hacking.Host {
	return map[string]*hacking.Host{
		"deck": {
			Name:     "deck",
			Username: "paws_in_the_machine",
			Home:     "/home/paws_in_the_machine",
			Root: hacking.Dir("/",
				hacking.Dir("home",
					hacking.Dir("paws_in_the_machine",
						hacking.Dir("notes",
							hacking.DynamicFile("notes.md", func(w *engine.World) string {
								if w.Flags["found_password"] {
									return "# Notes\n\n- password: apple"
								}
								return "# Notes\n\nNothing solid yet."
							}),
						),
						hacking.File(".aliases", "# aliases\nalias list=ls\nalias look=cat\n"),
					),
				),
			),
		},
		"relay.net": {
			Name:   "relay.net",
			Home:   "/",
			Banner: "RELAY — abandoned but listening.",
			Services: []*hacking.Service{
				{Port: 21, Protocol: hacking.ProtocolFTP, State: hacking.StateOpen},
				{Port: 22, Protocol: hacking.ProtocolSSH, State: hacking.StateOpen},
				{Port: 23, Protocol: hacking.ProtocolTelnet, State: hacking.StateClosed},
			},
			Root: hacking.Dir("/",
				hacking.Dir("var",
					hacking.Dir("log",
						&hacking.Node{Name: "net.log", Text: "ping ok\nsunfarm.arc keeps answering\nnoise", OnRead: "read_netlog"},
					),
				),
				hacking.Dir("srv",
					&hacking.Node{Name: "sun.frag", Text: "SUN//frag", OnCopy: "got_fragment"},
					&hacking.Node{Name: "dig.bin", RunText: "unearthing... interrupted.", OnRun: "ran_dig"},
				),
			),
			Procs: []*hacking.Process{
				{PID: 27, Name: "whisperd", OnKill: "silenced_whisper"},
			},
			Served: map[string]string{"/": "relay index: sunfarm.arc [answers]"},
		},
		"microslop": {
			Name:     "microslop",
			Username: "jane_doe",
			Home:     "/",
			Password: "apple",
			Require:  "test_route_open",
			Banner:   "MICROSLOP CORP intranet.",
			Root: hacking.Dir("/",
				hacking.Dir("home",
					hacking.File("readme.txt", "search logs for sun"),
				),
				hacking.Dir("var",
					hacking.Dir("log",
						hacking.File("access.log", "sun notice moved to /srv/archive/sun_notice.txt"),
					),
				),
				hacking.Dir("srv",
					hacking.Dir("archive",
						&hacking.Node{
							Name:   "sun_notice.txt",
							Text:   "sun liability memo",
							OnRead: "read_notice",
							OnCopy: "got_notice",
						},
					),
				),
			),
		},
	}
}

func newShell(t *testing.T) (*engine.World, *hacking.Session) {
	t.Helper()
	w := engine.NewWorld()
	s, err := hacking.NewSession(w, testNet(), "deck")
	if err != nil {
		t.Fatal(err)
	}
	return w, s
}

func TestCompletionCommandsAliasesHostsAndPaths(t *testing.T) {
	_, s := newShell(t)

	if line, matches := s.Complete("sca"); line != "scan " || len(matches) != 1 {
		t.Fatalf("command completion: line=%q matches=%v", line, matches)
	}
	if line, matches := s.Complete("lis"); line != "list " || len(matches) != 1 {
		t.Fatalf("alias completion: line=%q matches=%v", line, matches)
	}
	if line, matches := s.Complete("ssh rel"); line != "ssh relay.net " || len(matches) != 1 {
		t.Fatalf("host completion: line=%q matches=%v", line, matches)
	}
	if line, matches := s.Complete("ssh ja"); line != "ssh jane_doe@microslop " || len(matches) != 1 {
		t.Fatalf("username host completion: line=%q matches=%v", line, matches)
	}
	if line, matches := s.Complete("cat ~/notes/no"); line != "cat ~/notes/notes.md " || len(matches) != 1 {
		t.Fatalf("path completion: line=%q matches=%v", line, matches)
	}
	if line, matches := s.Complete("cd ~/no"); line != "cd ~/notes/" || len(matches) != 1 {
		t.Fatalf("directory completion: line=%q matches=%v", line, matches)
	}
}

func TestCompletionIsDisabledForPasswords(t *testing.T) {
	w, s := newShell(t)
	w.Flags["test_route_open"] = true
	s.Exec("ssh jane_doe@microslop")
	if !s.AwaitingPassword() {
		t.Fatal("ssh should leave the session waiting for a password")
	}
	if line, matches := s.Complete("app"); line != "app" || len(matches) != 0 {
		t.Fatalf("password input must not complete: line=%q matches=%v", line, matches)
	}
}

func exec(t *testing.T, s *hacking.Session, line string) string {
	t.Helper()
	out, done := s.Exec(line)
	if done {
		t.Fatalf("%q unexpectedly ended the session", line)
	}
	return out
}

func TestFilesystemNavigation(t *testing.T) {
	w, s := newShell(t)
	if got := s.Prompt(); got != "paws_in_the_machine@deck:~ $ " {
		t.Fatalf("home prompt: %q", got)
	}
	if out := exec(t, s, "ls"); out != "notes/" {
		t.Fatalf("ls home: %q", out)
	}
	if out := exec(t, s, "cat ~/notes/notes.md"); !strings.Contains(stripANSI(out), "Nothing solid yet") {
		t.Fatalf("cat notes: %q", stripANSI(out))
	}
	w.Flags["found_password"] = true
	if out := exec(t, s, "grep -ir apple ~/notes"); !strings.Contains(out, "~/notes/notes.md") {
		t.Fatalf("grep generated notes: %q", out)
	}
	if out := exec(t, s, "cd /home"); out != "" {
		t.Fatalf("cd: %q", out)
	}
	if out := exec(t, s, "pwd"); out != "/home" {
		t.Fatalf("pwd: %q", out)
	}
	if out := exec(t, s, "cd .."); out != "" || s.Path() != "/" {
		t.Fatalf("cd ..: %q path=%q", out, s.Path())
	}
	if out := exec(t, s, "cat nope"); out != "cat: nope: No such file or directory" {
		t.Fatalf("cat missing: %q", out)
	}
	if out := exec(t, s, "frobnicate"); out != "frobnicate: command not found" {
		t.Fatalf("unknown cmd: %q", out)
	}
}

func TestDetailedCatReportsMarkdownDocument(t *testing.T) {
	_, s := newShell(t)
	result := s.ExecDetailed("cat ~/notes/notes.md")
	if result.Done {
		t.Fatalf("cat should not end session")
	}
	if result.Document == nil {
		t.Fatalf("markdown cat should include document metadata")
	}
	if result.Document.Path != "~/notes/notes.md" {
		t.Fatalf("document path=%q", result.Document.Path)
	}
	if result.Document.Kind != hacking.DocumentMarkdown {
		t.Fatalf("document kind=%q", result.Document.Kind)
	}
	if !strings.Contains(result.Document.Text, "Nothing solid yet") {
		t.Fatalf("document markdown missing notes: %q", result.Document.Text)
	}
	if !strings.Contains(stripANSI(result.Output), "Nothing solid yet") {
		t.Fatalf("legacy output should still render markdown: %q", stripANSI(result.Output))
	}

	exec(t, s, "touch scratch.txt")
	result = s.ExecDetailed("cat scratch.txt")
	if result.Document == nil || result.Document.Kind != hacking.DocumentText {
		t.Fatalf("txt cat should include text document: %#v", result.Document)
	}

	result = s.ExecDetailed("cat nope")
	if result.Document != nil {
		t.Fatalf("missing file should not include document: %#v", result.Document)
	}
	if result.Output != "cat: nope: No such file or directory" {
		t.Fatalf("missing file output: %q", result.Output)
	}
}

func TestMarkdownRenderUsesCompactReaderStyle(t *testing.T) {
	out := hacking.RenderMarkdown("# Mine\n\nhello *soft* **loud**\n\n- one\n* star item\n- two with `code`", 36)
	plain := stripANSI(out)
	if strings.Contains(plain, "# Mine") {
		t.Fatalf("heading marker should not render in compact style: %q", plain)
	}
	for _, want := range []string{"// MINE", "hello *soft* **loud**", "• one", "• star item", "• two with code"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("rendered markdown missing %q: %q", want, plain)
		}
	}
	if strings.Contains(out, "\x1b[48;") {
		t.Fatalf("reader markdown should not emit background-filled blocks: %q", out)
	}
}

func TestEditTouchedTextAndMarkdownFiles(t *testing.T) {
	_, s := newShell(t)
	exec(t, s, "touch myNotes.md scratch.txt")

	result := s.ExecDetailed("edit myNotes.md")
	if result.Edit == nil {
		t.Fatalf("edit should open markdown editor: %#v", result)
	}
	if result.Edit.Path != "~/myNotes.md" || result.Edit.Text != "" {
		t.Fatalf("edit buffer: %#v", result.Edit)
	}
	if out := s.SaveEdit(result.Edit.Path, "# My Notes\n\nhello"); out != "saved ~/myNotes.md" {
		t.Fatalf("save markdown: %q", out)
	}
	if out := exec(t, s, "cat myNotes.md"); !strings.Contains(stripANSI(out), "hello") {
		t.Fatalf("saved markdown missing: %q", stripANSI(out))
	}
	if listing := exec(t, s, "ls"); strings.Contains(listing, ".bak") {
		t.Fatalf("touch-created file should not get backup: %q", listing)
	}

	result = s.ExecDetailed("edit scratch.txt")
	if result.Edit == nil {
		t.Fatalf("edit should open txt editor: %#v", result)
	}
	if out := s.SaveEdit(result.Edit.Path, "plain note"); out != "saved ~/scratch.txt" {
		t.Fatalf("save txt: %q", out)
	}
	result = s.ExecDetailed("cat scratch.txt")
	if result.Document == nil || result.Document.Kind != hacking.DocumentText ||
		result.Document.Text != "plain note" {
		t.Fatalf("saved txt document: %#v", result.Document)
	}
}

func TestEditExistingTextFileCreatesOneBackup(t *testing.T) {
	w, s := newShell(t)
	w.Flags["test_route_open"] = true
	exec(t, s, "ssh jane_doe@microslop -p 22")
	exec(t, s, "apple")

	result := s.ExecDetailed("edit /home/readme.txt")
	if result.Edit == nil || result.Edit.Text != "search logs for sun" {
		t.Fatalf("edit existing txt: %#v", result.Edit)
	}
	if out := s.SaveEdit(result.Edit.Path, "changed"); out != "saved /home/readme.txt" {
		t.Fatalf("save existing txt: %q", out)
	}
	if out := exec(t, s, "cat /home/readme.txt.bak"); out != "search logs for sun" {
		t.Fatalf("backup should contain original text: %q", out)
	}
	if out := s.SaveEdit(result.Edit.Path, "changed again"); out != "saved /home/readme.txt" {
		t.Fatalf("second save existing txt: %q", out)
	}
	if listing := exec(t, s, "ls /home"); strings.Contains(listing, "readme.txt.bak.1") {
		t.Fatalf("second save should not create another backup: %q", listing)
	}
	if out := exec(t, s, "cat /home/readme.txt"); out != "changed again" {
		t.Fatalf("second save text: %q", out)
	}
}

func TestEditErrors(t *testing.T) {
	_, s := newShell(t)
	if out := s.ExecDetailed("edit").Output; out != "usage: edit <file>" {
		t.Fatalf("edit usage: %q", out)
	}
	if out := s.ExecDetailed("edit missing.txt").Output; out != "edit: missing.txt: No such file or directory" {
		t.Fatalf("edit missing: %q", out)
	}
	if out := s.ExecDetailed("edit notes").Output; out != "edit: notes: Is a directory" {
		t.Fatalf("edit dir: %q", out)
	}
	if out := s.ExecDetailed("edit ~/notes/notes.md").Output; out != "edit: ~/notes/notes.md: generated file is read-only" {
		t.Fatalf("edit dynamic: %q", out)
	}
	exec(t, s, "touch data.log")
	if out := s.ExecDetailed("edit data.log").Output; out != "edit: data.log: only .txt and .md files are editable" {
		t.Fatalf("edit extension: %q", out)
	}
}

// Friendly aliases from ~/.aliases expand to the real command (list→ls),
// so newbs and pros hit the same code path.
func TestAliasExpandsToRealCommand(t *testing.T) {
	_, s := newShell(t)
	if aliased, real := exec(t, s, "list"), exec(t, s, "ls"); aliased != real {
		t.Fatalf("list should expand to ls: list=%q ls=%q", aliased, real)
	}
}

// Editing ~/.aliases takes effect on the next command — no source needed
// — and a full-string expansion (alias with its own args) works.
func TestAliasEditTakesEffectAndCarriesArgs(t *testing.T) {
	_, s := newShell(t)
	res := s.ExecDetailed("edit .aliases")
	if res.Edit == nil {
		t.Fatalf(".aliases should be editable: %#v", res)
	}
	// `ll` expands to `ls -a`, revealing the (hidden) .aliases itself.
	s.SaveEdit(res.Edit.Path, "alias ll='ls -a'\n")
	if out := exec(t, s, "ll"); !strings.Contains(out, ".aliases") {
		t.Fatalf("ll should expand to `ls -a` and show dotfiles: %q", out)
	}
}

// A cyclic alias must terminate rather than expand forever.
func TestAliasLoopTerminates(t *testing.T) {
	_, s := newShell(t)
	res := s.ExecDetailed("edit .aliases")
	s.SaveEdit(res.Edit.Path, "alias a=b\nalias b=a\n")
	if out := exec(t, s, "a"); !strings.Contains(out, "command not found") {
		t.Fatalf("a cyclic alias should terminate at a real dispatch: %q", out)
	}
}

// ls hides dotfiles unless -a — the real-shell behavior, and the puzzle
// surface it enables (a clue in a .file is only found by looking).
func TestLsHidesDotfilesUnlessAll(t *testing.T) {
	_, s := newShell(t)
	exec(t, s, "touch .secret")

	if out := exec(t, s, "ls"); strings.Contains(out, ".secret") {
		t.Fatalf("plain ls should hide the dotfile: %q", out)
	}
	if out := exec(t, s, "ls -a"); !strings.Contains(out, ".secret") {
		t.Fatalf("ls -a should reveal the dotfile: %q", out)
	}
	// The path can follow the flag.
	if out := exec(t, s, "ls -a ."); !strings.Contains(out, ".secret") {
		t.Fatalf("ls -a <path> should reveal the dotfile: %q", out)
	}
	// The friendly form: `hidden` as a bare word means -a, so the
	// help table's `list hidden` works.
	if out := exec(t, s, "list hidden"); !strings.Contains(out, ".secret") {
		t.Fatalf("list hidden should reveal the dotfile: %q", out)
	}
}

func TestCreateDirectoriesAndFiles(t *testing.T) {
	_, s := newShell(t)

	if out := exec(t, s, "mkdir scratch logs"); out != "" {
		t.Fatalf("mkdir should be silent: %q", out)
	}
	listing := exec(t, s, "ls")
	if !strings.Contains(listing, "scratch/") || !strings.Contains(listing, "logs/") {
		t.Fatalf("created dirs missing from ls: %q", listing)
	}
	if out := exec(t, s, "cd scratch"); out != "" || s.Path() != "/home/paws_in_the_machine/scratch" {
		t.Fatalf("cd created dir: out=%q path=%q", out, s.Path())
	}
	if out := exec(t, s, "touch note.txt empty.log"); out != "" {
		t.Fatalf("touch should be silent: %q", out)
	}
	listing = exec(t, s, "ls")
	if !strings.Contains(listing, "empty.log") || !strings.Contains(listing, "note.txt") {
		t.Fatalf("created files missing from ls: %q", listing)
	}
	if out := exec(t, s, "cat note.txt"); out != "" {
		t.Fatalf("empty touched file should cat silently: %q", out)
	}
	if out := exec(t, s, "touch note.txt"); out != "" {
		t.Fatalf("touch existing file should be silent: %q", out)
	}
}

func TestCreateCommandErrors(t *testing.T) {
	_, s := newShell(t)

	if out := exec(t, s, "mkdir"); out != "usage: mkdir <dir...>" {
		t.Fatalf("mkdir usage: %q", out)
	}
	if out := exec(t, s, "touch"); out != "usage: touch <file...>" {
		t.Fatalf("touch usage: %q", out)
	}
	if out := exec(t, s, "mkdir notes"); out != "mkdir: cannot create directory 'notes': File exists" {
		t.Fatalf("mkdir existing file: %q", out)
	}
	if out := exec(t, s, "mkdir missing/child"); out != "mkdir: cannot create directory 'missing/child': No such file or directory" {
		t.Fatalf("mkdir missing parent: %q", out)
	}
	if out := exec(t, s, "mkdir scratch"); out != "" {
		t.Fatalf("mkdir scratch: %q", out)
	}
	if out := exec(t, s, "mkdir scratch"); out != "mkdir: cannot create directory 'scratch': File exists" {
		t.Fatalf("mkdir existing dir: %q", out)
	}
	if out := exec(t, s, "touch scratch"); out != "touch: cannot touch 'scratch': Is a directory" {
		t.Fatalf("touch existing dir: %q", out)
	}
	if out := exec(t, s, "touch missing/file.txt"); out != "touch: cannot touch 'missing/file.txt': No such file or directory" {
		t.Fatalf("touch missing parent: %q", out)
	}
}

func TestCreateCommandsOnRemoteAndDeckHome(t *testing.T) {
	_, s := newShell(t)
	exec(t, s, "ssh relay.net")

	if out := exec(t, s, "mkdir /tmp"); out != "" {
		t.Fatalf("remote mkdir: %q", out)
	}
	if out := exec(t, s, "touch /tmp/remote.txt"); out != "" {
		t.Fatalf("remote touch: %q", out)
	}
	if out := exec(t, s, "cat /tmp/remote.txt"); out != "" {
		t.Fatalf("remote touched file should be empty: %q", out)
	}
	if out := exec(t, s, "touch ~/deck-note.txt"); out != "" {
		t.Fatalf("touch deck home from remote: %q", out)
	}
	if out := exec(t, s, "cat ~/deck-note.txt"); out != "" {
		t.Fatalf("deck-home touched file should be empty: %q", out)
	}
}

func TestSSHGrepAndHooks(t *testing.T) {
	w, s := newShell(t)
	if out := exec(t, s, "curl relay.net"); !strings.Contains(out, "sunfarm.arc") {
		t.Fatalf("curl: %q", out)
	}
	if out := exec(t, s, "curl nowhere.net"); !strings.Contains(out, "(6)") {
		t.Fatalf("curl unknown: %q", out)
	}
	if out := exec(t, s, "ssh relay.net"); !strings.Contains(out, "RELAY") {
		t.Fatalf("ssh banner: %q", out)
	}
	out, done := s.Exec("exit")
	if done || !strings.Contains(out, "closed") {
		t.Fatalf("exit should pop to deck: %q done=%v", out, done)
	}
	if out := exec(t, s, "ssh relay.net -p 21"); !strings.Contains(out, "running ftp, not ssh") {
		t.Fatalf("ssh wrong service: %q", out)
	}
	if out := exec(t, s, "ssh relay.net -p 22"); !strings.Contains(out, "RELAY") {
		t.Fatalf("ssh explicit port: %q", out)
	}
	if out := exec(t, s, "grep sunfarm /var/log/net.log"); !strings.Contains(out, "sunfarm.arc keeps answering") {
		t.Fatalf("grep: %q", out)
	}
	if !w.Flags["read_netlog"] {
		t.Fatalf("grep match should fire OnRead; flags=%v", w.Flags)
	}
	if out := exec(t, s, "grep -ir SUN /var"); !strings.Contains(out, "/var/log/net.log:sunfarm.arc keeps answering") {
		t.Fatalf("grep recursive case-insensitive: %q", out)
	}
	if out := exec(t, s, "grep zebra /var/log/net.log"); out != "" {
		t.Fatalf("grep no-match should be silent: %q", out)
	}
	out, done = s.Exec("exit")
	if done || !strings.Contains(out, "closed") {
		t.Fatalf("exit should pop to deck: %q done=%v", out, done)
	}
	if s.HostName() != "deck" {
		t.Fatalf("after exit host=%q", s.HostName())
	}
}

func TestPasswordGatedSSH(t *testing.T) {
	w, s := newShell(t)

	// scan is pure recon: 22 reads open from the start (a fact of the
	// machine), the route/password gate connecting, not the scan
	// (user ruling 2026-07-10).
	if out := exec(t, s, "scan microslop"); !strings.Contains(out, "22    SSH      | open") {
		t.Fatalf("scan should list port 22 open regardless of route: %q", out)
	}
	if out := exec(t, s, "ssh jane_doe@microslop"); !strings.Contains(out, "Network is unreachable") {
		t.Fatalf("ssh without local route: %q", out)
	}
	if got := s.Prompt(); got != "paws_in_the_machine@deck:~ $ " {
		t.Fatalf("unreachable host should keep shell prompt, got %q", got)
	}

	w.Flags["test_route_open"] = true
	if out := exec(t, s, "ssh microslop"); out != "(Placeholder) ssh: username required for microslop" {
		t.Fatalf("configured host should require username@host syntax: %q", out)
	}
	if s.AwaitingPassword() {
		t.Fatal("missing username must not enter password mode")
	}
	if out := exec(t, s, "ssh wrong_user@microslop"); !strings.Contains(out, "password required") {
		t.Fatalf("syntactically valid username should reach masked authentication: %q", out)
	}
	if out := exec(t, s, "apple"); out != "Permission denied, please try again." {
		t.Fatalf("wrong username with correct password should be denied: %q", out)
	}
	if s.HostName() != "deck" {
		t.Fatalf("wrong username should stay on deck, host=%q", s.HostName())
	}
	if out := exec(t, s, "ssh someone@relay.net"); out != "(Placeholder) ssh: username not configured for relay.net" {
		t.Fatalf("undeclared hosts should reject invented usernames: %q", out)
	}
	if out := exec(t, s, "scan microslop"); !strings.Contains(out, "22    SSH      | open") {
		t.Fatalf("port 22 stays open after the route opens: %q", out)
	}
	if out := exec(t, s, "ssh jane_doe@microslop"); !strings.Contains(out, "password required") {
		t.Fatalf("ssh password prompt: %q", out)
	}
	if got := s.Prompt(); got != "password: " {
		t.Fatalf("password prompt: %q", got)
	}
	if out := exec(t, s, "wrong"); out != "Permission denied, please try again." {
		t.Fatalf("wrong password: %q", out)
	}
	if s.HostName() != "deck" {
		t.Fatalf("wrong password should stay on deck, host=%q", s.HostName())
	}

	exec(t, s, "ssh jane_doe@microslop -p 22")
	if out := exec(t, s, "apple"); !strings.Contains(out, "MICROSLOP") {
		t.Fatalf("correct password banner: %q", out)
	}
	if s.HostName() != "microslop" || s.Prompt() != "jane_doe@microslop:/ $ " {
		t.Fatalf("connected host=%q prompt=%q", s.HostName(), s.Prompt())
	}
	if !s.IsRemote() {
		t.Fatal("connected Microslop session should report remote context")
	}
	if out := exec(t, s, "grep sun /var/log/access.log"); !strings.Contains(out, "/srv/archive/sun_notice.txt") {
		t.Fatalf("grep microslop logs: %q", out)
	}
	if out := exec(t, s, "cat /srv/archive/sun_notice.txt"); out != "sun liability memo" {
		t.Fatalf("cat notice: %q", out)
	}
	if !w.Flags["read_notice"] {
		t.Fatalf("reading notice should set flag; flags=%v", w.Flags)
	}
	if out := exec(t, s, "cp /srv/archive/sun_notice.txt ~/"); out != "" {
		t.Fatalf("copy notice: %q", out)
	}
	if !w.Flags["got_notice"] {
		t.Fatalf("copying notice should set flag; flags=%v", w.Flags)
	}
	if out, done := s.Exec("exit"); done || !strings.Contains(out, "closed") {
		t.Fatalf("exit should return to deck: out=%q done=%v", out, done)
	}
	if s.IsRemote() {
		t.Fatal("deck session should not report remote context")
	}
}

func TestCopyToDeckFiresHookAndDiscovery(t *testing.T) {
	w, s := newShell(t)
	exec(t, s, "ssh relay.net")
	if out := exec(t, s, "cp /srv/sun.frag ~/"); out != "" {
		t.Fatalf("cp should be silent: %q", out)
	}
	if !w.Flags["got_fragment"] {
		t.Fatalf("cp to deck should fire OnCopy; flags=%v", w.Flags)
	}
	// The copy landed on the deck's fake filesystem.
	exec(t, s, "exit")
	if out := exec(t, s, "cat ~/sun.frag"); out != "SUN//frag" {
		t.Fatalf("downloaded copy: %q", out)
	}
}

func TestProcessesAndRun(t *testing.T) {
	w, s := newShell(t)
	exec(t, s, "ssh relay.net")
	if out := exec(t, s, "ps"); !strings.Contains(out, "whisperd") || !strings.Contains(out, "S") {
		t.Fatalf("ps: %q", out)
	}
	if out := exec(t, s, "kill 27"); out != "" {
		t.Fatalf("kill should be silent: %q", out)
	}
	if !w.Flags["silenced_whisper"] {
		t.Fatalf("kill should fire OnKill; flags=%v", w.Flags)
	}
	if out := exec(t, s, "ps"); !strings.Contains(out, "Z") {
		t.Fatalf("killed process should show as zombie: %q", out)
	}
	if out := exec(t, s, "kill 27"); !strings.Contains(out, "No such process") {
		t.Fatalf("double kill: %q", out)
	}
	if out := exec(t, s, "run /srv/dig.bin"); !strings.Contains(out, "unearthing") {
		t.Fatalf("run: %q", out)
	}
	if !w.Flags["ran_dig"] {
		t.Fatalf("run should fire OnRun; flags=%v", w.Flags)
	}
	if out := exec(t, s, "run /srv/sun.frag"); !strings.Contains(out, "Permission denied") {
		t.Fatalf("run non-executable: %q", out)
	}
}

func TestExitPopsThenEndsAtDeck(t *testing.T) {
	_, s := newShell(t)
	exec(t, s, "ssh relay.net")

	// exit from a remote host pops back to the deck without ending.
	out, done := s.Exec("exit")
	if done || !strings.Contains(out, "closed") {
		t.Fatalf("exit from remote should pop, not end: %q done=%v", out, done)
	}
	if s.HostName() != "deck" {
		t.Fatalf("after exit host=%q, want deck", s.HostName())
	}

	// exit at the deck (empty stack) closes the terminal.
	out, done = s.Exec("exit")
	if !done || out != "logout" {
		t.Fatalf("exit at deck should end the session with logout: %q done=%v", out, done)
	}

	// End() narrates the Esc "close the window" affordance.
	_, s2 := newShell(t)
	exec(t, s2, "ssh relay.net")
	if got := s2.End(); !strings.Contains(got, "closed by local host") {
		t.Fatalf("End() from remote: %q", got)
	}
}
