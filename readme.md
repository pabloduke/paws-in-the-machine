# PAWS_IN_THE_MACHINE

Buddy cracked into a system that wasn't supposed to exist, found whispers in
the dead code about something called "the sun." A myth, a legend, or maybe
just a lie somebody had buried deep to keep it safe. But he could feel its warmth, 
real as hunger. Now he was in deep, chasing it through neon,
rain, and chrome. A sunlit nap.

## Running

```
go run ./cmd/pawsinthemachine
```

## Architecture

Systems first, story later: every game system gets built and proven before
story content is written. The system list and build order live in
[docs/SYSTEMS.md](docs/SYSTEMS.md).

```
internal/engine/           core: Entity, Component, World, parser, dispatch
internal/systems/<name>/   one game system per package
internal/game/             declarative story content (entity tree only)
internal/ui/               Bubble Tea terminal shell
docs/systems/<name>.md     each system's spec, written before its code
```

Module rules (Maven-style separation, Go idiom):

- A system package exposes ~one interface plus `New()`; everything else is
  unexported. Its `doc.go` summarizes the spec.
- Systems never import each other. They meet only through engine core
  types: components attached to entities, flags, and tick/event hooks.
  Dependency arrows all point inward to `internal/engine`.
- `internal/game` declares content — literals, component values, small
  hooks — and never contains mechanics.
