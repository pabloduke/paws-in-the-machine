# Hacking

Status: first playable slice built; ICE, richer credential gates, and XP
payouts still open.

Buddy is a cat sitting at a scavenged deck. He doesn't go anywhere: he
logs into the deck — his powerful box — and works from a terminal,
sshing outward from there, exactly the way a person sits at a laptop
logged into a bigger machine. He doesn't leave his body or the room —
the puzzle is just done with (simplified) Unix-shaped tools instead of
adventure verbs. The lair and everything in it stay put while the
terminal is open; `talk`, hub travel, and the normal prompt are simply
out of reach until he closes it.

Implementation note: it's all game fiction over in-memory content —
`internal/systems/hacking` imports only the engine and stdlib string
helpers. Hosts, files, processes, and `curl`-able resources are data
declared in the game package, exactly like rooms; the commands operate
on that data and nothing else.

## Data model (`internal/systems/hacking`)

- **Node** — one entry in a host's filesystem: a directory (children)
  or a file (text). Files may carry hooks (below) and a `RunText` if
  they're executable.
- **Host** — a named system: a filesystem root, a process table,
  `curl`-served resources, configured services/ports, an optional fake
  password, an optional route flag, and a connect banner. The deck itself
  is a host (the local one). The set of hosts is a **net**, declared by
  game content the same way rooms are.
- **Service** — one fake port on a host: port number, protocol (`ssh`,
  `ftp`, `telnet`, `http`), state (`open`, `closed`, `filtered`, `hidden`),
  optional open-when flag, and optional password.
- **Process** — `{PID, name, state}` plus an optional `OnKill` hook.
- **Session** — `{current host, cwd, ssh stack}` — the state behind
  the screen. Created on login, discarded when the terminal closes; the
  net (and anything copied onto the deck) lives on the content and
  persists.
- **Deck** — a data-only marker component (the `dialogue.Talkable`
  pattern) attached to the deck entity: it carries the net, the local
  host name, and the objective hint list. Engine dispatch declines all
  verbs; the UI owns the interactive session.

## Commands (v1)

| command | behavior |
|---|---|
| `ls [path]` | list a directory; dirs get a `/` suffix |
| `cd <path>` | change directory (within the current host) |
| `pwd` | print the working directory |
| `cat <file>` | print file text; fires `OnRead` |
| `grep [-ir] <pat> [path...]` | case-insensitive recursive substring search; a match fires `OnRead` |
| `cp <src> <dst>` | copy a file; copying to the deck fires `OnCopy`. `~` always resolves to the deck's home from any host — no scp needed |
| `mkdir <dir...>` | create fake directories; parent directories must already exist |
| `touch <file...>` | create empty fake files; existing files are unchanged |
| `scan <host>` | list configured ports and states for a host |
| `ssh <host> [-p port]` | connect to SSH, defaulting to port 22 |
| `exit` / `logout` | pop back one connection; at the deck, closes the terminal and returns to the room |
| `curl <host>[/path]` | print a served resource — the recon tool |
| `ps` | list the current host's processes |
| `kill <pid>` | stop a process; fires `OnKill` |
| `run <path>` | execute a binary: prints its `RunText`, fires `OnRun` |
| `help` | list commands |

Esc closes the terminal outright from any connection depth — the
"shut the window" affordance.

Output stays realistic: `cp` and `kill` are silent on success, errors
read like the real strings (`cat: x: No such file or directory`,
`curl: (6) Could not resolve host: ...`), `grep` with no match prints
nothing.

`grep` uses the useful puzzle shape: `grep -ir <pattern> <path...>`.
Search is case-insensitive and recursive so players can quickly hunt clues
across fake host trees without memorizing exact filenames first. Directories
are searched recursively, files searched directly, and a matched hooked file
still counts as read. With no path, `grep` searches the current directory.

Creation commands are intentionally simple: Buddy is effectively root
inside this fake shell. There is no `sudo`, permissions, timestamps,
file modes, recursive `mkdir -p`, text redirection, or real filesystem
access. `mkdir` and `touch` mutate only the content-declared in-memory
host trees.

Some hosts can require a story route before they answer. If the route
flag is missing, `ssh microslop` prints a fake network-unreachable error
and leaves Buddy on the deck. This models the non-terminal part of a
hack: social engineering, stealth, and physically plugging the carried
deck into a local jack.

For hardened corpo targets, it is valid for every externally visible port
to be closed or filtered at first. Buddy does not run a magic "firewall
breacher" program to force ports open. Instead, overworld actions create
legitimate-looking holes: starting diagnostics, plugging into an internal
maintenance VLAN, tricking someone into remote support, or physically
bridging forgotten hardware. The terminal reports the wall; Buddy changes
the world so a service becomes reachable.

Some hosts can also ask for a fake password. `ssh microslop` and
`ssh microslop -p 22` both target SSH on port 22 unless content config says
otherwise. A password-required line changes the prompt to `password: `. The
next line is compared to the service or host's content-declared password. A
correct password connects; a wrong one prints `Permission denied, please try
again.` and leaves Buddy on the current host. Passwords are intentionally
simple puzzle words, not real authentication.

For the first Microslop beat, the intended setup is:

- The barista was fired from Microslop and can reveal the simple password
  after Buddy earns enough trust.
- Buddy always carries the deck.
- Buddy must get past the corpo hound into the back room and use the
  server rack to patch the deck into the local network before `ssh
  microslop` can reach the host.

## Hooks — the progression surface

Content attaches flag names to files and processes:

- `OnRead` — set when the file is first `cat`ed or `grep`-matched.
- `OnCopy` — set when the file lands on the deck.
- `OnKill` — set when the process is killed.
- `OnRun` — set when the executable is run.

Each hook sets a World flag. Dialogue, room descriptions, checks, and
quests already read flags, so `cat`-ing the right log can change what
the barista says. Discovery texture comes from reading: hostnames are
found inside files and `curl` indexes, not given away.

## Logging in

A deck in scope appears in the left panel (the hub-travel panel) under
a green `// UPLINK` subhead, set apart from the city districts to read
as a log-in target rather than a place you walk to. Tab focuses the
panel, arrows move, Enter on the deck row opens the terminal — the same
navigation as hub travel. `DecksInScope` includes visible decks and the
deck Buddy carries, so the terminal is available outside the lair when
the story wants physical plug-in beats. (Typing `log in` / `use deck`
still works as an alias path.)

## Screen layout

Full-screen swap while a session is live (the three-panel room UI and
LOG are hidden, not destroyed — closing the terminal restores them
exactly):

- **Terminal** (~75% width): bordered, titled `CYBERDECK // <host>`,
  scrollback viewport for narration/output (PgUp/PgDn), and a prompt
  line attached beneath it — `paws_in_the_machine@host:path $` — where all typing
  lands.
- **Quest panel** (~25%, read-only): OBJECTIVE (first unmet entry in
  the deck's flag→hint list, else a placeholder), LOCATION
  (`host:/path`), DISCOVERIES (files copied onto the deck, "none yet"
  when empty), STATUS (link/ICE/trace placeholders).
- Narrow terminals keep the terminal usable first; the quest panel
  shrinks. Widths are clamped — no negative-width rendering.

## Later

- Credential gates on `ssh` beyond simple passwords (a host requiring a key
  file present on the deck).
- ICE as processes that fight back; traces; disconnect pressure.
- XP payouts for completed intrusion beats (content-assigned per
  `stealth.md` — hacks are played, not rolled).
- More commands as puzzles need them (`mv`, `rm`, `chmod` ...).
