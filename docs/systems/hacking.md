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

## Design rule: no checks in the terminal, ever

The game's central mechanical split: the cat's body is **character
skill** (Stealth/Agility/Charm, XP, hidden seeded rolls — the world
says "you can't, *yet*: come back stronger"); the deck is **player
skill** (the terminal says "you haven't figured it out, *yet*: come
back smarter"). Buddy already knows how to hack — the player is
Buddy's mind at the deck.

So nothing in the terminal may roll, gate on a stat, or spend a
check: no Hacking stat, no breach rolls, no XP-gated commands.
ICE difficulty is always puzzle difficulty — harder to figure out,
never harder to roll. If hacking ever needs progression, it comes as
**tools** (a cracked binary found in play that opens new commands),
never numbers. Completed intrusion beats may *award* XP (the body
grows; see below) — they must never *require* it.

Implementation note: it's all game fiction over in-memory content —
`internal/systems/hacking` imports only the engine and stdlib string
helpers. Hosts, files, processes, and `curl`-able resources are data
declared in the game package, exactly like rooms; the commands operate
on that data and nothing else. Markdown notes render through Glamour so
`cat` can show Buddy's generated notes cleanly in the terminal reader
panel.

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
  pattern) attached to the deck entity: it carries the net and the local
  host name. Engine dispatch declines all verbs; the UI owns the
  interactive session.
- **Dynamic file** — a fake file whose text is rendered from world state
  when read or searched. Buddy's `~/notes/notes.md` uses this for notes
  derived from flags, without duplicating save state.

## Commands (v1)

| command | behavior |
|---|---|
| `ls [path]` | list a directory; dirs get a `/` suffix |
| `cd <path>` | change directory (within the current host) |
| `pwd` | print the working directory |
| `cat <file>` | print file text; `.md` and `.txt` files open in the reader panel; fires `OnRead` |
| `edit <file>` | open a static `.md` or `.txt` file in the right-panel editor; autosaves on Tab |
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

`edit <file>` is a tiny notepad for static `.md` and `.txt` files.
Markdown edits are raw source text; formatted display happens when the
player later `cat`s the file. Dynamic generated files like
`~/notes/notes.md` are read-only because they are derived from world
flags. Files created with `touch` save directly. Any other editable text
file gets a one-time sibling backup before the first save, using
`<name>.bak`, then `.bak.1`, `.bak.2`, and so on if needed.

Some hosts can require a story route before they answer. If the route
flag is missing, `ssh microslop` prints a fake network-unreachable error
and leaves Buddy on the deck. This models the non-terminal part of a
hack: social engineering, stealth, and physically bridging local
hardware (a rack's maintenance jack, a patched cable) onto the lair's
uplink. The deck itself never leaves the lair — hacking starts at home.

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
Buddy's notes already read flags, so `cat`-ing the right log can change
what the barista says. Discovery texture comes from reading: hostnames are
found inside files and `curl` indexes, not given away.

## Buddy's notes

The player-facing notes surface is a file on the deck, not a menu. Buddy keeps
`~/notes/notes.md`, a Markdown-flavored text file generated from flags.
Players read it with `cat ~/notes/notes.md` and can search it with
`grep -ir <pattern> ~/notes`. Notes are written as things Buddy has found
or inferred, with vague clue texture allowed, but not explicit next-step
instructions.

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

- **Terminal** (~60% width when the reader is active, otherwise dominant):
  bordered, titled `CYBERDECK // <host>`, with scrollback bottom-anchored above the in-panel prompt —
  `paws_in_the_machine@host:path $` — where all typing lands in CRT green.
- **Reader/editor panel** (~40% width when active): opens when `cat`
  reads a Markdown or text file, rendering Markdown with Glamour and text
  as wrapped plain output. `edit` opens the same panel as a multiline text
  editor. The terminal scrollback keeps the typed command plus a short
  `opened <path> in reader` or save notice instead of duplicating the full
  document. Tab toggles reader focus; in the editor, Tab autosaves and
  returns focus to the terminal. Up/Down scroll the focused reader or move
  the editor cursor.
- Narrow terminals keep the terminal usable first; the reader panel
  shrinks. Widths are clamped — no negative-width rendering.

## Later

- Credential gates on `ssh` beyond simple passwords (a host requiring a key
  file present on the deck).
- ICE as processes that fight back; traces; disconnect pressure.
- XP payouts for completed intrusion beats (content-assigned per
  `stealth.md` — hacks are played, not rolled).
- More commands as puzzles need them (`mv`, `rm`, `chmod` ...).
