# Rough Draft — the whole game, no Resistance

> Ruling (2026-07-09): draft the full game — overworld + hacking,
> basic quest structure — with **no Resistance faction**. NPCs are the
> only obstacles. Purpose: flesh out the overworld, systems, and
> puzzles; find where the systems fall short; and let the draft reveal
> where Resistance (#34) naturally inserts rather than designing it on
> a blank page.
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

*(declare here)*

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
