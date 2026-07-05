# Project State

Current branch: `terminal`

## Current Focus

The game is building toward a cat-hacker loop where Buddy uses normal room
actions, dialogue, stealth, and a fake terminal shell together. The current
beat is the first Microslop Corp terminal puzzle.

## Implemented

- Dedicated hacking terminal mode exists and replaces the normal room UI while
  active.
- The terminal has a fake shell with simplified commands including `ls`, `cd`,
  `pwd`, `cat`, `grep`, `cp`, `mkdir`, `touch`, `scan`, `ssh`, `curl`, `ps`,
  `kill`, `run`, `exit`, and `help`.
- Buddy's deck is now carried, so `use deck` / `log in` works outside the lair.
- SSH hosts can require a route flag and/or a fake password.
- Hosts can declare fake services/ports, `scan <host>` lists configured port
  states, and `ssh <host> -p <port>` targets SSH on a specific port.
- `grep -ir` searches recursively and case-insensitively through fake host
  filesystems.
- The hacking terminal has a terminal-only green phosphor CRT look.
- Microslop is currently gated by:
  - social intel from the barista, who was fired from Microslop and reveals the
    password `apple`;
  - physical access in the coffee shop back room, reached by bypassing the
    corpo hound;
  - using the backroom server rack to patch the deck into the local network.
- Once connected, Microslop has a small fake filesystem with logs pointing to
  `/srv/archive/sun_notice.txt`.
- Reading and copying the Microslop notice set flags that feed the hacking
  discoveries/objective panel.

## Recent Verification

`go test ./...` passes after the Microslop route/password flow changes.

## Next Useful Steps

- Make the social-engineering path less linear: add alternate ways to learn or
  infer `apple`.
- Add clearer in-game hints when `ssh microslop` reports network unreachable.
- Give the backroom/rack interaction a stronger stealth consequence or tension.
- Expand Microslop's filesystem into a real mini puzzle instead of one log and
  one archive file.
- Decide whether terminal history/discoveries should persist across deck
  sessions or reset per login.
