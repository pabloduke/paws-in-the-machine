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

1. Start at the terminal (the lair, the deck).
2. Tutorial — **mission 1 is the tutorial** (declared 2026-07-09; no
   separate practice host). The teaching lives in the notes: early
   hints are explicit step-by-step orders — go to the barista, look
   at the badge, return to the deck, scan ports — written into
   `~/notes/notes.md` as each flag lands (built machinery). See
   section 4 for the hint-fade rule.
3. Resistance accepts Buddy → Mission 1: break into Microslop.
4. Resistance hint, recorded in notes: the barista at the local
   coffee shop was laid off from Microslop.
5. **[rework]** The badge peek (declared 2026-07-09). The barista's
   old Microslop badge is clipped to the backpack she keeps behind
   the counter. The badge is laminated, and tucked behind the card is
   a post-it with her employee ID and the password. Buddy doesn't
   steal anything — he reads the post-it and the info lands in his
   notes on its own (flag → generated `~/notes/notes.md`, which the
   PDA mirrors live; all built machinery). She never speaks the
   password — this **replaces** the current dialogue reveal of
   `apple` (rewrites her tree + `TestMicroslopPasswordPuzzle`). Two
   routes to the badge, Okuda-shaped:
   - **Charm her**: she sets out a saucer of milk behind the counter,
     right next to the backpack — Buddy is invited in, the peek is
     unwatched. (Charm works on *her*, never on the backpack — the
     fire-escape rule.)
   - **Sneak past her**: uninvited, behind the counter while she
     works — a real stealth check; the existing hound-distraction /
     lure circumstances apply.
   - Failure fork: caught nosing at her backpack → `barista_burned`,
     charm route goes cold, stealth stays open at a penalty. Never a
     wall.
   - The employee ID rides along for later: candidate hook — the
     layoff document is findable *by* her ID in Microslop's
     filesystem (search her ID, find her own termination record).
6. Physical route — the flowchart skipped this, the draft keeps it:
   past the corpo hound into the back room, patch the deck in at the
   server rack (`microslop_route_open`). Best overworld↔terminal
   weave in the game so far; the password alone must not be enough.
7. Scan Microslop; connect (built).
8. Explore the filesystem; locate the layoff plans. **[new]** the
   layoff document itself — lands beside `/srv/archive/
   sun_notice.txt`, and the barista's firing sets it up. "Internal
   network" stays one host for mission 1; if a later mission wants
   depth behind it, that's the #42 net-depth question waking up.
9. Retrieve (copy home) the layoff document.
10. **[new]** Return the document to the Resistance — the delivery
    mechanic Resistance-lite exists for. Payoff beat: the Resistance
    recruits the affected engineers.

## 2. City map

Hubs and rooms. Existing: the lair (deck lives here, only here), the
neighborhood, the plaza, the coffee shop (+ back room), Okuda HQ
(lobby → corridor → records office → Deep Archive annex).

Constraints: buildings deepen rather than the city sprawling
(hubs.md ruling); every hack target should have a physical location
the overworld work happens in (scan from the lair → do the legwork →
come home and jack in).

*(declare here)*

## 3. Target ladder

The hackable hosts, roughly easiest → hardest, and **how each one
falls** — that's the ingress taxonomy's menu (docs/systems/hacking
ingress notes, #23/#42): straight hack of a sloppy host / charm an
insider / stealth heist. Existing hosts: microslop (route + password,
via barista + back room), okuda.grid (port opened from the records
office), sunfarm.arc and undernet.relay (no routes in yet).

Constraints: vary the ingress type across the ladder; expect the
open net-depth question (#42) to demand an answer around the third
target — that's the draft doing its job, bring it back here.

*(declare here)*

## 4. Quest structure

How the player knows what to do next. Draft rule: **use only the
three mechanisms that exist** — deck `Objectives` (quest-panel hints
that clear on flags), the generated `~/notes/notes.md`, and the
event journal. No new quest system in the draft; where these creak
is exactly the data #34 needs.

**Hint-fade rule (declared 2026-07-09):** missions 1–3 are the
tutorial ramp; there is no separate tutorial content. Mission 1's
notes read like a checklist (go to the barista, look at the badge,
return to the deck, scan ports); missions 2 and 3 teach too, with
less and less guidance each; from mission 4 on, notes are just
intel, not instructions.

*(declare here — the quest chain as a flag list: what sets each,
what it unlocks)*

## 5. NPC roster

Per hub: who gates what, and what they remember. The pattern is the
barista (dialogue.md "State And Memory"): per-NPC memory flags
(softened/burned), memory-aware greeting/description, favors refused
to a cruel stray — and never walling a critical path behind a burned
bridge.

Existing: the barista (microslop intel), the corpo hound (back-room
gate), the Okuda receptionist (front-desk charm fork).

*(declare here)*

## Falls-short log

Running list, filled in as the draft is built: places a system
creaked, an authoring step felt gamey, or a mechanism was missing —
plus Resistance insertion points as they show themselves (who
recruits Buddy, which hack draws attention, what arrives on the
deck).

*(empty)*
