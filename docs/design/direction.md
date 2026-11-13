# Amorae: The shared book

## Concept

Amorae is set like a book the two of you keep together: one paper, one iron-gall ink, one plum rubric. In old prayer books the rubric (from the Latin for red) was the second ink. It marked what to do, never the prayer itself. Here plum does the same job: actions, references, the one thing to tap. The app talks in a quiet sans. Anything a person wrote (a prayer, a request, a question, a memory) is set in a text serif, so your own words never look like interface. Sections are separated the way a book separates them: with rules, a change of paper, and the type. Dark mode is the same book read by lamplight. It is a second printing on umber stock, not an inverted screen.

**What survives, and why.** The linen and plum palette, the warm dark mode (not blue-grey), the grain, the hairline-ruled rows, the 24px gutter and 44px targets, one token name for both themes, and the restrained copy. **What makes it generic.** Manrope sets everything, headlines included, which gives it the SaaS look. Uppercase tracked labels head every section. Home stacks four or five identical bordered 16px `bg-surface` cards. Every list is the iOS Settings pattern of icon, text and chevron, and that includes the Our story feed. History filters are pills. Photos show as 40px thumbnails. `bg-celebrate` puts a gradient on an everyday card.

## Palette

Only values change. The names are the existing `globals.css` tokens, and the aliases follow their targets (`bg-elevated`→surface, `fg`→ink, `fg-muted`→stone, `border`→line, `accent`→plum, `accent-hover`→plum-dark, `accent-fg`→surface, `danger*`→red*, `success*`→green*).

| Token | Role | Light | Dark |
| --- | --- | --- | --- |
| `--color-bg` | page stock | `#F5F1EA` | `#191512` |
| `--color-paper` | second stock: tone bands, prayer mode | `#ECE5D9` | `#211C18` |
| `--color-surface` | the one raised leaf; text on plum | `#FCFAF5` | `#28221D` |
| `--color-ink` | iron-gall ink | `#261F1C` | `#EFE7DB` |
| `--color-stone` | secondary text | `#645A51` | `#ADA193` |
| `--color-line` | hairline rules | `#DCD2C4` | `#38312A` |
| `--color-edge` | controls and borders (≥3:1) | `#8A7F72` | `#776B5E` |
| `--color-faint` | track, disabled fill | `#E2D9CB` | `#3A332C` |
| `--color-photo` | image placeholder | `#E4DACB` | `#2E2822` |
| `--color-plum` | rubric: actions, refs, filled button, active tab, progress | `#5B2A4A` | `#B4899F` |
| `--color-plum-dark` | pressed / emphasis | `#44203A` | `#CBA6BA` |
| `--color-plum-tint` | rare wash, initials | `#EFE3E8` | `#3B2634` |
| `--color-green` / `-tint` | prayed, answered | `#3B5E44` / `#E1E7DA` | `#9DC8A4` / `#213128` |
| `--color-amber` / `-tint` | warning | `#77500F` / `#F2E5CA` | `#E3B66B` / `#382C17` |
| `--color-red` / `-tint` | error, destructive | `#963829` / `#F4E2DC` | `#EE9C8E` / `#3E231E` |
| `--color-gold` | occasions only | `#836226` | `#DDBE7E` |

Grain is unchanged: noise at 0.055 opacity in light, 0.04 in dark.

Dark plum is a dusty mauve, deliberately deeper and greyer than the brand's light plum tint. The filled button, active tab rule and label, progress segments and links all read the same `--color-plum`, so they stay one family. On dark `faint` it measures 4.15, so plum is never text on `faint`.

### Measured contrast (WCAG 2.x relative luminance, computed)

| Pair | Light | Dark |
| --- | --- | --- |
| ink / bg · paper · surface | 14.40 · 12.95 · 15.54 | 14.80 · 13.77 · 12.81 |
| ink / plum-tint · faint | 12.99 · 11.60 | 11.32 · 10.13 |
| stone / bg · paper · surface | 5.97 · 5.37 · 6.45 | 7.17 · 6.67 · 6.21 |
| stone / plum-tint · faint (disabled btn) | 5.39 · 4.81 | 5.48 · 4.91 |
| plum / bg · paper · surface · plum-tint | 9.95 · 8.95 · 10.74 · 8.97 | 6.07 · 5.64 · 5.25 · 4.64 |
| plum-dark / bg · plum-tint | 12.32 · 11.12 | 8.38 · 6.41 |
| surface on plum (primary button label) | 10.74 | 5.25 |
| surface on ink (toast) | 15.54 | 12.81 |
| green / green-tint · bg | 5.80 · 6.50 | 7.32 · 9.70 |
| amber / amber-tint | 5.73 | 7.25 |
| red / red-tint · bg | 5.79 · 6.44 | 6.68 · 8.45 |
| red / surface · paper | 6.96 · 5.80 | 7.32 · 7.86 |
| bg on ink (selected segment) | 14.40 | 14.80 |
| gold / bg (large type only) | 4.99 | 10.14 |
| edge / bg · paper · surface (non-text, ≥3) | 3.48 · 3.13 · 3.76 | 3.50 · 3.26 · 3.03 |

Every text pair passes 4.5:1. `line` (1.33 light, 1.42 dark) is decorative only and never the sole boundary of a control.

## Typography

- **Newsreader** (Production Type, SIL OFL 1.1, Google Fonts). A variable text serif with an optical-size axis (opsz 6–72, wght 200–800, roman + italic). Self-host the latin variable woff2, roman and italic, from the Google Fonts/`productiontype/Newsreader` release or the Fontsource variable package. Set `font-optical-sizing: auto`.
- **Manrope** (already self-hosted, OFL). Kept because it matches the wordmark. From now on it is only the interface voice.
- **Cormorant Garamond** stays scoped to occasions.

| Token | Size / line height | Face | Use |
| --- | --- | --- | --- |
| `--text-micro` | 12 / 1.4, 600 | Manrope | the floor: uppercase folios (date, "No. 41", weekday under a date) and tab-bar labels only |
| `--text-support` | 14 / 1.5 | Manrope | meta, bylines; entry kind ("Memory · Ada") and goal labels at 600 |
| `--text-body` | 16 / 1.55 | Manrope | UI copy, rows, buttons (600) |
| `--text-bodylg` | 18 / 1.45 | Newsreader, 400, ink | the couple's words in lists: requests, notes, entries, prayer-point titles on Home |
| `--text-reading` | 21 / 1.6 | Newsreader | prayer text |
| `--text-section` | 22 / 1.25, 500 *italic* | Newsreader | section heads ("Today's prayer") |
| `--text-title` | 30 / 1.15, 450, −0.015em | Newsreader | month chapters, prayer titles (34 in prayer mode) |
| `--text-display` | 40 / 1.08, 400, −0.02em | Newsreader | screen titles, greeting (italic 300 stone for the salutation) |

Nothing is set below 12px, and 12px is only for uppercase folios and tab labels.

Numerals that carry meaning (countdowns, entry dates) use Newsreader with `lining-nums tabular-nums`. The countdown figure is the one deliberate exception to the scale, at 64px weight 300, in ink.

Tailwind `--text-*` tokens can't carry a family. The rollout therefore adds `--font-text` to `@theme inline` and puts `font-text` on `Title`, `Section`, `Scripture` and the prayer components.

## Navigation: the foot of the page

The tabs, in `nav-items.ts` order, are Home, Together, Prayers, Our story (formerly History) and Settings. Tab roots, Our story included, have no back button. A display title is their masthead.

- **Phone (`TabBar`).** The bar is the page's foot. It uses the same `bg` and grain as the page, with one `line` hairline above. It has no fill, blur, float or shadow. Each tab is a 64px target, about 76px wide, above `--safe-bottom`, with the icon (existing set) over a 12px label. The **active tab** has three cues, only one of them colour: a plum ribbon bookmark (12×15, swallowtail) hanging from the hairline, an ink label at 750 weight, and a 1.9 icon stroke. Inactive tabs are `stone` at 600 weight with a 1.5 stroke. There is no centre "+"; adding stays in each screen's top bar.
- **Desktop, `lg` (`SideNav`).** The side nav is the contents page. From the top it holds the wordmark, the couple's names as a folio, five 48px entries, and an italic colophon at the foot ("Kept together since…"). The active entry is a `paper` row with an ink label at 750 weight, and the ribbon turns sideways, pointing in from the left edge. This replaces today's `bg-plum-tint` pill.
- **Rail, `md`.** The phone tab, stood on end, with the ribbon on the left edge.

The ribbon measures 9.95:1 against `bg` in light and 6.07:1 in dark.

## Our space (Together hub)

The hub is a contents page. Groups are italic section heads, each followed by plain ruled rows with a live detail line and a chevron, and no leading icons. Our story sits between the groups as a chapter opening (ink rule, title, latest entry), because it is read, not used.

## Space, radii

Spacing uses the 4px base, in practice 4, 8, 12, 16, 24, 32, 40 and 48. The screen gutter is 24px (prayer mode text 32px). Gaps between sections on a screen are 32px, and 40px before a month chapter. Every touch target is at least 44px. Primary buttons are 54px.

Radii drop to the crispness of paper: `--radius-input` 8, `--radius-btn` 10, `--radius-card` 6, `--radius-sheet` 20, and photos 2.

**One new token: `--scrim`.** In light it is `color-mix(in srgb, var(--color-ink) 40%, transparent)`; in dark it is `color-mix(in srgb, var(--color-bg) 72%, transparent)`. Today the sheet's `bg-ink/40` turns into a cream veil in dark mode.

## Components

`mockups/components.html` shows every shared piece in both themes, each labelled with the component in `components/ui/*` to restyle. Two components are new and need extracting:

- **`Tabs`**, from the history filters.
- **`Segmented`**, from five inline radiogroups. Their pills are 36px tall today, below the 44px minimum.

Labels, hints and errors move up to 14px.

## Separating things without cards

1. **Rules first.** Hairlines (`line`) separate rows. A 1.5px `ink` rule opens a chapter (a month). In the story feed, rules start at the text column, not the screen edge.
2. **Tone shift.** A band of `paper` running full-bleed marks a different kind of object (the sealed question, the goals strip, prayer mode).
3. **One leaf per screen, at most.** `surface` with a 1px `line` border and no shadow, kept for the thing someone is waiting on.
4. **Type does the rest.** Serif against sans, italic section heads, and a date set in the left margin.

## Icons

Keep the existing hand-drawn 1.6-stroke set in `components/icons.tsx`. Icons go in navigation and controls only: tab bar, close, add, bell, previous/next. Feed and list rows lose their leading icon. A small word ("Memory · Ada") names the kind instead. No icon sits in a tinted circle. Status marks (prayed / not yet) stay.

## Motion

Keep the existing ones: `rise` for page entry (0.2s), `sheet`, and `draw-check` on "I've prayed". Add nothing. Nothing loops on an everyday screen. Reduced motion stays as it is.

## Don't

- Use gradients on everyday surfaces (retire `bg-celebrate`), or glow, blur, or soft shadows.
- Use uppercase tracked labels as section headings. Those are folios, not titles.
- Use pills for filters. Use underlined tabs.
- Use plum for decoration. If it isn't an action, a reference or progress, it's ink.
- Use emoji, sparkles, or illustrations standing in for content.
- Put more than one raised leaf on a screen, or stack bordered cards.
- Set the couple's own words in the sans, or the interface in the serif.
- Shrink a photo to a thumbnail in the story.

## Round 3: what changed and why

- **Countdown numeral: plum to ink.** Plum there was decoration, which broke this document's own rule.
- **Working-on goals: paper tiles to ruled columns.** The tiles were the one place where cards had crept back in.
- **Photo placeholder: coloured clip-art to a grainy duotone.** The clip-art read as a template.
- **Entry kind: weight 600 to 500.** At 600 it competed with the title beneath it.
- **Menu bar: generic strip to the page's foot with a ribbon.**

Kept: the seal, the date margin, the single leaf and the type scale. They hold up.

The anniversary's foil, glow and keepsake gradients sit outside this direction. Each needs its own review against the rules above.
