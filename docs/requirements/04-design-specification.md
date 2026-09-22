# Amorae: Design Specification

| | |
|---|---|
| **Document ID** | AMR-REQ-04 |
| **Version** | 3.0 |
| **Status** | Draft for review |
| **Owner** | Design |
| **Last updated** | 2026-09-22 |

---

## 1. Purpose, scope and sources of truth

*(Normative)*

This document specifies how Amorae looks, behaves and reads: design tokens, typography,
components and their states, the screen inventory, interaction and motion, state patterns,
accessibility requirements, and UX writing. It does not repeat product rationale ([`01-product-requirements.md`](./01-product-requirements.md)),
feature behaviour ([`02-functional-requirements.md`](./02-functional-requirements.md)), or
operating targets ([`03-non-functional-requirements.md`](./03-non-functional-requirements.md),
categories `NFR-A11Y`, `NFR-COMPAT`, `NFR-PERF`). Decisions and open
questions are the single register at [`05-decisions-and-open-questions.md`](./05-decisions-and-open-questions.md);
this document cites `DEC-NN` and `Q-NN` rather than restating them.

**Sources of truth, in order.** When this document and the code disagree, the code wins, and this
document is corrected in the next revision:

1. `apps/web/src/app/globals.css` — the actual design tokens shipped to the browser.
2. [`../BRAND.md`](../BRAND.md) — palette, logo and mark rules.
3. This document — the narrative around those tokens: principles, components, screens, states,
   writing.

This is `DEC-14`: `globals.css` and `BRAND.md` are the source of truth for visual tokens, and the
v2.0 values this document used to carry (a different background hex, "Inter or Manrope", a
charcoal manifest theme) are superseded. Section 3.5 lists every value v2.0 got wrong against the
code as it now stands.

**Every claim in this document about what is built** — a component, a state, a screen, a
behaviour — was checked against the code in `apps/web/src/components/ui/`, `apps/web/src/components/layout/`,
and the route tree under `apps/web/src/app/`. Where the code does not yet do something this
document says it should, that requirement is marked **Gap** rather than described as existing.

**Sections are marked Normative or Informative.** A Normative section states a requirement:
designers and engineers build to it, and a deviation is either a bug or needs a documented
decision. An Informative section explains context, history, or intent, and carries no
independent requirement.

**Scope.** Covers the installed, mobile-first PWA (`apps/web`), portrait phone width first. Does
not cover Go API design (see [`../API.md`](../API.md)) or infrastructure (see
[`../ARCHITECTURE.md`](../ARCHITECTURE.md)).

---

## 2. Design principles

*(Normative)*

Every design principle below ties to a core principle in [01 PRD §6](./01-product-requirements.md).
Both are checked before a design or engineering decision ships.

| Design principle | PRD principle it serves |
|---|---|
| **Human-designed, not generated.** Restraint and intention over decoration. If an element does not aid understanding, orientation, or tone, remove it. | Useful before decorative |
| **Warm minimalism.** Generous whitespace, strong typography, restrained borders, calm rhythm. | Warm, not childish |
| **Shared, not competitive.** Everything reads as "ours." No leaderboards, streak pressure, or guilt. | Intentional, not addictive; A two-person experience |
| **Tactile and native.** The installed app should feel closer to a native iPhone app than a website. | Designed for real life |
| **Typography carries the design.** Use type hierarchy, not ornament, to create structure. | Useful before decorative |
| **Faith invites, never pressures.** Prayer is a full pillar with no streaks, guilt copy, or religious iconography. | Faith without pressure (`DEC-13`) |
| **AI stays out of the way when off.** Any AI surface must degrade to a fully usable screen with AI disabled. | Human before AI |

**Avoid:** excessive gradients, glassmorphism, glowing blobs, floating 3D, neon, generic AI
dashboards, deep card nesting, huge hero sections, unnecessary illustrations, cliché romance
imagery, repeated heart icons, religious clip-art, and playful gamification. This is checked
against the built UI in Section 7 (components) and Section 8 (screens): none of the shipped
screens use these patterns as of this document date.

---

## 3. Design tokens

*(Normative — token tables and contrast table. The "Changed from v2.0" table in §3.5 is
Informative: it explains history, not a current requirement.)*

Use the CSS custom properties below by name in components; never a raw hex or pixel value. They
are defined once, in `apps/web/src/app/globals.css` under `@theme`, and read into Tailwind 4 as
utility classes (for example `bg-plum`, `text-stone`, `rounded-card`).

**No dark-mode values exist in code.** `globals.css` has no `@media (prefers-color-scheme: dark)`
block, and none of the tokens below carry a dark variant. `BRAND.md` documents exactly one
dark-mode value ("Plum tint (dark mode) `#E9D5E1`") with no accompanying dark background, surface
or text palette, so it cannot be used to build a working dark mode on its own. Treat dark mode as
**Not started**; whether it belongs in MVP is `Q-21` (recommendation: light only for MVP).

### 3.1 Colour

| Token | Light value | Dark value | Usage |
|---|---|---|---|
| `--color-bg` | `#f7f4ef` | — | Page background; PWA manifest `background_color` and `theme_color` (`DEC-14`) |
| `--color-paper` | `#f2ede5` | — | Secondary background: Prayer Mode, `/offline`, full-screen error boundaries |
| `--color-surface` | `#fffdfa` | — | Card and input fill |
| `--color-ink` | `#24201f` | — | Primary text. `BRAND.md` also calls this the dark-mode surface colour, but no dark palette is wired into CSS |
| `--color-stone` | `#6b635c` | — | Secondary text, sub-lines, placeholder text |
| `--color-line` | `#e4ddd3` | — | Default hairline border and divider |
| `--color-edge` | `#8f857b` | — | Input and secondary-button borders, unfilled status marks, unchecked switch track |
| `--color-faint` | `#ddd4c8` | — | Skeleton fill, disabled fill, progress-bar track |
| `--color-photo` | `#eae3d9` | — | Photo/frame placeholder background |
| `--color-plum` | `#5b2a4a` | — | Primary accent: primary buttons, active nav item, links, focus ring |
| `--color-plum-dark` | `#47203a` | — | Accent pressed/hover (alias `--color-accent-hover`) |
| `--color-plum-tint` | `#f1e8ee` | — | Accent tint background, e.g. `Initial` avatar fill |
| `--color-plum-soft` | `#b59aab` | — | Mid-tone plum; e.g. visited-but-unprayed dots in the Prayer Mode progress strip |
| `--color-green` | `#3f6b52` | — | Positive/success text and icon |
| `--color-green-tint` | `#e4ede6` | — | Positive background: `Banner`, `Alert` success, `StatusMark` done, `Checkbox` checked |
| `--color-amber` | `#7f5210` | — | Warning text and icon |
| `--color-amber-tint` | `#f6ebd6` | — | Warning background: offline `Banner` |
| `--color-red` | `#9b3b32` | — | Error/destructive text, icon, input border |
| `--color-red-tint` | `#f6e7e4` | — | Error background: `Alert` error |

**Legacy alias layer** (kept so older template components keep working; new components should use
the names above): `--color-bg-elevated` = surface, `--color-fg` = ink, `--color-fg-muted` = stone,
`--color-border` = line, `--color-accent` = plum, `--color-accent-hover` = plum-dark,
`--color-accent-fg` = surface (text on plum), `--color-danger` = red, `--color-danger-bg` =
red-tint, `--color-success` = green, `--color-success-bg` = green-tint.

### 3.2 Type scale

See Section 4 for the full typography rules. Token values, exactly as in `globals.css`:

| Token | Size | Line-height | Letter-spacing | Weight | Usage |
|---|---|---|---|---|---|
| `--text-micro` | 12px | 1.4 | 0.08em | 700 | Uppercase labels, timestamps, section labels |
| `--text-support` | 14px | 1.5 | normal | inherited (400) | Sub-lines on rows, supporting text |
| `--text-body` | 16px | 1.55 | normal | inherited (400) | Default body text; also the `<body>` base size |
| `--text-bodylg` | 17px | 1.55 | normal | inherited (400) | Larger reading paragraphs (Home intro copy) |
| `--text-reading` | 21px | 1.6 | normal | inherited (400) | Defined for long-form reading. **Gap:** not applied anywhere in the components read; Prayer Mode's prayer body text uses a hard-coded `text-[19px] leading-[1.6]` instead of this token (`prayer-session.tsx`) |
| `--text-section` | 20px | 1.3 | -0.015em | 600 | Section group headers |
| `--text-title` | 28px | 1.18 | -0.022em | 600 | Screen titles (`Title` default) |
| `--text-display` | 34px | 1.18 | -0.025em | 600 | Splash, big moments (`Title size="display"`) |

`Title` also has an untokenised `size="lg"` step (`text-[32px] leading-[1.15] font-semibold
tracking-[-0.025em]`) used nowhere in the token table above. **Gap:** this should either get its
own named token or be retired in favour of `display`.

### 3.3 Spacing

**No named spacing scale exists in `globals.css`.** Unlike the v2.0 spec's `space-0.5`…`space-16`
scale, the code uses Tailwind 4's built-in spacing utilities directly (`px-6`, `gap-3.5`, `pt-3`),
frequently with arbitrary bracket values for exact pixel heights (`h-[54px]`, `min-h-[76px]`).
There is no enforced 4pt/8pt rhythm beyond Tailwind's own default scale. Treat this as the current
state, not a target: a future revision could reintroduce named spacing tokens if drift becomes a
problem.

### 3.4 Radius, elevation and motion

| Token | Value | Usage |
|---|---|---|
| `--radius-input` | 12px | Inputs |
| `--radius-btn` | 14px | Buttons |
| `--radius-card` | 16px | Cards, rows with a card frame |
| `--radius-sheet` | 24px | Bottom sheets (top corners) |

Pills and circles (`Switch`, `Initial`, `StatusMark`, avatar fallbacks) use Tailwind's built-in
`rounded-full` utility directly; there is no dedicated `--radius-pill` token.

**Elevation: no shadow tokens exist.** `globals.css` defines no `--shadow-*` custom properties at
all. The v2.0 `elev-1`/`elev-2` box-shadow tokens were dropped, not implemented differently —
elevation in the shipped UI is hairline borders (`--color-line`) only, consistent with principle
"warm minimalism" but a stronger position than v2.0 described.

| Motion token | Definition | Usage |
|---|---|---|
| `--animate-breathe` | `breathe 1.8s ease-in-out infinite` (opacity 0.25 ↔ 1) | Skeleton pulse |
| `--animate-rise` | `rise 0.24s ease-out both` (fade + translateY 6px) | Banners, toast entrance |
| `--animate-page` | `rise 0.2s ease-out both` | `Main` content mount |
| `--animate-sheet` | `sheet 0.32s cubic-bezier(0.2,0.8,0.2,1) both` (fade + translateY 24px) | Bottom sheet entrance |
| `--animate-fade` | `fade 0.24s ease-out both` | Sheet scrim |
| `--animate-draw` | `draw 0.32s 0.08s ease-out forwards` (stroke-dashoffset) | Prayer completion checkmark draw-on |

The `.press` utility (`transition: transform 0.12s ease-out, filter 0.12s ease-out`; active state
`scale(0.98)` + `brightness(0.86)`) is the de facto tap-feedback timing across nearly every
interactive component, but it is not a named duration token — there is no shared `dur-fast` /
`dur-base` / `dur-slow` / `ease-standard` scale as v2.0 specified. See Section 9 for the full
motion picture, including `prefers-reduced-motion` behaviour.

### 3.5 Changed from v2.0 (Informative)

| v2.0 said | Current position (code) |
|---|---|
| `warm-white` background `#F6F4EF` | `--color-bg` `#f7f4ef` (`DEC-14`) |
| "Inter for UI and body. Manrope is an acceptable alternative for display." | Manrope only, self-hosted variable font, used for both body and display (`DEC-14`). `BRAND.md`'s typography line was corrected to match in this revision. |
| Manifest theme colour: deep charcoal (per `DEC-14`'s account of the old value) | `#F7F4EF`, matching the background (`manifest.ts`, `docs/BRAND.md`) |
| Named 4pt spacing scale, `space-0.5`…`space-16` | No named spacing tokens; Tailwind's default utilities used directly, often with arbitrary pixel values |
| `radius-sm` 8 (inputs), `radius-md` 12 (cards and buttons) | `--radius-input` 12, `--radius-btn` 14, `--radius-card` 16 — buttons and cards no longer share one radius |
| `radius-lg` 16 (sheets) | `--radius-sheet` 24 — sheets now have their own, larger radius |
| `radius-xl` 20 (Prayer Mode surfaces) | Dropped. Prayer Mode uses the `paper` background colour, not a rounded surface |
| `elev-1` / `elev-2` box-shadow tokens | Dropped entirely; no `--shadow-*` tokens in `globals.css` |
| `dur-fast`/`dur-base`/`dur-slow` + `ease-standard` shared scale | Replaced by named `--animate-*` composites, each with its own duration and easing (see 3.4) |
| Palette: `charcoal`, `stone-light`, `hairline`, `berry` (secondary accent), `clay` | `ink` (renamed, hex shifted), `stone-light` and `hairline` dropped (no code equivalent), `berry` dropped (no secondary accent exists), `clay` → `red` |
| `plum-soft` = light accent tint `#F0E7EC` | Renamed `--color-plum-tint` (`#f1e8ee`). **Naming collision:** code has a *different* `--color-plum-soft` (`#b59aab`), a mid-tone value not present in v2.0 |
| No tint-pair tokens for status colours | New: `--color-paper`, `--color-photo`, `--color-red-tint`, `--color-green-tint`, `--color-amber-tint` |
| Type scale: page title 26/32, section title 19/26, body 15/24, caption 13/18 (weight 500, 0.04em tracking) | `--text-title` 28px, `--text-section` 20px, `--text-body` 16px (deliberately ≥16px, see §4), `--text-support` 14px (renamed "caption" → "support"); `--text-micro` keeps 12px but weight is 700 (not 500) and tracking is 0.08em (not 0.04em) |
| Invite code example `AB12-CD34` | `ABC-123` (`DEC-08`) |

### 3.6 Contrast

Computed against the token hex values above (relative luminance, WCAG 2.x formula). Every text
pair passes **4.5:1**. One non-text pair fails its 3:1 minimum (`plum-soft`, below).

| Pair | Ratio | Result |
|---|---|---|
| ink `#24201f` on bg `#f7f4ef` | 14.71:1 | Pass (4.5:1) |
| stone `#6b635c` on bg `#f7f4ef` | 5.37:1 | Pass (4.5:1) |
| stone `#6b635c` on surface `#fffdfa` | 5.80:1 | Pass (4.5:1) |
| stone `#6b635c` on paper `#f2ede5` | 5.06:1 | Pass (4.5:1) |
| accent-fg (surface `#fffdfa`) on plum `#5b2a4a` | 11.03:1 | Pass (4.5:1) |
| plum `#5b2a4a` on bg `#f7f4ef` | 10.21:1 | Pass (4.5:1) |
| plum `#5b2a4a` on surface `#fffdfa` | 11.03:1 | Pass (4.5:1) |
| red `#9b3b32` on red-tint `#f6e7e4` | 5.70:1 | Pass (4.5:1) |
| green `#3f6b52` on green-tint `#e4ede6` | 5.11:1 | Pass (4.5:1) |
| amber `#7f5210` on amber-tint `#f6ebd6` | 5.70:1 | Pass (4.5:1) |
| red `#9b3b32` on bg `#f7f4ef` (field error text) | 6.24:1 | Pass (4.5:1) |
| amber `#7f5210` on bg `#f7f4ef` | 6.14:1 | Pass (4.5:1) |
| stone `#6b635c` on plum-tint `#f1e8ee` | 4.91:1 | Pass (4.5:1) |
| plum `#5b2a4a` on plum-tint `#f1e8ee` | 9.34:1 | Pass (4.5:1) |
| green `#3f6b52` on bg `#f7f4ef` | 5.57:1 | Pass (4.5:1) |
| edge `#8f857b` on bg `#f7f4ef` (border/icon, non-text) | 3.30:1 | Pass (3:1 non-text minimum only — do not use `edge` for text) |
| plum-soft `#b59aab` on bg `#f7f4ef` (Prayer Mode progress dots, non-text) | 2.34:1 | **Fail** (below the 3:1 non-text minimum, SC 1.4.11). **Gap:** the dot state must also be conveyed by shape or fill, or the colour darkened to reach 3:1 |
| **Dark-mode equivalents** — only pair computable from documented values | | |
| plum-tint (dark accent) `#e9d5e1` on ink `#24201f` | 11.58:1 | Pass (4.5:1) — theoretical only; no dark background/surface token exists to pair it with beyond `ink` itself |
| surface `#fffdfa` on ink `#24201f` (hypothetical dark body text) | 15.89:1 | Pass (4.5:1) — hypothetical; `ink` as a dark background and `surface` as dark-mode body text are not implemented anywhere in code |

No text pair in the built light-mode palette falls below 4.5:1. `--color-edge` is a border/icon
colour (3.30:1, meeting only the 3:1 non-text threshold) and must never be used for text; no
component read does so. `--color-plum-soft` fails even the non-text threshold and must not be
the only signal of a state.

---

## 4. Typography

*(Normative)*

**Typeface:** Manrope only, self-hosted as a variable woff2 font, loaded via `next/font` and
exposed as `--font-manrope` (see `--font-sans` and `--font-display` in `globals.css`, both of
which resolve to it). One family for the whole product — body, UI and display all use Manrope.
No Inter anywhere in the shipped app (`DEC-14`). The wordmark itself is drawn as an SVG outline and needs no
font (`BRAND.md`).

**Scale:** see Section 3.2 for the full token table. Display and titles carry negative tracking
(-0.015em to -0.025em, tightening as size increases); body sizes are untracked.

**Input font size is always ≥16px.** `globals.css`: `input, textarea, select { font-size: max(16px, 1em); }`.
This is deliberate — it is the one thing standing between the product and iOS Safari zooming the
whole page on focus. Any new form control must not override this downward.

**Tabular numbers:** the `.tabular` utility (`font-variant-numeric: tabular-nums`) is applied
wherever digits must not shift width as they change — day-of-month in `DateRow`, prayer index in
`PrayerRow` and `PrayerSession`, the reminder-time display, the timer label. Use it on any new
numeric readout that updates in place.

**Line length and hierarchy:** body and body-lg text run inside `Main`'s `px-6` padding on a
520px-max column (Section 6), which keeps measure well under the ~75-character guideline on all
supported widths. Hierarchy is carried by size and weight, never by colour alone: `Para` renders
secondary-toned text (`text-stone`) by default even at body size, so colour is a texture, not a
substitute for a heading.

---

## 5. Iconography and imagery

*(Normative)*

**Icons.** A single custom outline set (`apps/web/src/components/icons.tsx`), 24×24 grid, 1.6px
stroke by default (`strokeWidth` is a prop; `TopBar`/`ErrorState` sometimes use 1.4 for a lighter
large glyph), round caps and joins (`strokeLinecap="round" strokeLinejoin="round"`), no fill. This
is close to, but not identical to, the v2.0 spec's "1.5px stroke" — the shipped default is 1.6.
About 30 icon names are defined (`home`, `together`, `book`, `clock`, `check`, `calendar`, `pin`,
`target`, `flag`, `image`, `bookmark`, `bell`, `alert`, `logout`, `trash`, `pencil`, `moon`,
`offline`, `sync`, and others); there is no repeated heart glyph anywhere in the set, matching
`DEC-13`/principle "warm, not childish".

**Avatars.** `Initial` renders a circular (`rounded-full`) plum-tint fallback with the partner's
first letter — no photo-avatar component exists yet; every avatar in the read code is an initial.

**Photos.** No stock imagery anywhere in the app. `--color-photo` (`#eae3d9`) is reserved as the
placeholder/frame background for user photos (memories); the memories screen itself is Mock only
(Section 8) so the actual photo-frame treatment could not be verified against a real image upload
— treat photo framing inside `radius-card` as a **Gap** to confirm once Memories is built against
the real API (`Q-06` covers where photos are stored).

**Empty and illustrative art.** `Ghost` (in `states.tsx`) is the shipped answer to "minimal or no
illustration": four ghost-content variants (`lines`, `frames`, `bars`, `dots`) built from plain
divs and opacity steps, not drawings. No SVG illustrations or empty-state art exist in the
component set. This matches the old spec's "prefer warm words over illustration."

**Logo and mark.** Full rules live in [`../BRAND.md`](../BRAND.md) — clear space (half the mark's
width), minimum size 16px, heavier drawing below 32px (handled automatically by `<Mark>`), plum on
warm background or soft white on plum, no gradients/shadows/outlines, never rotated or placed
inside a heart or ring. `DEC-13`: no religious iconography in the logo or navigation — confirmed in
the built icon set (Section 5 above) and the five-item tab bar (Section 6): none of the shipped
icons are crosses, praying hands, doves, or similar. The mark itself renders via `<Mark>`
(`apps/web/src/components/icons.tsx`) and is used at 56px on the
splash view, 48px on `/offline`, and favicon/home-screen sizes per `BRAND.md`'s asset table.

---

## 6. Layout and responsive

*(Normative)*

**Frame.** `Screen` (`components/layout/screen.tsx`) is the outer shell for every screen: a single
column, `mx-auto`, `max-w-[520px]`, filling `min-h-[100dvh]` — the dynamic viewport unit, so the
layout survives the iOS Safari/PWA chrome resizing under keyboard or scroll. Minimum supported
width is 320px (iPhone SE class); the layout is single-column throughout, so nothing needs to
reflow at that width.

**Safe areas.** `:root` defines `--safe-top` / `--safe-bottom` from `env(safe-area-inset-*)`.
`SafeTop` reserves `calc(var(--safe-top) + 42px)` above content on screens that draw their own top
area; the `pt-safe` utility adds `calc(var(--safe-top) + 12px)`; the `pb-safe` utility adds
`max(var(--safe-bottom), 16px)`. The tab bar pads its bottom with
`max(var(--safe-bottom), 16px)` directly. `body` pads left/right insets so content never sits
under a landscape notch; `viewport-fit=cover` (set in `app/layout.tsx`) is what lets pages draw
under the status bar/home indicator in the first place.

**Navigation structure.** `TabBar` (`components/layout/tab-bar.tsx`) is a five-item bottom nav —
Home, Together, Prayers, History, Settings — sticky at the bottom, active item in bold plum
(`aria-current="page"`), inactive in `text-stone`. It hides itself while the on-screen keyboard is
open (`visualViewport` shrink of more than 150px). `AppShell` wraps every signed-in screen except
Prayer Mode (which gets its own full-bleed chrome, no tab bar, no `NetworkBanner`) in `Screen` +
`NetworkBanner` + the page + `TabBar`.

**Content width and one-hand reach.** The 520px max column keeps content centred and readable on
larger phones/tablets without becoming a dashboard. Primary actions sit low on the screen:
`BottomActions`/`Bottom` pin CTAs above the safe-area/tab bar, and `LinkButton`/`Button` are full
width at a 54px (primary) or 44px (secondary/text) minimum height, so the primary action on any
screen is reachable without a grip shift. Row-based lists (`Row`, `DateRow`, `PrayerRow`,
`SettingRow`) are the default content pattern over dashboards or grids, matching principle "warm
minimalism."

---

## 7. Component specification

*(Normative)*

One row per exported component, grouped by file. "Target size" is checked against two bars: the
44×44px design standard (iOS/Android platform guidance) and the WCAG 2.2 minimum of 24×24px
(Success Criterion 2.5.8). A component below 44px but at or above 24×24 is flagged; below 24×24
would be a Gap (none found). Where the code has no visible state for something this document
requires, the row says **Gap**.

### 7.1 Buttons and actions — `components/ui/buttons.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Button` | Primary interactive action | `primary` (h-54, plum fill), `secondary` (h-54, edge border), `text` (h-11, plum label), `danger` (h-11, red label) | default, pressed (`.press`: scale 0.98, brightness 0.86), disabled (opacity-100 override — `bg-faint`/`text-stone`, not just faded), loading (`aria-busy`, spinner replaces icon, implicitly disabled), focus-visible (global plum outline, `globals.css`) | Native `<button>`; accessible name = visible label (always required, no icon-only instance in the codebase); 54/44px height meets 44×44 |
| `LinkButton` | Primary action that navigates | Same variants as `Button` | default, pressed. **Gap:** no loading/busy state — a slow navigation gives no feedback, unlike `Button` | Renders `<Link>` styled as a button; 54/44px height meets 44×44 |
| `Spinner` | Loading indicator inside `Button`/`ComposeBar` | `light` (on plum) / default (on surface) | Continuous spin, `prefers-reduced-motion` disables the animation globally | Decorative, no text alternative needed (always paired with `aria-busy` on the parent control) |

### 7.2 Typography — `components/ui/typography.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Micro` | Uppercase label/metadata | tone `stone` / `plum` | none (presentational) | Plain text node; relies on surrounding landmark for context |
| `Title` | Screen/section heading | size `title` (28px) / `display` (34px) / `lg` (untokenised 32px, see §3.2 gap); `as` selects `h1`/`h2` | none | Heading level is caller-supplied via `as`; **Gap:** nothing enforces one `h1` per screen — a screen that forgets to pass `as="h2"` for a secondary heading silently ships two `h1`s |
| `Para` | Body copy | size `body` / `lg` / `support` | none | Plain `<p>`; always `text-stone` by default (colour is a texture, not the only signal — see §11) |

### 7.3 Form fields — `components/ui/fields.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Field` | Labelled input with hint/error | trailing slot for an icon/action | default, focus (`border-plum` 1.5px + global outline), error (`border-red` 1.5px, message with alert icon, `role="alert"`), hint. **Gap:** no explicit disabled visual treatment beyond the browser default | `label htmlFor`/`id` pair, `aria-invalid`, `aria-describedby` to the error or hint node; 52px height meets 44×44 |
| `BareInput` | Minimal-chrome input for compose screens | `hideLabel` (label becomes `sr-only`) | default, error (same pattern as `Field`). No visible border — focus relies solely on the global focus-visible outline | Same labelling pattern as `Field`; height is caller-controlled via `className`, so target size must be checked per screen |
| `BareTextarea` | Minimal-chrome multi-line input | `hideLabel` | Same as `BareInput` | Same as `BareInput` |

### 7.4 Rows and lists — `components/ui/rows.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Row` | Generic list row (icon, title, sub, trailing) | renders `<Link>`, `<button>`, or static `<div>` depending on `href`/`onClick` | default, pressed, `last` (drops the divider) | min-height 62px exceeds 44×44; trailing chevron signals navigability when interactive |
| `Section` | Labelled group wrapper | — | — | `<section>` landmark with a `Micro` label |
| `StatusMark` | Prayed / not-yet-prayed indicator | `done` boolean | — | `role="img"` with an explicit `aria-label` ("Prayed"/"Not yet prayed" by default); status is carried by icon *and* colour, never colour alone |
| `Segments` | Segmented progress (e.g. weekly prayer) | `color`, `height` | — | `role="img"` `aria-label="{done} of {total} prayed"` — the count is in the label, not inferred from colour |
| `Bar` | Linear progress bar | — | animated width transition | `role="progressbar"` with `aria-valuenow/min/max` and `aria-label` |
| `Initial` | Avatar fallback (first letter) | `size` | — | `aria-hidden="true"` — always paired with adjacent visible name text in the screens that use it |
| `Switch` | On/off toggle | — | checked/unchecked, pressed | `role="switch"`, `aria-checked`, `aria-label`; 44px height (width 52px) meets 44×44 |
| `SwitchRow` | Labelled preference row wrapping `Switch` | `last` | — | Visible label text is also passed as the `Switch`'s `aria-label`, so the accessible name matches what's on screen |
| `SettingRow` | Settings list row (icon, label, value, chevron) | tone `ink`/`red` (destructive) | default, pressed | 54px height exceeds 44×44; destructive actions (`Delete account`) use `tone="red"` plus the word "Delete", not colour alone |
| `Checkbox` | Custom checkbox | — | checked/unchecked, pressed | `role="checkbox"`, `aria-checked`; 48px height exceeds 44×44; check state shown by icon + green-tint fill, not colour alone |

### 7.5 Chrome — `components/ui/chrome.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `TopBar` | Screen header: back, title, one right action | — | default, pressed | Back control always carries a visible text label next to the icon (never icon-only); 44px height meets 44×44 exactly |
| `ComposeBar` | Compose-screen header: Cancel / title / Done | — | default, `busy` (spinner replaces Done, disabled) | 44px height meets 44×44 exactly (no margin above minimum) |
| `BottomActions` | Sticky action area above the tab bar / home indicator | `safe` (adds safe-area padding) | — | Layout only |
| `Banner` | Inline status line (offline, synced) | tone `amber`/`green` | — | `role="status"`. **Gap:** no manual dismiss control on the component itself; visibility is driven entirely by app state (`NetworkBanner`) |
| `Sheet` | Bottom sheet / modal | — | open/closed | `role="dialog"` `aria-modal="true"` `aria-labelledby`; backdrop button has `aria-label="Close"`. **Gap:** no visible focus trap or explicit initial-focus/return-focus handling in this component — verify against NFR-A11Y before relying on it for a complex form |
| `Toast` | Transient confirmation | optional action | Auto-dismissed after 4000ms by `Toaster` | `role="status"` (implicit polite live region) |

### 7.6 Loading, empty and error — `components/ui/states.tsx`, `query-state.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Skeleton` | Text-shaped loading placeholder | `lines` count | Breathing opacity animation | `aria-hidden="true"`. **Gap:** nothing announces "loading" to assistive technology; a screen reader user gets silence until content or an error arrives |
| `Ghost` | Shaped loading placeholder | `lines`/`frames`/`bars`/`dots` | Static (dimmed steps) | `aria-hidden="true"` |
| `EmptyState` | No-data state | `ghost` kind, headline, text, one CTA | — | CTA is a normal `Button`/`LinkButton`, so it carries its own accessible name |
| `ErrorState` | Recoverable error | secondary action slot | — | `role="alert"` — announced automatically when it mounts, which is what makes it the correct component for a failed async result (§11) |
| `Alert` | Inline error/success line | `variant` error/success | — | `role="alert"` (error) or `role="status"` (success) |
| `QueryState` | The loading/empty/error/ready switch for a screen's data | — | Renders `Skeleton` while pending, `ErrorState` with offline-specific copy ("You're offline") when the query is paused, `ErrorState` with the server's message on failure, else the ready content | Central to Section 10; every screen that fetches data is expected to go through this component rather than hand-rolling its own loading branch |

### 7.7 Domain rows — `prayer-row.tsx`, `date-row.tsx`, `pick-row.tsx`, `scripture.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `PrayerRow` | One prayer point in a list | optional `note` (amber annotation) | default, pressed | min-height 76px exceeds 44×44; completion is `StatusMark` (icon + colour) |
| `DateRow` | Event/milestone row (big day number) | `past` (muted), `action`/`right` trailing | default, pressed | min-height 76px exceeds 44×44 |
| `PickRow` | Compose-screen row opening a native date/time picker | `type` date/time/text | default (empty vs filled styling) | Overlaid native `<input>` carries `aria-label` matching the visible label; the whole 54px row is one `<label>` tap target. **Gap:** a `type="text"` `PickRow` is a fully transparent input with no visible caret/border, so its focus state is easy to miss for sighted keyboard users |
| `Scripture` | Reference + verse | `faint` (lighter divider) | — | Presentational; no interactive state |

### 7.8 Onboarding — `onboarding-bits.tsx`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Bottom` | Sticky bottom action area for onboarding screens | — | — | Layout only |
| `Fact` | Read-only icon/title/text row (e.g. profile facts) | trailing slot | — | Presentational |
| `ShowButton` | Show/Hide toggle (e.g. reveal an invite code) | — | shown/hidden | `aria-pressed` reflects state; 44px height meets the minimum exactly |

### 7.9 Layout shell — `components/layout/*`

| Component | Purpose | Variants | States | Accessibility |
|---|---|---|---|---|
| `Screen` | Outer column for every screen | `tone` bg/paper | — | 520px max width, `min-h-[100dvh]` |
| `SafeTop` / `Main` | Top safe-area spacer / scrollable content wrapper | `Main` `pad` toggle | `Main` mounts with `animate-page` | Layout only |
| `SplashView` | Brand loading screen while the couple query resolves | — | — | `role="status"` region around the loading label |
| `Toaster` | Renders the current global toast | — | Auto-dismiss 4000ms | Delegates to `Toast` (`role="status"`) |
| `NetworkBanner` | Offline / just-synced status line | — | offline (amber), just-synced (green), hidden (online, nothing pending) | Delegates to `Banner` (`role="status"`); see §10 |
| `TabBar` | Five-item bottom navigation | — | active/inactive per tab, hidden while the keyboard is open | `nav aria-label="Primary"`, `aria-current="page"` on the active tab; each tab is ≥52px tall |
| `AppShell` | Signed-in shell: couple gate, network banner, tab bar | — | loading (`SplashView`), error with retry (`ErrorState`), unpaired (redirect to `/couple`), session-expired (redirect to `/login`), ready | Orchestrates the top-level state patterns in Section 10 |

**Component count:** 46 exported components/hooks-with-UI across 13 `components/ui` files and 6
`components/layout` files (excluding the `kit.tsx` barrel re-export and the non-visual `cx` helper).

---

## 8. Screen inventory

*(Normative — the table and status column. The reconciliation list and the Home/Together/Prayer
Mode notes are Informative background for that Normative table.)*

**Status column key.** Status here describes where a screen's data comes from. It is a looser
bar than the README §4 definition of done, which also requires automated tests; the web app has
none yet (NFR-MAINT, `Q-15`), so no screen is "Implemented" in that stricter sense.

- **Implemented** — the screen's primary data comes from the real Go API (or the screen has no
  data dependency at all and is fully built).
- **Partial** — the screen's primary data comes from an API surface that is still in development
  (the couples module — `PAIR`) or the screen mixes real and mock data.
- **Mock only** — the screen's primary data comes entirely from the in-browser mock
  (`NEXT_PUBLIC_API_MOCK`); the Go endpoint does not exist yet.

| ID | Route | Screen | Module | Required states | Status |
|---|---|---|---|---|---|
| SCR-01 | `/welcome` | Welcome | AUTH | — | Implemented |
| SCR-02 | `/register` | Sign up | AUTH | loading, error (field validation) | Implemented |
| SCR-03 | `/login` | Log in | AUTH | loading, error (field validation, session-expired banner, rate-limited — Gap, see §10) | Implemented |
| SCR-04 | `/couple` | Create or join couple | PAIR | loading, error | Partial |
| SCR-05 | `/join` | Join couple (enter code) | PAIR | loading, error, empty (invalid code) | Partial |
| SCR-06 | `/invite` | Invite partner (share code) | PAIR | loading, error | Partial |
| SCR-07 | `/notifications` (onboarding) | Notification permission | NOTF | — (native OS prompt) | Mock only (subscribe endpoint is mock) |
| SCR-08 | `/install` | PWA install guidance | PWA | — | Implemented |
| SCR-09 | `/` | Home | PRAY/EVT/GOAL (couple-gated) | loading, empty, offline, pending-sync | Partial |
| SCR-10 | `/together` | Together hub | EVT/CAL/GOAL/CHAL/JRNL/APPR/MEM/DATE | loading | Mock only |
| SCR-11 | `/together/calendar` | Couple calendar | CAL | loading, empty, error | Mock only |
| SCR-12 | `/together/events` | Events list | EVT | loading, empty, error | Mock only |
| SCR-13 | `/together/events/new` | Create/edit event | EVT | loading (edit), error (validation) | Mock only |
| SCR-14 | `/together/events/[id]` | Event detail | EVT | loading, error, not-found | Mock only |
| SCR-15 | `/together/goals` | Shared goals | GOAL | loading, empty, error | Mock only |
| SCR-16 | `/together/goals/new` | Create goal | GOAL | error (validation) | Mock only |
| SCR-17 | `/together/goals/[id]` | Goal detail | GOAL | loading, error, not-found | Mock only |
| SCR-18 | `/together/challenges` | Challenges | CHAL | loading, empty | Mock only |
| SCR-19 | `/together/journal` | Journal | JRNL | loading, empty, error | Mock only |
| SCR-20 | `/together/appreciation` | Appreciation | APPR | loading, empty, error | Mock only |
| SCR-21 | `/together/memories` | Memories | MEM | loading, empty, error | Mock only |
| SCR-22 | `/together/milestones` | Milestones and important dates | DATE | loading, empty | Mock only |
| SCR-23 | `/prayers` | Current prayer week | PRAY | loading, empty, waiting-for-partner, offline, error | Mock only |
| SCR-24 | `/prayers/[id]` | Prayer detail | PRAY | loading, error | Mock only |
| SCR-25 | `/prayers/mode` | Prayer Mode (distraction-free) | PRAY | loading, empty (no points → quiet prayer) | Mock only |
| SCR-26 | `/prayers/set` | Set this week's prayers | PRAY | loading, error (validation) | Mock only |
| SCR-27 | `/prayers/edit` | Edit a single prayer point | PRAY | error (validation) | Mock only |
| SCR-28 | `/prayers/done` | Prayer completion state | PRAY | — | Mock only |
| SCR-29 | `/history` | Prayer history | PRAY | loading, empty, error | Mock only |
| SCR-30 | `/history/[id]` | Prayer history week detail | PRAY | loading, error, not-found | Mock only |
| SCR-31 | `/settings` | Settings | ACCT | loading, error | Partial |
| SCR-32 | `/settings/profile` | Profile | ACCT | loading, error (validation) | Implemented |
| SCR-33 | `/settings/notifications` | Notification preferences | NOTF | loading, error | Mock only |
| SCR-34 | `/offline` | Cold-start offline page | PWA | — (static, precached) | Implemented |
| SCR-35 | `/privacy` | Privacy policy | — (no module defined; see note) | — | Implemented (content marked draft, `Q-17`) |
| SCR-36 | `/terms` | Terms | — (no module defined; see note) | — | Implemented (content marked draft, `Q-17`) |
| SCR-37 | (global) | Not found | — | — (cross-couple 404 looks the same as an unknown route, `DEC-19`) | Implemented |
| SCR-38 | (global) | Runtime error boundary | — | error (root `app/error.tsx` and `(app)/error.tsx`) | Implemented |

**Note on SCR-35/36:** the module code list in the [README](./README.md) (`AUTH`, `ACCT`, `PAIR`,
`PRAY`, `EVT`, `CAL`, `GOAL`, `CHAL`, `JRNL`, `APPR`, `MEM`, `DATE`, `NOTF`, `PWA`, `AI`) has no
entry for static legal pages. They are left without a module code rather than force-fit into one.

### 8.1 Reconciliation against the v2.0 inventory

**In the v2.0 inventory but not a route today:**

- *Splash / loading* (v2.0 #1) — not a URL. Implemented as `SplashView`, shown by `AppShell` while
  the couple query is pending (the loading state of Home and every signed-in screen). Cross-refer
  to §9 and §10 instead of the screen table.
- *Offline state* (v2.0 #35, generic) — split into the cold-start `/offline` page (SCR-34) and the
  in-app `NetworkBanner` state pattern (§10); there was never a single generic "offline screen".
- *Error state* (v2.0 #36, generic) — not one screen. Implemented as `ErrorState` (used inside
  `QueryState` on almost every data screen) plus the two error boundaries (SCR-38).
- *Empty states* (v2.0 #37, generic) — not one screen; `EmptyState` is used per-screen (§10).

**Routes that exist but were not in the v2.0 inventory:**

- `/(app)/dashboard` — a leftover from the starter template. `page.tsx` immediately `redirect("/")`
  and renders nothing. **Withdrawn** — not a UI screen, should probably be deleted rather than kept
  as a redirect stub, but that is an engineering cleanup question, not a design one.
- `/privacy`, `/terms` (SCR-35, SCR-36) — legal pages, added after v2.0 (`Q-17`).
- `/prayers/edit` (SCR-27) — v2.0's "Create/edit prayer" (#27) covered composing the week as a
  whole; the shipped app splits that into `/prayers/set` (the week) and `/prayers/edit` (one
  point).

### 8.2 Home

`HomeScreen` (`features/home/home-screen.tsx`) branches on the week's status rather than showing
one fixed layout:

- **"It's your week"** (I am this week's setter, week still a draft) — prompts to set the week's
  prayers, shows last week's completion and the reminder time.
- **"Your partner is setting this week's prayers"** (waiting) — offers "Enter quiet prayer"
  (`/prayers/mode?quiet=1`) instead of a progress view, plus links to history and reminder settings.
- **Normal week** — greeting, tonight's event (if any) in a bordered card, a "This week" section
  (prayer progress via `Segments`, next event, active goal), a single "Continue praying" / "Begin
  praying" / "Open prayer mode" CTA (label depends on completion), and the partner's prayer count.

This matches the old spec's intent ("prioritizes today, then this week, then one or two
meaningful actions") but is more literally state-driven than the old static mock implied — there
is no single "Home" layout, there are three, chosen by data.

### 8.3 Together hub

`Together` (`app/(app)/together/page.tsx`): header "Our space" / "Everything we're building
together", four `Section`s (Plan, Grow, Connect, Remember) each holding two `Row`s that summarise
and link out to Events/Calendar, Goals/Challenges, Journal/Appreciation, Memories/Milestones. This
matches the old spec's grouping exactly (Section 7.2 there), including the section labels.

### 8.4 Prayer Mode

`PrayerModePage` → `PrayerSession` (`features/prayers/components/prayer-session.tsx`): its own
full-bleed chrome on `--color-paper`, no tab bar. Top row is a close button (`×`, "Leave prayer
mode"), a segmented progress strip (current/prayed/upcoming, `aria-hidden` since the count is
announced elsewhere), and a quiet-timer toggle. The body shows one prayer at a time — index label
("Prayer 3 of 7", `aria-live="polite"`), title, body text, optional `Scripture` — with an "I've
prayed" button that becomes an animated checkmark ("Prayed" + "Undo") once marked, then
Previous/Next, and "Finish"/"Done for now" once every point is marked. `?quiet=1` or a week with
zero points renders `QuietPrayer` instead (not read in full for this document; referenced by name
only). This matches the old spec's requirements: one point at a time, large readable text, minimal
chrome, optional timer, previous/next, no gamification or streak framing — confirmed, there is no
streak or score language anywhere in this component.

---

## 9. Interaction and motion

*(Normative)*

Motion tokens are listed in full in §3.4. In practice:

- **Tap feedback** (`.press`): every interactive component (`Button`, `Row`, `SettingRow`,
  `DateRow`, `PrayerRow`, `Switch`, `Checkbox`, tab items) scales to 0.98 and dims to 0.86
  brightness on `:active`, 0.12s ease-out. This is the shipped equivalent of v2.0's "button
  feedback."
- **Entrance:** `Main` content rises in (`--animate-page`, 0.2s), banners and toasts rise in
  (`--animate-rise`, 0.24s), sheets slide up with a slightly bouncier curve
  (`--animate-sheet`, `cubic-bezier(0.2,0.8,0.2,1)`, 0.32s), the sheet scrim fades
  (`--animate-fade`, 0.24s).
- **Completion:** the Prayer Mode checkmark draws itself on (`--animate-draw`, 0.32s, 0.08s
  delay, `stroke-dashoffset`) — the "soft checkmark on completion" the old spec asked for.
- **Loading:** `Skeleton` and the plum "breathing" line on `SplashView` pulse opacity
  (`--animate-breathe`, 1.8s, infinite) rather than spin — no blocking full-screen spinner exists
  anywhere in the components read.
- **What's absent, by design:** no bounce/spring easing anywhere except the sheet curve above, no
  confetti, no floating/idle animation outside the two deliberate "breathing" uses, no animation
  that isn't tied to a state change. This matches the old spec's "avoid" list.

**Reduced motion is required and implemented globally**, not per component:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation: none !important; transition: none !important; }
  .draw-check { stroke-dashoffset: 0 !important; }
}
```

Every `--animate-*` keyframe and every `.press`/`transition-[width]` (e.g. `Bar`'s fill) is
disabled outright rather than swapped for a faster or opacity-only version — a stronger, simpler
position than v2.0's "replace movement with instant or opacity-only transitions," and it needs no
per-component opt-in.

**Haptics:** none. No `navigator.vibrate` or Vibration API usage exists anywhere in the web app,
and there is no haptics layer to spec — the PWA has no access to native haptic feedback APIs
beyond what the OS does automatically for standard controls.

---

## 10. State patterns

*(Normative)*

Every screen that reads data is expected to go through `QueryState` (§7.6) rather than hand-roll
its own branches, so the matrix below is enforced in one place, not copy-pasted per screen.

| State | Pattern |
|---|---|
| **Loading** | `QueryState` renders `Skeleton` (or a screen-supplied `loading` node matching the final layout) while any query is missing data. No full-screen blocking spinner exists in the component set. |
| **Empty** | Per-screen `EmptyState`: a `Ghost` placeholder, a warm headline, one line of text, one clear CTA. Not part of `QueryState` itself — each screen decides what "no data" means for its own content and renders `EmptyState` accordingly. |
| **Error (query failure)** | `QueryState` renders `ErrorState` with `title="This didn't load"` and the server's own message when it is an API error (`isApiError`), else the generic `GENERIC_ERROR_MESSAGE` ("Something went wrong. Please try again.") — never a stack trace, error code, or field name. A "Try again" button re-fetches every missing query. |
| **Offline (query paused, not failed)** | `QueryState` distinguishes this from a hard failure: `title="You're offline"`, `text="This will load when you're back online."` — same retry affordance, different, accurate copy. |
| **Offline (ambient, in-app)** | `NetworkBanner`: a subtle inline `Banner` ("Offline · Showing your saved content"), never a full-screen takeover — content underneath stays usable, matching `DEC-04`/`DEC-05`. Not shown in Prayer Mode (`AppShell` skips `NetworkBanner` there). |
| **Back online / synced** | `NetworkBanner` shows a green "Back online · N change(s) synced" banner for 3.5 seconds (`useSyncedCount`), derived from the drop in TanStack Query's paused-mutation count, then disappears on its own. |
| **Cold-start offline** | `/offline` (SCR-34), a static page precached by the service worker, shown when a navigation can't be reached at all (`DEC-04`). Contains no personal data — the service worker caches this one page for everyone, not per user. |
| **Pending offline changes / unsynced warning on logout** | The web exposes the paused-mutation count via `usePausedCount()`/`useMutationState` (`lib/query/offline.ts`). The logout confirmation sheet (`app/(app)/settings/page.tsx`) reads it: 0 pending → "Your prayers and everything you've shared stay safe. You'll just need to log in again."; N pending → red warning text naming the exact count ("1 change hasn't synced yet. If you log out now, it'll be lost. Reconnect first to keep it.") before the user can confirm. Matches `DEC-05`. |
| **Push permission denied** | `usePush()` (`lib/pwa/push.ts`) exposes a `"denied"` state from the browser's `Notification.permission`. **Gap:** `settings/notifications/page.tsx` does not read this state — its switches call `save.mutate` regardless of the OS-level permission, so a user who denied the browser prompt sees toggles that appear "on" with no indication that no notification will actually arrive. This should show a banner or disable the switches when `state === "denied"`. |
| **Session expired** | `SESSION_EXPIRED_EVENT` (`lib/api/http.ts`) triggers `AppShell` to clear signed-in state and `router.replace(routes.login({ expired: true }))`. `/login?reason=expired` shows "Your session expired. Please log in again." Any unsaved paused mutations are cleared with the signed-out state (`DEC-05`). |
| **Rate limited (429)** | **Gap.** No screen or shared component reads a 429 status or a `Retry-After` header specifically; a rate-limited request would currently surface through the generic error path (`GENERIC_ERROR_MESSAGE` or the server's message if the envelope carries one). `DEC-11` defines the limits server-side, but the client has no dedicated "Too many attempts — try again in Xs" copy yet. Required for `/login` and `/register` (`AUTH` shares its attempt limit with email change, `DEC-07`). |
| **Not found** | `app/not-found.tsx`: plain "That page isn't here" / "Nothing of yours is lost. Head back to your space." with a "Go home" button — no technical detail, no distinction from a cross-couple 404. This is exactly what `DEC-19` requires: a request for another couple's resource returns 404 and must look identical to an unknown route, to prevent resource enumeration. Confirmed: the copy carries no request ID or code. |

---

## 11. Accessibility

*(Normative — the numeric targets are `NFR-A11Y` in [03](./03-non-functional-requirements.md);
this section states the design-level requirements those targets are checked against.)*

- **Contrast:** every light-mode text/background pair in §3.6 passes 4.5:1 for normal text; `edge`
  (borders/icons only) passes the 3:1 non-text minimum and must never carry text. **Gap:**
  `plum-soft` fails the 3:1 non-text minimum in the Prayer Mode progress strip (§3.6). Dark mode
  has no palette to check (`Q-21`).
- **Target size:** the 44×44px design standard is met or exceeded by every interactive component
  in §7 except `TopBar`'s back control and `ComposeBar`'s three controls, which land exactly at
  44px (no margin above the minimum, still compliant) — see the per-component notes in §7.5.
  Nothing in the codebase read falls below the WCAG 2.2 24×24px absolute minimum.
- **Focus order and visible focus:** the tab/DOM order follows the visual order on every screen
  read (no explicit `tabindex` overrides found). Every focusable element gets a visible
  plum focus ring by the single global rule in `globals.css` (`:focus-visible { outline: 2px solid
  var(--color-plum); ... }`) covering links, buttons, inputs, textareas, selects and
  `role="switch"` — there is no component that suppresses this.
- **Labels:** every control read has either a visible label, an `aria-label`, or both (§7 notes
  the couple of narrow exceptions, e.g. `Initial`'s `aria-hidden` reliance on adjacent text).
  Icon-only controls (`TopBar`'s close/back, `ShowButton`) all carry a text label or `aria-label`.
- **Dynamic type / 200% zoom:** not yet verified in a browser; flagged as a **Gap to verify**,
  not asserted either way. The single-column, `max-w-[520px]` layout with no
  fixed-height text containers is structurally favourable, but that is an inference from the code,
  not a measurement.
- **Reduced motion:** honoured globally (§9) — the strongest possible implementation (animations
  and transitions removed outright, not swapped).
- **Colour independence:** confirmed component-by-component in §7 — `StatusMark` (icon + label,
  not colour alone), `Segments` (numeric `aria-label`), `SettingRow`'s destructive tone (red *and*
  the word "Delete"), `Checkbox`/prayer completion (icon + colour together). The one exception is
  the Prayer Mode progress strip, whose current/prayed/upcoming dots differ by colour only (see
  the contrast gap above).
- **Screen-reader announcements for async results:** `ErrorState` and `Alert` (error variant) use
  `role="alert"`; `Banner`, `Toast`, and `Alert` (success variant) use `role="status"` — both are
  live regions, so a query failure, a toast, or an offline/synced banner is announced without a
  manual `aria-live` wire-up. **Gap:** `Skeleton`/`Ghost` (the loading state itself) announce
  nothing — a screen-reader user gets silence, then whichever of `role="alert"`/`role="status"`
  fires next, rather than an explicit "Loading" cue.

---

## 12. Content and UX writing

*(Normative)*

**Voice and tone:** warm, calm, intimate, encouraging, human. Never preachy, never
guilt-driven, never technical. Confirmed against the copy actually shipped in the components and
pages read for this document (Home, Prayer Mode, onboarding, settings, error/offline/not-found
pages) — none of it uses streak, score, or shame language.

**Faith language without pressure or guilt.** Prayer copy invites rather than instructs: "It's
your week. What would you like the two of you to pray about?", "We'll let you know the moment
{partner} shares them. Until then, the time is yours." (waiting state), "Enter quiet prayer" as an
opt-in, not a nudge. No streak counters, no "you're behind" framing anywhere in the prayer flow —
`StatusMark`/`Segments` show completion as a fact ("5 of 7"), not a judgement.

**Word list** (carried over from v2.0, still the standard to write against):

*Use:*
```text
Take a quiet moment when you're ready.
Your prayer week starts today.
Your partner has shared this week's prayers.
What would you like to do together?
Save a moment from today.
```

*Never:*
```text
You haven't prayed yet!
Don't let your partner down.
Your prayer streak is broken.
You're falling behind.
You need to complete this.
```

**Error message pattern:** what happened, then what to do — never technical. Confirmed shipped
copy follows this exactly: `GENERIC_ERROR_MESSAGE` = "Something went wrong. Please try again.";
`NETWORK_ERROR_MESSAGE` = "We couldn't reach the server. Please try again."; the root error
boundary adds a reassurance clause specific to the data at risk ("Your prayers and everything
you've marked are safe. Try again in a moment."). No error string read anywhere in the codebase
contains a stack trace, a database error, an internal field name, or a bare HTTP status — matching
the coding standard that API responses and their client-side rendering must stay user-friendly.
`Field`'s validation errors are short and specific ("That code didn't match.", "Check it with your
partner and try again.") rather than generic.

**Notification copy must not reveal private content on a lock screen.** This is a requirement on
any future push payload, not yet a verified fact: the Go worker that would generate and send push
notifications is **Not started** (`Q-16`). The onboarding permission screen's previews show the
intended shape — activity type and a name, never the content itself: "Your new prayer week is
ready.", "You have a date tonight at 7:00 pm.", "Adeola added something to your shared journal."
None of these reveal what was prayed, written, or planned — only that something happened. Any
future server-side notification payload must hold to the same rule: name the kind of update and,
where relevant, who made it, never the prayer text, journal entry, or appreciation note itself.

**Formats**, as implemented in `lib/dates.ts` (all confirmed from the code, not assumed):

| Format | Example | Rule |
|---|---|---|
| Short date | `27 September` | Day number + full month name, no year |
| Weekday date | `Tuesday 29 September` | Full weekday + short date |
| Date range | `27 Sep - 3 Oct` or `27 September - 3 October` | Same-month ranges collapse to one month name |
| Relative day | `Today` / `Tomorrow` / `Thursday` / `27 September` | Today/tomorrow for 0–1 days out, weekday name for 2–6 days out, full date beyond that |
| Time | `7:00 pm` | 12-hour, lowercase `am`/`pm`, no leading zero on the hour. **Gap:** formatted from the raw `HH:MM` string with no timezone conversion — it displays whatever wall-clock time the record carries, which only reads correctly if that's already the viewer's own or the couple's timezone (`Q-07` is the open question on which timezone governs this) |
| Currency | `₦500,000` | `₦` prefix, `Intl`/`toLocaleString('en-NG')` thousands separators, no decimal places shown for whole amounts |
| Invite code | `ABC-123` | Stored as `ABC123`, shown with a hyphen after the third character (`DEC-08`) |

---

## 13. Appendix A: Master design prompt

*(Informative — this is a compressed prompt for UI-generation tools, not a replacement for the
document above. Colours and typeface are updated to match the current tokens; everything else is
carried over from v2.0 unchanged in substance.)*

> Design a premium, mobile-first PWA called Amorae, a private digital space for couples to
> connect, grow, plan, share experiences, practice their faith, and build memories together.
>
> It should feel like a beautifully designed private product for two people, not a church app,
> dating app, productivity dashboard, or social network. The central idea is two people
> intentionally building a life together. Prayer is one pillar, not the identity — there is no
> religious iconography anywhere in the logo or navigation. The experience also supports shared
> events, date nights, couple planning, calendars, shared goals, relationship challenges,
> appreciation, journaling, memories, milestones, and important dates.
>
> Design around five pillars: Connect (appreciation, journal, prompts, reflections), Plan (events,
> calendar, dates, reminders), Grow (shared goals and challenges), Faith (prayer, scripture,
> devotion), Remember (memories and milestones).
>
> The visual design must look human-designed, intentional, calm, premium, editorial, intimate, and
> timeless. Avoid excessive gradients, neon, glassmorphism, glowing blobs, floating 3D, generic AI
> imagery, giant hero sections, excessive rounded cards, generic SaaS dashboards, cliché romantic
> imagery, excessive heart icons, religious clip-art, and playful gamification.
>
> Use a warm off-white background (`#F7F4EF`), a warm paper tone for secondary surfaces
> (`#F2EDE5`), a soft-white card surface (`#FFFDFA`), deep ink text (`#24201F`), muted stone
> secondary text (`#6B635C`), and a single restrained deep plum accent (`#5B2A4A`) — no secondary
> accent colour. Set everything in Manrope, one weight family, with strong hierarchy and generous
> whitespace; body text never drops below 16px so it never triggers iOS zoom.
>
> Design for an iPhone-first installed PWA: safe areas, home indicator, standalone mode, dynamic
> viewport (`100dvh`), keyboard behaviour, 44px touch targets, a five-item bottom navigation
> (Home, Together, Prayers, History, Settings), push notifications that never reveal private
> content on a lock screen, a quiet inline offline indicator that never takes over the screen, and
> Home Screen installation. It should feel native and tactile even though it is a web app. Use
> subtle, purposeful motion only — tap feedback, gentle entrances, a soft checkmark draw-on for
> completion — never decorative animation, and respect `prefers-reduced-motion` by removing motion
> outright, not just softening it.
>
> Bottom navigation: Home, Together, Prayers, History, Settings. Home is state-driven — it's your
> week to set prayers, or you're waiting on your partner, or a normal week showing tonight's plan,
> this week's prayer progress, the next event, and one active goal — never a grid of statistics.
> Prayer Mode is distraction-free and peaceful: one prayer at a time, large readable text, a
> progress strip, and a calm "I've prayed" action, with no streaks or scores anywhere. Goals feel
> collaborative, not competitive. Memories feel like a private archive, not a social feed. Empty
> states are warm and inviting, each with exactly one action. UX writing is warm, calm, and never
> guilt-driven, even when something has been missed.
>
> Logo: an abstract symbol of two individuals moving toward a shared centre and becoming connected
> without losing individuality. No praying hands, crosses, Bibles, doves, wedding rings, obvious
> hearts, or infinity symbols. It must work at 16px minimum, as a favicon, and as an iPhone Home
> Screen icon, with a heavier stroke below 32px.
>
> Produce a consistent design system: tokens (colour, type scale, radius, motion), components and
> their states (default, pressed, focus-visible, disabled, loading, error), a full screen
> inventory with loading/empty/error/offline states for every data-driven screen, and interaction
> patterns. The result should feel like a real premium consumer app from a strong product design
> team, with strong UX, restraint, clarity, emotional warmth, and excellent mobile ergonomics.
>
> Amorae. Two hearts, one faith.

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 3.0 | 2026-09-22 | Full rework against the codebase as built. Reconciled every token, type-scale value, radius, elevation and motion value against `apps/web/src/app/globals.css` and `docs/BRAND.md` (`DEC-14`); added a "Changed from v2.0" table and a computed WCAG contrast table (every text pair passes 4.5:1; `plum-soft` fails the 3:1 non-text minimum). Rewrote the component specification against the real files in `components/ui/` and `components/layout/`, marking gaps where the code lacks a state this document requires. Rebuilt the screen inventory from the actual route tree (38 screens) with a status per screen (Implemented/Partial/Mock only) and reconciled it against the v2.0 inventory. Rewrote state patterns, accessibility and content/UX writing sections against shipped copy and behaviour, flagging gaps (rate-limited copy, push-denied handling, loading-state screen-reader announcements, dark mode) rather than asserting them as built. Updated Appendix A's master prompt to the current palette and typeface. |
| 2.0 | 2026-09-21 | Split out of the combined spec. Added a real token system (hex colour, spacing, radius, elevation, motion), a typography scale table, a component list with states, a grouped screen inventory, a state-pattern matrix, concrete accessibility criteria, and a UX writing library. Moved the master prompt to an appendix. |

