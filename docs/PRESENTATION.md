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
  the presentation layer. The alternate `chrome` theme uses cold silver,
  cyan, and magenta styling.

The existing presentation-only `Model.phase` clock remains the animation
source. Chrome title shimmer obeys the same rule as rain: a tick may advance
presentation phase and nothing else.

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

## Runtime command

The hacking terminal intercepts `theme` as a machine-local UI command before
input reaches the fake shell. It works while connected to a remote host but
never becomes a capability of that host. When the shell is waiting for a
password, interception is disabled and the text remains a credential.

```text
theme                    show the current theme and usage
theme list               list registered themes; `*` marks the current one
theme next               select the next registered theme
theme previous           select the previous registered theme
theme <id>               select a theme directly
```

Tab completion merges `theme` and its arguments with the shell's existing
completion results without adding presentation concerns to the hacking
system. The shell's `help` output gets a UI-owned local-display footer.

Selection currently lasts for the running UI session. It is not written to
game saves or world state.

## Current boundary

The presentation layer and runtime selection are implemented. Two themes are
registered: `wet-neon` and `chrome`. Theme command response strings are marked
`(Placeholder)` for the user to replace or approve.

The `chrome` theme adds a truecolor gradient and a narrow specular highlight
that sweeps across the overworld room sign and the hacking terminal's main
title. The effect is an inline, grapheme-aware renderer: it preserves the raw
title, display width, alignment, and surrounding geometry at every phase.
Wet-neon keeps its original title rendering. Gradient endpoints, highlight
color and width, direction, and period are isolated as provisional theme data
for later visual tuning. The renderer remains safe under terminals that reduce
truecolor to a smaller color profile, although gradients will carry less
detail there.

Markdown rendered by Glamour keeps its existing renderer-owned style for now.
If themes need to control Markdown later, that renderer must receive a
theme-specific style rather than post-processing its ANSI output.

## Next work

- [x] Add at least one alternate theme.
- [x] Add a local `theme` terminal command for listing, selecting, and cycling
  registered themes.
- [ ] Decide whether to keep the current session-only selection or store it in
  a separate UI preferences file.
- [x] Add theme-owned title gradient, highlight, and shimmer settings.
- [ ] Add spatial shadow and glow effects where layout allows them.
- [ ] Add reduced-motion and simplified-Unicode presentation options.
- [ ] Route Markdown styling through the active theme if alternate themes
  require it.

These are cosmetic presentation changes. They do not change player-visible
game logic or state transitions and therefore do not require a
`docs/GAME_FLOW.md` update.
