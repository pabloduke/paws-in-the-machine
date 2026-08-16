# Hacking

Status: first playable slice built; ICE and richer credential gates
still open. XP payouts are built (#24): terminal beats pay
content-valued XP through event rules over the hook flags
(`netEvents()` in `internal/game/net.go`) — no rolls, no formula, the
amounts are content. Rules don't run while the terminal is open, so
the payout lands at logout: the session's haul, tallied at the door.
Amounts are tuning data for the calibration pass (#27).

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
- **Host** — a named system with an optional SSH username: a filesystem
  root, a process table,
  `curl`-served resources, configured services/ports, an optional fake
  password, an optional route flag, and a connect banner. The deck itself
  is a host (the local one). The set of hosts is a **net**, declared by
  game content the same way rooms are.
- **Service** — one fake port on a host: port number, protocol (`ssh`,
  `ftp`, `telnet`, `http`), state (`open`, `closed`, `hidden`),
  optional open-when flag, and optional password. Port states are
  binary on purpose (user ruling 2026-07-09): a port is open or
  closed, never `filtered` — one less word between a newb and the
  puzzle. `hidden` is authoring-only (a hidden standard port scans as
  `closed`, the disguise; a hidden extra port is omitted).

  **Two puzzle shapes** (user ruling 2026-07-10), and `scan` reports
  the true machine state for both:

  1. **Port already open, guarded by a credential.** The box is a
     running server — 22 is genuinely open — and the work is getting
     the password/keyfile. `scan` shows `open` from the start; `ssh`
     asks for the password. Example: microslop (22 open, password
     `apple`). A missing route flag (`Require`) makes the host
     *unreachable* — `ssh` reports the network unreachable — but that
     never changes what `scan` lists: you can't reach it *is not* the
     same as its ports are closed.
  2. **Ports closed, opening one is the work.** The box is dark — all
     ports read `closed` — and a port only opens when something in
     the world genuinely re-enables it (`OpenWhen` flag). This is a
     real state change to the machine, not the story hiding a scan:
     `scan` honestly shows `closed`, then `open` once the port is
     switched on. Example: okuda.grid, whose physically-disabled port
     is re-enabled by the records-office console (#16 closed-ports
     loop).
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
| `ls [-a] [path]` | list a directory; dirs get a `/` suffix. Dotfiles (names starting with `.`) are hidden unless `-a` — the real-shell behavior, and a puzzle surface: a clue authored into a `.file` is only found by a player who thinks to look. Flag and path may appear in either order. The bare word `hidden` also means `-a`, so the friendly `list hidden` works |
| `cd <path>` | change directory (within the current host) |
| `pwd` | print the working directory |
| `cat <file>` | print file text; `.md` and `.txt` files open in the reader panel; fires `OnRead` |
| `edit <file>` | open a static `.md` or `.txt` file in the right-panel editor; autosaves on Shift+Tab |
| `vi` / `vim` / `nvim <file>` | the same editor, run modal: opens in normal mode, `i` inserts, Esc returns to normal, `h/j/k/l` and arrows move, `:w` writes, `:q` quits without saving, `:wq`/`:x` both. The easter egg behind the help table's `edit → vim` Linux-equivalent column |
| `grep [-ir] <pat> [path...]` | case-insensitive recursive substring search; a match fires `OnRead` |
| `cp <src> <dst>` | copy a file; copying to the deck fires `OnCopy`. `~` always resolves to the deck's home from any host — no scp needed |
| `mkdir <dir...>` | create fake directories; parent directories must already exist |
| `touch <file...>` | create empty fake files; existing files are unchanged |
| `scan <host>` | port table for a host: `PORT SERVICE \| STATE`, state binary open/closed. The standard four (21 FTP, 22 SSH, 23 TELNET, 80 HTTP) always appear — a bare host still reads like a real machine — with configured services overlaid and extra ports appended. **scan is pure recon: it always lists ports and their real states** (user ruling 2026-07-10). A port's state is a fact of the machine — 22 open on a box running SSH — never something the story flips; the route flag gates *connecting*, not the scan |
| `ssh [username@]<host> [-p port]` | connect to SSH, defaulting to port 22; hosts with a configured username require it |
| `exit` / `logout` | pop back one connection; at the deck, closes the terminal and returns to the room |
| `curl <host>[/path]` | print a served resource — the recon tool |
| `ps` | list the current host's processes |
| `kill <pid>` | stop a process; fires `OnKill` |
| `run <path>` | execute a binary: prints its `RunText`, fires `OnRun` |
| `messenger` | toggle the messenger in the right panel (see "The messenger" below); `talk` is its alias |
| `send <file>` | upload a deck-resident file to the contact's drop; fires `OnSend`. Deck-only on purpose (retrieve, then deliver): a file on a remote host errors with "copy it home first". Any file sends (no wall); only hooked files advance the story. `scp` is its alias |
| `help` | the simple commands as a three-column table — Command / Purpose / Linux Equivalent — friendly names first; the full Linux reference is deliberately not listed (terminal folk already know it, and `.aliases` documents the mapping) |

Esc closes the terminal outright from any connection depth — the
"shut the window" affordance. Exception: while a vim buffer is open,
Esc belongs to the editor (mode switch); `:q` first, then Esc.

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

## Aliases (`~/.aliases`)

The shell has one accessibility affordance: the deck's home holds a
hidden `~/.aliases` file of `alias name=command` lines. It ships with
friendlier verbs — `list`→`ls`, `look`/`read`→`cat`, `search`→`grep`,
`copy`→`cp`, `connect`→`ssh`, `talk`→`messenger` — so a player who has
never touched a shell can still play, while the real commands always
work. Two aliases run the other way: `nmap`→`scan` and `scp`→`send`
give pros the real-world names for the game commands, same trick in
reverse (`talk` is also a nod to the old Unix chat command). This lowers the
floor without a menu; it does not replace the terminal.

Design line held deliberately (see the conversation that shaped it):

- **Real `alias` syntax**, because anyone off Linux expects it. The
  leading command word is expanded, repeatedly while the new head is
  itself an alias, with a seen-set to break `a=b`/`b=a` loops. Full
  expansion means `alias ll='ls -la'` and `alias deep='grep -ir'` just
  work. Not env vars, functions, or `$PATH` — the filename is `.aliases`,
  not `.bashrc`, precisely so it doesn't promise the whole init surface.
- **Discovery is a dotfile.** `.aliases` is hidden; `ls -a` reveals it,
  `edit .aliases` changes it, and the change takes effect on the next
  command (no `source`). `help` names the file for terminal users;
  newbs skim past it and just use the defaults.
- **Pipes are deliberately out** for now: with `grep` the only consumer,
  the useful cases (`ls`/`ps`/`scan` `| grep`) are thin and serve no
  newb — add them when a puzzle wants a real filter and a second
  consumer exists.
- **Session-scoped for now.** Edits hold within a play session but reset
  to the authored defaults on load, because the deck filesystem rebuilds
  from code and is not in the save (`docs/systems/saveload.md`).
  Persisting player edits wants a generic saved-text store rather than a
  shell-specific field on `engine.World` (which would breach the
  engine/game boundary); that is a deliberate follow-up, tracked with
  the terminal-persistence work in #28.

Some hosts can require a story route before they answer. If the route
flag is missing, `ssh username@host` prints a fake network-unreachable error
and leaves Buddy on the deck. `scan` still lists the ports (recon is
never gated); only connecting is. This models the non-terminal part of a
hack: social engineering, stealth, and physically bridging local
hardware (a rack's maintenance jack, a patched cable) onto the lair's
uplink. The deck itself never leaves the lair — hacking starts at home.

## The PDA

Buddy carries a PDA into the overworld: a salvaged pocket slab that
mirrors the deck's storage over the collar link and sniffs ports. It is
**read-only by design and menu-driven, never a shell** (user ruling
2026-07-07): a main menu offers *Check notes* (every `.md`/`.txt` on
the deck, opened in a reader) and *Scan ports* (every known host, each
showing the same report as `scan`). Nothing can be hacked, written, or
logged into from it.

Its job is field intel: after an overworld action that should open a
port (the Okuda console, the coffee-shop rack), the PDA shows the state
flip from anywhere — so Buddy knows the trip home to the deck is worth
it. Seeing is portable; touching happens at home. Mechanically it is
the `hacking.PDA` component sharing the deck's net map, with read paths
(`TextFiles`, `PortReport`, `KnownHosts`) exposed sessionless; reading
a file fires `OnRead` exactly like `cat`.

For hardened corpo targets, it is valid for every externally visible port
to be closed at first. Buddy does not run a magic "firewall
breacher" program to force ports open. Instead, overworld actions create
legitimate-looking holes: starting diagnostics, plugging into an internal
maintenance VLAN, tricking someone into remote support, or physically
bridging forgotten hardware. The terminal reports the wall; Buddy changes
the world so a service becomes reachable.

Some hosts can also declare a fake username and password. A configured
username requires `ssh username@host`; bare `ssh host` is rejected before
password entry. `-p 22` may follow either target form. A password-required
line changes the prompt to `password: `. The next line is compared against
both the attempted username and the service or host's content-declared
password. Either credential being wrong prints `Permission denied, please
try again.` and leaves Buddy on the current host. Passwords are intentionally
simple puzzle words, not real authentication. Hosts without an authored
username retain bare-host SSH syntax until their accounts are declared.

For the first Microslop beat, the intended setup is:

- The barista was fired from Microslop. Her old badge is clipped to the
  backpack behind the coffee-shop counter, with username `jane_doe` and the
  simple password on a
  post-it behind the laminated card. Buddy can meow for an invited look or
  sneak past her; `look post-it`, `examine post-it`, and `read post-it`
  work. She never speaks the password.
- Buddy's deck stays in the lair; the PDA carries the discovered credential
  home through the shared notes view.
- Microslop's SSH service is directly reachable from the lair. After the
  post-it look, Buddy returns home, runs `ssh jane_doe@microslop`, connects
  with `apple`, and searches for
  employee ID `1008476` to locate the layoff plans.

## Hooks — the progression surface

Content attaches flag names to files and processes:

- `OnRead` — set when the file is first `cat`ed or `grep`-matched.
- `OnCopy` — set when the file lands on the deck.
- `OnKill` — set when the process is killed.
- `OnRun` — set when the executable is run.
- `OnSend` — set when the file is `send`-uploaded to the contact's drop
  (the delivery beat; marduk's reply message keys on the same flag).

Each hook sets a World flag. Dialogue, room descriptions, checks, and
Buddy's notes already read flags, so `cat`-ing the right log can change
what the barista says. Discovery texture comes from reading: hostnames are
found inside files and `curl` indexes, not given away.

## Ingress — how a host falls (#23, design skeleton)

The net must not be a one-trick pony: a single skeleton key is boring
the second time. Instead, *how* a host falls tells you who runs it, and
which door is open to you depends on the character you built. Hosts are
either **sloppy** or **hardened**, and the four ingress types split
cleanly along the game's central axis — deck is player-skill, body is
character-skill (see the top-of-file rule).

**Sloppy hosts expose credentials through carelessness:**

- **Known** — a person tells you a password; you type it. Knowledge moves
  through the player's head.
- **Leaked** — you *find* the credential the owner left lying around:
  in a shell's environment (`env`), a config, a committed secret in a
  repo mirror, or a post-it tucked behind a physical badge. Terminal leaks
  reward `ls -a` / `grep` / `env` recon; physical leaks use ordinary
  overworld access and checks. This is the easy credential and it is
  *supposed* to stop at the door of anyone competent.

**Hardened hosts do not leak. Their lock lives in meatspace, and there
are two doors — the two stat builds (character skill, seeded checks in
the overworld, never in the terminal):**

- **Insider (Charm)** — an NPC who remembers you warmly opens the
  route. Gated on NPC-memory flags: soften them and the route just
  opens; burn them and this door shuts. (Ties the NPC-memory system to
  network access; see docs/systems/dialogue.md.)
- **Heist (Stealth + Agility)** — you throw the physical switch
  yourself in the overworld (the records-office switch / maglock shape
  already built for Okuda). The no-friends-needed hard path.

**Failure is a fork, per the ruling.** Burning an insider does not wall
a hardened host — it *demotes* you from the Charm door to the heist. A
player who dumped both stats *and* burned the insider is the one
self-inflicted dead end we allow (same license as any XCOM build).

**Soft-lock discipline:** a hardened host on the **critical path** must
keep *both* doors reachable, so no build is stranded. Off-critical-path
hosts may be single-door and punish a narrow build; that is fair.

**Characterization for free:** Microslop is careless (the badge post-it,
secrets in env, committed keys) → falls to scavenging. Okuda was paid to
lose records and assumes it is being hacked → its core does not leak, so
`env` is useless there and you must come through a person or a heist.

Everything above runs on existing machinery — `OnRead`/`OnCopy` flags,
overworld checks under the XCOM rule, NPC-memory flags, `Host.Require`
route gating. The one net-shape decision this implies is **depth**:
`env`-style leaks only matter as a *pivot* — you stand on a reachable
box (a bastion) to reach an interior host you could not touch from
outside. A flat hub-and-spoke net needs none of this; a layered net
does. Depth is the open call (flat vs bastion layer); the taxonomy
above holds either way. Content (which hosts, which NPCs, an `env`
command + per-host var table) is TBD and declared by the author.

The **net map** is a requirement, not a nicety, the moment the net
gains a second layer: even the author cannot hold the graph in his
head, so a player has no chance. It renders as a discovered-hosts tree
in `~/notes/notes.md` (already Markdown through Glamour) — branches
fill in as you snoop, an empty branch reads as a to-do. Recon becomes
cartography; no diagram renderer needed.

## The messenger

The deck runs an instant messenger (user ruling 2026-07-09,
docs/draft.md): the Resistance-lite mission-giver's voice, and the
answer to what the terminal's right panel is for. `messenger` (alias
`talk`) toggles it in the right panel.

- **One contact for the draft** — `marduk`, the mission-giver (handle
  tentative, user ruling 2026-07-09) — but messages are authored per
  contact (`hacking.Messenger` on the `Deck` component), so later
  senders (a sentinel taunting mid-hack, corp spam) slot in without a
  rewrite.
- **The game boots into the terminal** (`main` calls
  `Model.BootIntoDeck`) with marduk's mission brief waiting as the
  first unread — typing `messenger` is the first thing the game
  teaches. The overworld intro text greets the first logout instead.
- **Reading a brief downloads its mission** (`Msg.Grants`, user ruling
  2026-07-10): a mission message carries a flag set when it is read,
  and a mission file gated on that flag (`Node.PresentWhen`) appears in
  `~/notes` at that moment — no `download`/`cp` to type, it just shows
  up. See "Missions" below.
- **Delivery closes the loop**: `send <file>` uploads a deck-resident
  file to marduk's drop; the file's `OnSend` flag makes his reply
  arrive in the same session. Mission 1 ends exactly here.
- **The thread is derived, never stored** (the journal pattern): a
  message has arrived exactly when its `When` flag is true. Arrival is
  a consequence of play — copy the notice, and the channel lights up —
  never a timer (`turns.md`); save/load reconstructs the thread for
  free. The only stored state is a per-message read marker
  (`msg_read_<id>`, system-owned).
- **Unread surfaces in the scrollback**: a dim `[messenger] N unread`
  line when a message lands (including at login). Opening the panel
  reads the whole thread. Flags set mid-session deliver immediately —
  the messenger is the one place the world talks back before logout.
- **Replies come later**: the plan (docs/draft.md) is to reuse the
  dialogue system — a tree rendered as chat — so the contact gets NPC
  memory for free. Not built yet; the panel is read-only today.

## Buddy's notes

The player-facing notes surface is a file on the deck, not a menu. Buddy keeps
`~/notes/notes.md`, a Markdown-flavored text file generated from flags.
Players read it with `cat ~/notes/notes.md` and can search it with
`grep -ir <pattern> ~/notes`. Notes are written as things Buddy has found
or inferred, with vague clue texture allowed, but not explicit next-step
instructions.

## Missions

Missions are their own files in `~/notes` (user ruling 2026-07-10;
`notes.md` was a stopgap and is now just field notes). A mission file
**auto-downloads** when its handler's briefing is read: the briefing
message's `Msg.Grants` flag flips, and the mission file — declared with
a matching `Node.PresentWhen` — appears in `~/notes`. No `download` or
`cp` to type; it just shows up, and mirrors onto the PDA the same
instant (`TextFiles` honors `PresentWhen`).

The file holds the mission params and a checklist that ticks `[x]` as
its flags land (mission 1: `Microslop_Find_The_Layoff_List.md`).
Filenames are underscored, no spaces, so `cat` reads them without
quoting. Mission 1 is the loud end of the hint-fade (docs/draft.md):
its steps read like orders; later missions get vaguer files. This is
also where a mission's objectives live — there is no separate
quest-panel objective list (the terminal's right panel idles blank).

**Mission 0 — the bootstrap note.** `~/notes/use_the_messenger.md` is
present from the very first boot and teaches the one thing the player
needs to start everything: open the messenger. It uses `AbsentWhen`
(the inverse of `PresentWhen`) to vanish the instant the messenger is
read — the same flag that downloads mission 1. So the tutorial hands
off cleanly: read the note → open the messenger → note disappears,
mission 1 appears. This is what answers "how does the player know to
type `messenger`?"

`Node.PresentWhen` / `Node.AbsentWhen` are general: a node is absent
from `ls`/`cat`/`grep`/PDA until `PresentWhen` is set and again once
`AbsentWhen` is set — a clean way to make files appear (and disappear)
as the story turns flags on.

## Logging in

A deck in scope appears in the left panel (the hub-travel panel) under
a green `// UPLINK` subhead, set apart from the city districts to read
as a log-in target rather than a place you walk to. Shift+Tab focuses the
panel, arrows move, Enter on the deck row opens the terminal — the same
navigation as hub travel. `DecksInScope` includes visible decks and the
deck Buddy carries, so the terminal is available outside the lair when
the story wants physical plug-in beats. (Typing `log in` / `use deck`
still works as an alias path.)

**The uplink row follows Buddy** (user ruling 2026-07-10): when no
deck is in scope — anywhere but the lair — the `// UPLINK` subhead
offers **the PDA** instead, and Enter thumbs it awake (the same menu
`use pda` opens). The slab yields to the real terminal: the panel
never offers both.

## Screen layout

Full-screen swap while a session is live (the three-panel room UI and
LOG are hidden, not destroyed — closing the terminal restores them
exactly):

- **Terminal** (~60% width when the reader is active, otherwise dominant):
  bordered, titled `CYBERDECK // <host>`, with scrollback bottom-anchored
  above the in-panel prompt.
  The deck renders `paws_in_the_machine@deck:path $` in CRT green. A successful
  SSH connection renders the configured `username@host:path $` and recolors
  the complete main terminal pane amber; returning to the deck restores green.
The right panel is modal and **idles blank** (user ruling 2026-07-10:
no objectives, no status, no game state — just a dim screen-saver
mark). It stays green even while the main terminal is amber. It's claimed
wholesale by the reader, the editor, or the
messenger — whichever the player brought up last (opening a document
closes the messenger and vice versa) — and returns to idle when they
close.

- **Reader/editor panel** (~40% width when active): opens when `cat`
  reads a Markdown or text file, rendering Markdown with Glamour and text
  as wrapped plain output. `edit` opens the same panel as a multiline text
  editor. The terminal scrollback keeps the typed command plus a short
  `opened <path> in reader` or save notice instead of duplicating the full
  document. Shift+Tab moves focus between the terminal and right panel; in
  the editor, Shift+Tab autosaves and returns focus to the terminal. Up/Down
  scroll the focused reader or move the editor cursor. When the terminal has
  focus, Tab completes commands, aliases, hosts, and fake-filesystem paths,
  while Up/Down recalls the session's command history. Passwords are masked
  and excluded from history. The Shift+Tab and Tab bindings are shown in the
  right panel's idle and active footers.
- Narrow terminals keep the terminal usable first; the reader panel
  shrinks. Widths are clamped — no negative-width rendering.

## Later

- Credential gates on `ssh` beyond simple passwords (a host requiring a key
  file present on the deck).
- ICE as processes that fight back; traces; disconnect pressure.
- XP payouts for completed intrusion beats (content-assigned per
  `stealth.md` — hacks are played, not rolled).
- More commands as puzzles need them (`mv`, `rm`, `chmod` ...).
