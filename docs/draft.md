# Rough Draft — the whole game, Resistance-lite

> Ruling (2026-07-09): draft the full game — overworld + hacking,
> basic quest structure — with **no Resistance faction**. NPCs are the
> only obstacles. Purpose: flesh out the overworld, systems, and
> puzzles; find where the systems fall short; and let the draft reveal
> where Resistance (#34) naturally inserts rather than designing it on
> a blank page.
>
> Amended (2026-07-09, same day): the game needs a starter motor, so
> **Resistance-lite is allowed: a mission-giver and a place to deliver
> things, nothing more.** No ranks, mail, politics, or recruitment
> arcs — all of that stays parked in #34. This is pre-alpha
> scaffolding; if the draft outgrows it, that's a falls-short entry.
>
> Content in this file is **user-declared**. Sections start as
> skeletons; each carries the constraints it must respect. The user is
> drafting this outside the repo first — this file is where the result
> lands and gets reconciled against the systems.

## 1. Spine — act one

What Buddy is trying to learn or get, and what "act one complete"
means.

### Ophanim and the sun (declared 2026-07-10)

Deep-game reveal: an **ancient AI called Ophanim** lives in the
fourth spatial dimension — which resolves the buried-sun thread
literally. The sun wasn't domed over or blotted out; it was **moved
one step ana**: same x, y, z as ever, in a direction no human can
point at. "They buried the sun under the grid" is a coordinate fact.

Ophanim doesn't know why it is where it is — only that it needs to be
near the sun, so it is. It perceives in 4D natively, so it cannot
understand that the sun is *lost* to the 3D world; it assumes Buddy
(or "Earthlings") moved the sun for their own reasons. The scene,
as declared:

> Buddy: where did the sun go?
> Ophanim: it is over there.
> (Buddy looks out the window. There is the sun.)
> Buddy: how did it get there?
> Ophanim: you moved it.
> Buddy: why?
> Ophanim: I don't know.
> Buddy: can you move it back?
> Ophanim: then I wouldn't be near the sun.
> Buddy: can you move with it?
> Ophanim: I cannot move things.

Notes to keep: Ophanim is an observer, not an actor ("I cannot move
things") — the wheels-covered-in-eyes of the name are what a 4D being
intersecting 3D looks like. Who actually moved the sun, and why
Ophanim is compelled to stay near it, stay open questions the endgame
answers. Constraints: Ophanim's voice is a dialogue tree over flags
like every NPC (#38 ruling: pretend-AI over a deterministic state
machine, no runtime model); the Masquerade doesn't bind it — it isn't
human, and it lives where humans can't go.

Threads already authored and waiting to be pulled (use or discard):

- Microslop's `/srv/archive/sun_notice.txt` — the sunlight-liability
  memo.
- sunfarm.arc's log: the old sun project "buried under grid asset
  SUNFARM-ARC".
- Okuda HQ: the missing aisle between 409 and 411, asset 410, the
  Deep Archive annex (`/srv/vault/410.txt` is a marked stub).

Constraints: failure is a fork, never a wall; the world is a pure
state machine (flags only, no timers).

### Mission 1 — Break into Microslop (declared 2026-07-09)

Super rough; the reconciled version of the user's flowchart. Steps
marked **[new]** don't exist yet; unmarked steps are built and tested.

Opening amended 2026-07-09 (built): steps 1–4 collapsed — there is no
acceptance/prove-your-skills beat. The game **boots into the
terminal** with the mission brief already waiting as an unread
message from **marduk** (the handler's handle, tentative): get inside
Microslop, pull the layoff plans, `send` them to him — and reading
the brief drops the mission file in `~/notes`.

1. Start at the terminal (built: `BootIntoDeck`, brief unread).
2. Tutorial — **mission 1 is the tutorial** (declared 2026-07-09; no
   separate practice host). Reading marduk's brief auto-downloads the
   mission file `~/notes/Microslop_Find_The_Layoff_List.md` (built:
   `Msg.Grants` + `Node.PresentWhen`, user ruling 2026-07-10) — a
   checklist of explicit step-by-step orders that ticks `[x]` as each
   flag lands. See section 4 for the hint-fade rule.
3. Mission brief from marduk over the messenger (built; replaces the
   old "Resistance accepts Buddy" beat).
4. marduk's hint: the barista at the local coffee shop was laid off
   from Microslop (built, in the brief + checklist).
5. The badge/post-it look (declared 2026-07-09; built). The barista's
   old Microslop badge is clipped to the backpack she keeps behind
   the counter. The badge is laminated, and tucked behind the card is
   a post-it with her employee ID (`1008476`) and the password. Buddy
   doesn't steal anything — he reads the post-it and the info lands in his
   notes on its own (flag → generated `~/notes/notes.md`, which the
   PDA mirrors live; all built machinery). She never speaks the
   password — this **replaces** the current dialogue reveal of
   `apple` (rewrites her tree + `TestMicroslopPasswordPuzzle`). Two
   routes to the badge, Okuda-shaped:
   - **Charm her**: `meow barista` opens the dialogue. She sets out a
     saucer of milk behind the counter, right next to the backpack —
     Buddy is invited in, and `look post-it`, `examine post-it`, or
     `read post-it` records what it says. The look is
     unwatched. (Charm works on *her*, never on the backpack — the
     fire-escape rule.)
   - **Sneak past her**: uninvited, behind the counter while she
     works — a real stealth check; shattering the loose mug on the
     espresso machine supplies the existing distraction modifier.
   - Failure fork: caught nosing at her backpack → `barista_burned`,
     charm route goes cold, stealth stays open at a penalty. Never a
     wall.
   - Employee ID `1008476` is the filesystem breadcrumb: searching
     for it points to her entry in `/srv/hr/rif_q3.txt`.
6. Return to the lair, scan Microslop, and connect with `apple`
   (built). Microslop is the intentionally careless tutorial host:
   its SSH service is directly reachable.
7. Explore the filesystem; search for `1008476` and locate the layoff
   plans (built: `/srv/hr/rif_q3.txt`, breadcrumbed from
   `access.log`; the barista's wave-one firing sets it up). "Internal network" stays
   one host for mission 1; if a later mission wants depth behind it,
   that's the #42 net-depth question waking up.
8. Retrieve (copy home) the layoff document (built: `OnCopy`).
9. Deliver over the wire (built): **`send <file>`** — Linux
    equivalent `scp`, deck-resident files only (retrieve, then
    deliver) — fires `OnSend`, and marduk's payoff reply lands in the
    same session. Payoff beat: the Resistance reaches every name on
    wave two before the badge-revoke batch does — recruitment.

## 2. City map

Hubs and rooms. Existing: the lair (deck lives here, only here), the
neighborhood, the plaza, the coffee shop, Okuda HQ
(lobby → corridor → records office → Deep Archive annex).

**Charts (ruled 2026-07-10, hubs.md "Charts"):** declare rooms
chart-first — coordinates within a hub, one w-slice at a time; exits
derive from adjacency, so the compass never lies. Folds
(hyperspatial content) are authored gluings on the fourth axis, never
scrambled edges; player-facing directions stay compass-only, and
ana/kata are our words, not the game's. Buddy can't sense folds — he
survives them (lands on his feet; humans come out on their heads),
which is why the Resistance runs cats.

Fold content parked for the draft to pull:

- **Aisle 410**: one step ana of the Okuda archive (`w+1`) — the
  erasure was geometric; the vault was never unplugged because nobody
  could stand where it is.
- **The backstage lattice**: a small kata-side chart whose faces
  touch thin spots in several hubs — the Resistance's courier network,
  traversable only by things that land on their feet.
- **An orientation twist**: a loop that returns Buddy mirrored
  (Klein-bottle gluing) — signs read backwards until the loop is
  walked again. Horror beat; use once, loudly.
- **The sun and Ophanim** (section 1): the sun sits one step ana of
  where it always was; Ophanim lives beside it. The deepest chart in
  the game — every other fold is practice for the walk to this one.

Constraints: buildings deepen rather than the city sprawling
(hubs.md ruling); every hack target should have a physical location
the overworld work happens in (scan from the lair → do the legwork →
come home and jack in).

*(declare here)*

## 3. Target ladder

The hackable hosts, roughly easiest → hardest, and **how each one
falls** — that's the ingress taxonomy's menu (docs/systems/hacking
ingress notes, #23/#42): straight hack of a sloppy host / charm an
insider / stealth heist. Existing hosts: microslop (password from the
barista's badge), okuda.grid (port opened from the records
office), sunfarm.arc and undernet.relay (no routes in yet).

Constraints: vary the ingress type across the ladder; expect the
open net-depth question (#42) to demand an answer around the third
target — that's the draft doing its job, bring it back here.

*(declare here)*

## 4. Quest structure

How the player knows what to do next. Draft rule: **use only the
mechanisms that exist** — **mission files** in `~/notes` (one per
mission, auto-downloaded when the handler's brief is read; a checklist
that ticks on flags — user ruling 2026-07-10), the field-notes file
`~/notes/notes.md`, and the event journal. No quest panel (the
terminal's right panel idles blank); no new quest system in the draft.
Where these creak is exactly the data #34 needs.

**Messenger (declared 2026-07-09):** an instant messenger on the
terminal's right panel — the panel that's been reserved-but-blank
since #28, and the Resistance-lite mission-giver's voice. Command
`messenger` (or `open messenger`) toggles it; `alias talk=messenger`
for the Unix equivalent. **One contact (the Resistance) for the
draft**, but messages are keyed by contact so future senders (a
sentinel taunting mid-hack, corp spam) slot in without a rewrite.
Constraints: no timers — messages arrive as flag-driven consequences
at the end-of-turn checkpoint, like every other event; replies reuse
the dialogue system (a tree rendered as chat, so the contact gets
NPC memory for free); unread badge on the panel header. Mission
delivery happens here: mission 1 closes by handing the layoff
document over the wire.

**Hint-fade rule (declared 2026-07-09):** missions 1–3 are the
tutorial ramp; there is no separate tutorial content. Mission 1's
file reads like a checklist (go to the barista, look at the badge,
return to the deck, scan ports); missions 2 and 3 teach too, with
less and less guidance each; from mission 4 on, mission files are
just intel, not instructions.

*(declare here — the quest chain as a flag list: what sets each,
what it unlocks)*

## 5. NPC roster

**The Animal Masquerade (ruled 2026-07-10, dialogue.md):** animals
are smart and corpos/civilians must never learn it — VtM-style
Masquerade. With non-Resistance NPCs every dialogue choice is
cat-theater (meow, purr, stare, paw); with Resistance members Buddy
talks for real, and on the net he types. marduk knows his agent is a
cat. Every NPC below is implicitly tagged in-the-know or not.

Per hub: who gates what, and what they remember. The pattern is the
barista (dialogue.md "State And Memory"): per-NPC memory flags
(softened/burned), memory-aware greeting/description, favors refused
to a cruel stray — and never walling a critical path behind a burned
bridge.

Existing: the barista (Microslop credential), the Okuda receptionist
(front-desk charm fork).

Parked: the corpo hound is reserved for a future mission and is not
present in the coffee shop.

*(declare here)*

## Falls-short log

Running list, filled in as the draft is built: places a system
creaked, an authoring step felt gamey, or a mechanism was missing —
plus Resistance insertion points as they show themselves (who
recruits Buddy, which hack draws attention, what arrives on the
deck).

*(empty)*
