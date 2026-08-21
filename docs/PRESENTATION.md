# Presentation layer

Status: initial semantic text and runtime theme foundation implemented.

## Purpose

The presentation layer keeps game content independent from terminal styling.
The engine, systems, and authored content produce plain text. The UI records
why a piece of text exists, selects the corresponding style from the active
theme, and asks Lipgloss to render the final ANSI output.

```text
plain game text
    -> semantic UI role
    -> active presentation theme
    -> Lipgloss layout and styling
    -> ANSI terminal frame
```

Lipgloss remains the renderer. The presentation layer does not replace its
color-profile fallback, ANSI-aware dimensions, borders, padding, alignment,
or layout.

## Invariants

- Engine and world state never contain themes, styles, or ANSI sequences.
- Transcript history stores raw text plus a semantic role; styling happens
  when the current frame is rendered.
- Changing a theme is presentation-only and cannot advance turns, set flags,
  move entities, or otherwise touch the world.
- The overworld and hacking terminal keep separate style vocabularies inside
  one selected application theme.
- Theme definitions are immutable after registration.
- The shipped `wet-neon` theme reproduces the appearance that existed before
  the presentation layer.

The existing presentation-only `Model.phase` clock remains the animation
source. Future shimmer or glow animation must obey the same rule as rain: a
tick may advance presentation phase and nothing else.

## Semantic text

`internal/ui/presentation.go` defines transcript elements as raw text plus a
small semantic role:

- body/output;
- player echo; and
- dim presentation narration.

The vocabulary should grow only when two pieces of text need meaningfully
different treatment across themes. It must not encode particular story lines
or inspect prose for keywords.

The hacking terminal already keeps raw scrollback and an entry kind
separately. Its rendering now resolves colors through the selected theme as
well.

## Themes

A registered presentation theme owns:

- overworld styles, including panels, titles, selections, signs, and rain;
- the local terminal palette;
- the remote terminal palette; and
- the terminal side-panel palette.

`ui.New` accepts `ui.WithTheme("<id>")` to inject a registered initial theme.
Unknown IDs fall back to `wet-neon`. `ThemeIDs` returns registered IDs in
stable order. Runtime switching rebuilds cached rendered views from their raw
semantic data, so existing transcript and terminal history do not retain the
old theme's ANSI colors.

## Current boundary

This first slice establishes the layer and routes the existing UI through it.
It intentionally adds no new visual theme and no player command yet. The only
registered theme is `wet-neon`.

Markdown rendered by Glamour keeps its existing renderer-owned style for now.
If themes need to control Markdown later, that renderer must receive a
theme-specific style rather than post-processing its ANSI output.

## Next work

- [ ] Add at least one alternate theme.
- [ ] Add a local `theme` terminal command for listing, selecting, and cycling
  registered themes.
- [ ] Decide whether the selected theme lasts only for the session or is
  stored in a separate UI preferences file.
- [ ] Add theme-owned gradient, highlight, shadow, glow, and shimmer settings.
- [ ] Add reduced-motion and simplified-Unicode presentation options.
- [ ] Route Markdown styling through the active theme if alternate themes
  require it.

These are cosmetic presentation changes. They do not change player-visible
game logic or state transitions and therefore do not require a
`docs/GAME_FLOW.md` update.
