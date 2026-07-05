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
			Name: "deck",
			Home: "/home/paws_in_the_machine",
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
			Home:     "/",
			Password: "apple",
			Require:  "microslop_route_open",
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

	if out := exec(t, s, "ssh microslop"); !strings.Contains(out, "Network is unreachable") {
		t.Fatalf("ssh without local route: %q", out)
	}
	if out := exec(t, s, "scan microslop"); !strings.Contains(out, "all scanned ports filtered") {
		t.Fatalf("scan route-gated host: %q", out)
	}
	if got := s.Prompt(); got != "paws_in_the_machine@deck:~ $ " {
		t.Fatalf("unreachable host should keep shell prompt, got %q", got)
	}

	w.Flags["microslop_route_open"] = true
	if out := exec(t, s, "scan microslop"); !strings.Contains(out, "22  ssh") || !strings.Contains(out, "open") {
		t.Fatalf("scan open route: %q", out)
	}
	if out := exec(t, s, "ssh microslop"); !strings.Contains(out, "password required") {
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

	exec(t, s, "ssh microslop")
	if out := exec(t, s, "apple"); !strings.Contains(out, "MICROSLOP") {
		t.Fatalf("correct password banner: %q", out)
	}
	if s.HostName() != "microslop" || s.Prompt() != "paws_in_the_machine@microslop:/ $ " {
		t.Fatalf("connected host=%q prompt=%q", s.HostName(), s.Prompt())
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
