# Amorae: Design Specification

**Version:** 2.0 | **Status:** MVP definition | **Owner:** Design | **Last updated:** 2026-09-21

Read [`01-foundation.md`](./01-foundation.md) first for product context. This document covers design principles, tokens (real values), typography, components with states, the screen inventory, motion, accessibility, and UX writing.

---

## Context in one screen

- Amorae is a private PWA for two people. Five pillars: Connect, Plan, Grow, Faith, Remember.
- The feel: quiet premium, intimate, editorial, calm, human-designed. Warm, not childish.
- It must not look like a church app, dating app, productivity dashboard, or social feed.
- iPhone-first, installed PWA. Design for safe areas, one-handed use, and weak networks.

---

## 1. Design principles

1. **Human-designed, not generated.** Restraint and intention over decoration. If an element does not aid understanding, orientation, or tone, remove it.
2. **Warm minimalism.** Generous whitespace, strong typography, restrained borders, calm rhythm.
3. **Shared, not competitive.** Everything reads as "ours." No leaderboards, streak pressure, or guilt.
4. **Tactile and native.** The installed app should feel closer to a native iPhone app than a website.
5. **Typography carries the design.** Use type hierarchy, not ornament, to create structure.

**Avoid:** excessive gradients, glassmorphism, glowing blobs, floating 3D, neon, generic AI dashboards, deep card nesting, huge hero sections, unnecessary illustrations, cliche romance imagery, repeated heart icons, religious clip-art, and playful gamification.

---

## 2. Design tokens

Use tokens, never raw values, in components. Values below are the starting palette and scale. Tune in implementation, but keep the semantic names stable.

### 2.1 Color

**Raw palette**

| Token | Hex | Role |
|-------|-----|------|
| `warm-white` | #F6F4EF | App background (warm off-white) |
| `soft-white` | #FCFBF9 | Card and surface |
| `pure-white` | #FFFFFF | Raised surface (sheets, modals) |
| `charcoal` | #26221F | Primary text (deep charcoal) |
| `stone` | #6E675F | Secondary text (muted stone) |
| `stone-light` | #948C82 | Tertiary text and microcopy |
| `border` | #E6E1D9 | Warm gray border |
| `hairline` | #EEEAE2 | Lightest divider |
| `plum` | #5B2A4A | Primary accent (deep plum) |
| `plum-dark` | #47213A | Accent pressed |
| `plum-soft` | #F0E7EC | Accent tint background |
| `berry` | #7A3B54 | Secondary accent (muted berry) |
| `green` | #4F7A5E | Positive / success (muted) |
| `amber` | #A9762C | Warning (restrained) |
| `clay` | #A6483C | Error / destructive (muted terracotta) |

**Semantic tokens (use these in components)**

| Semantic | Maps to |
|----------|---------|
| `bg` | warm-white |
| `surface` | soft-white |
| `surface-raised` | pure-white |
| `text-primary` | charcoal |
| `text-secondary` | stone |
| `text-tertiary` | stone-light |
| `border-default` | border |
| `divider` | hairline |
| `accent` | plum |
| `accent-hover` | plum-dark |
| `accent-tint` | plum-soft |
| `success` | green |
| `warning` | amber |
| `danger` | clay |

Palette stays calm and accessible. Color is never the only status signal (see accessibility).

### 2.2 Spacing (4pt base)

`2, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64`. Names: `space-0.5=2`, `space-1=4`, `space-2=8`, `space-3=12`, `space-4=16`, `space-5=20`, `space-6=24`, `space-8=32`, `space-10=40`, `space-12=48`, `space-16=64`. Default screen padding is `space-5` (20). Default gap between stacked cards is `space-3` (12).

### 2.3 Radius

| Token | Value | Use |
|-------|-------|-----|
| `radius-sm` | 8 | Inputs, chips |
| `radius-md` | 12 | Cards, buttons (default) |
| `radius-lg` | 16 | Sheets, large cards |
| `radius-xl` | 20 | Prayer mode surfaces |
| `radius-pill` | 999 | Pills, avatars |

Rounding is restrained. Avoid oversized rounded cards.

### 2.4 Elevation

Prefer hairline borders over shadows. Two shadow levels only.

| Token | Value | Use |
|-------|-------|-----|
| `elev-1` | 0 1px 2px rgba(38,34,31,0.05) | Cards that need lift |
| `elev-2` | 0 8px 24px rgba(38,34,31,0.10) | Bottom sheets, modals |

### 2.5 Motion

| Token | Value |
|-------|-------|
| `dur-fast` | 120 ms |
| `dur-base` | 200 ms |
| `dur-slow` | 320 ms |
| `ease-standard` | cubic-bezier(0.2, 0, 0, 1) |

All motion respects `prefers-reduced-motion`. See Section 8.

---

## 3. Typography

Font: Inter for UI and body. Manrope is an acceptable alternative for display and headings. Use one family per build for consistency.

| Style | Size / line-height | Weight | Use |
|-------|-------------------|--------|-----|
| Display | 34 / 40 | 600 | Splash, welcome, big moments |
| Page title | 26 / 32 | 600 | Screen headers |
| Section title | 19 / 26 | 600 | Group headers |
| Body large | 17 / 26 | 400 | Prayer mode, reading |
| Body | 15 / 24 | 400 | Default text |
| Caption | 13 / 18 | 500 | Supporting text, metadata |
| Micro | 12 / 16 | 500 | Labels, timestamps (often uppercase, +0.04em tracking) |

Display and titles use tracking around -0.01 to -0.02em. Hierarchy comes from type, not from color alone.

---

## 4. Iconography and imagery

- **Icons:** a single consistent line-icon set, 1.5px stroke, rounded joins. Neutral, not decorative. No repeated hearts.
- **Avatars:** circle, `radius-pill`, with a warm neutral fallback showing initials.
- **Photos:** user photos only (memories, journal). No stock romance imagery. Photos sit inside `radius-md` frames on the surface color.
- **Empty and illustrative art:** minimal or none. Prefer warm words over illustration.

---

## 5. Layout and responsive

- **Frame:** iPhone-first. Design at 390px width as the baseline, scale up gracefully.
- **Safe areas:** honor `env(safe-area-inset-top)` and `env(safe-area-inset-bottom)`. Content never sits under the notch or home indicator.
- **Bottom navigation:** fixed, five items, height 56 plus bottom safe-area inset. Touch targets at least 44px.
- **Screen structure:** top area (title or greeting), scrollable content, optional bottom nav. Full-height layouts use `100dvh` to survive the dynamic viewport and keyboard.
- **Content width:** single column. No dense multi-column dashboards.

**Bottom nav (five items):** Home, Together, Prayers, History, Settings. "Together" is the hub for events, calendar, goals, challenges, journal, and memories, which keeps the nav to five clear destinations.

---

## 6. Components

A reusable design system. Each component defines its states. States that apply broadly: default, hover, pressed, disabled, focus (visible ring), loading.

| Component | Notes and key states |
|-----------|----------------------|
| Button | Primary (accent fill), secondary (surface + border), text (accent label). States: default, pressed, disabled, loading (spinner replaces label). Min height 44. |
| Icon button | 44x44 target, transparent, subtle pressed state. |
| Input / Textarea | Surface fill, `border-default`, focus ring in accent. Error state uses `danger` border and a message below. Label above. |
| Select | Native-feeling, opens a bottom sheet on mobile rather than a tiny dropdown. |
| Bottom navigation | Five items, active item uses accent, inactive uses `text-tertiary`. |
| Top bar | Screen title left, at most one action right. Minimal chrome. |
| Event card | Title, date, time, location, status. Tap opens detail. |
| Goal card | Title, progress indicator, collaborative framing (no competition). |
| Prayer card | Point text, per-partner completion marks (see below). |
| Memory card | Photo, caption, date. Reads as archive, not feed. |
| Journal entry | Author, type tag, body preview, date. |
| Appreciation card | Warm, personal, quiet. Not styled like a social post. |
| Calendar | Simple chronological list grouped by date. No grid. |
| Progress indicator | Calm bar or ring. Shared, not gamified. |
| Empty state | Warm headline plus one clear action (see Section 7). |
| Offline indicator | Subtle inline banner, never full-screen. |
| Toast | Brief, low, non-blocking. |
| Modal / Bottom sheet | Sheet is the default on mobile. Uses `elev-2`, `radius-lg`. |
| Confirmation dialog | Calm copy, destructive action in `danger`. |
| Date picker / Time picker | Native-feeling, large touch targets. |

**Completion marks (prayer):** show both partners' status without shame. Example:

```text
For wisdom in the decisions ahead.
You  ✓        Adeola  ○
```

Components prioritize mobile ergonomics: large targets, thumb-reachable actions, generous spacing.

---

## 7. Screen inventory

Grouped by flow. Each screen has a clear job. States (loading, empty, offline, error) apply per Section 9.

**Onboarding and setup**

1. Splash / loading
2. Welcome
3. Sign up
4. Login
5. Create couple
6. Join couple
7. Invite partner
8. Notification permission (explain value first)
9. PWA install guidance (gentle, contextual, not a popup)

**Daily use**

10. Home
11. Together hub
12. Couple calendar
13. Events list
14. Create / edit event
15. Event detail
16. Shared goals
17. Create goal
18. Goal detail
19. Challenges
20. Journal
21. Appreciation
22. Memories
23. Milestones and important dates

**Prayer**

24. Current prayer week
25. Prayer detail
26. Prayer Mode (distraction-free)
27. Create / edit prayer
28. Publish confirmation
29. Waiting-for-partner state
30. Prayer completion state
31. Prayer history and history detail

**System**

32. Settings
33. Profile
34. Notification preferences
35. Offline state
36. Error state
37. Empty states

### 7.1 Home

Not a dashboard of tiny cards. It prioritizes today, then this week, then one or two meaningful actions. The screen should breathe.

```text
Good evening
Femi & Adeola

Tonight
Dinner together
7:00 PM

This week
Weekly Prayer      Setter: Femi      5 of 7 completed
Together           2 upcoming events    1 active goal

[ Open our week ]
```

Quick actions (kept short): add event, add goal, write appreciation, add memory, open prayer. Avoid competitive metrics and stat grids.

### 7.2 Together hub

Header "Our space" with supporting line "Everything we're building together." Groups: Events, Calendar, Goals, Challenges, Journal, Memories. Clear grouping, not a dense grid.

### 7.3 Prayer Mode

Distraction-free and peaceful. Requirements: large readable text, one point at a time, clear completion action, minimal chrome, optional timer, previous/next, calm background, no unnecessary animation.

```text
Prayer 3 of 7

"For wisdom and direction in the decisions ahead."

[ I've prayed ]
```

No gamification, no pressure, no streak framing.

---

## 8. Motion

Motion is subtle and purposeful.

**Use:** small page transitions, button feedback, a soft checkmark on completion, bottom-sheet slide, progress changes, gentle notification appearance.

**Avoid:** bouncy animation, confetti for ordinary actions, constant floating motion, decorative animation with no purpose.

**Reduced motion:** when `prefers-reduced-motion` is set, replace movement with instant or opacity-only transitions.

---

## 9. State patterns

Every data view defines these states. This is the matrix components and screens follow.

| State | Pattern |
|-------|---------|
| Loading | Skeletons that match final layout. No blocking spinner over the whole screen. |
| Empty | Warm headline plus one clear action. Never looks broken. |
| Offline | Subtle inline indicator: "Offline. Showing your saved content." App stays usable. |
| Error | Calm, specific, recoverable. Offer a retry. No stack traces or scary language. |
| Success | Quiet confirmation (toast or soft checkmark), not celebration. |

**Empty state examples**

```text
No events
Nothing planned yet.
Add something you'd love to do together.

No memories
Your story starts here.
Save your first shared moment.

No goals
What would you like to build together?
```

**Offline example:** "Offline. Changes will sync when you're back online."

**Install nudge (contextual, not a popup):** "Keep Amorae close. Add Amorae to your Home Screen for a more private, app-like experience." Remember when it has been dismissed.

---

## 10. Accessibility

- **Contrast:** meet WCAG AA. Body text and UI targets pass 4.5:1; large text passes 3:1.
- **Touch targets:** at least 44x44.
- **Focus:** every interactive element has a visible focus ring (accent).
- **Screen readers:** labels on all controls, meaningful reading order, live regions for toasts.
- **Reduced motion:** honored (see Section 8).
- **Color independence:** status is never carried by color alone. Pair it with an icon, mark, or text (for example the prayer completion check plus label).
- **Text scaling:** layouts survive larger system text without clipping.
- **Errors:** clear, specific, and tied to the field they concern.

---

## 11. UX writing

Tone: warm, calm, intimate, encouraging, human, never preachy, never guilt-driven.

**Use**

```text
Take a quiet moment when you're ready.
Your prayer week starts today.
Your partner has shared this week's prayers.
What would you like to do together?
Save a moment from today.
```

**Never**

```text
You haven't prayed yet!
Don't let your partner down.
Your prayer streak is broken.
You're falling behind.
You need to complete this.
```

The difference is intention. Amorae invites; it never pressures.

---

## 12. Logo direction

An abstract mark of two individuals moving toward a shared center and becoming connected, without losing their individuality. It should suggest connection, partnership, growth, belonging, and a shared journey.

**Do not use:** praying hands, crosses, Bibles, doves, wedding rings, generic hearts, infinity symbols, or religious clip-art.

The mark must work at 24px, 48px, as a favicon, and as an iPhone Home Screen icon. It should stay meaningful if Amorae expands beyond faith into broader couple features.

---

## Appendix A: Master design prompt

Use this as the master prompt for UI-generation or design tools. It is a compression of this document, not a replacement for it.

> Design a premium, mobile-first PWA called Amorae, a private digital space for couples to connect, grow, plan, share experiences, practice their faith, and build memories together.
>
> It should feel like a beautifully designed private product for two people, not a church app, dating app, productivity dashboard, or social network. The central idea is two people intentionally building a life together. Prayer is one pillar, not the identity. The experience also supports shared events, date nights, couple planning, calendars, shared goals, relationship challenges, appreciation, journaling, memories, milestones, and important dates.
>
> Design around five pillars: Connect (appreciation, journal, prompts, reflections), Plan (events, calendar, dates, reminders), Grow (shared goals and challenges), Faith (prayer, scripture, devotion), Remember (memories and milestones).
>
> The visual design must look human-designed, intentional, calm, premium, editorial, intimate, and timeless. Avoid excessive gradients, neon, glassmorphism, glowing blobs, floating 3D, generic AI imagery, giant hero sections, excessive rounded cards, generic SaaS dashboards, cliche romantic imagery, excessive heart icons, religious clip-art, and playful gamification.
>
> Use a warm off-white foundation, soft-white surfaces, deep charcoal text, warm gray borders, muted stone secondary text, and restrained deep plum or muted berry accents. Use Inter or Manrope with strong hierarchy and generous whitespace.
>
> Design for an iPhone-first installed PWA: safe areas, home indicator, standalone mode, dynamic viewport, keyboard behavior, 44px targets, bottom navigation, push notifications, offline states, network recovery, and Home Screen installation. It should feel native and tactile even though it is a web app. Use subtle motion and interaction feedback, no decorative animation.
>
> Bottom navigation: Home, Together, Prayers, History, Settings. Home prioritizes what matters today and this week, not a grid of statistics. Prayer Mode is distraction-free and peaceful. Goals feel collaborative, not competitive. Memories feel like a private archive, not a social feed. Empty states are warm and inviting. UX writing is warm, calm, and never guilt-driven.
>
> Logo: an abstract symbol of two individuals moving toward a shared center and becoming connected without losing individuality. No praying hands, crosses, Bibles, doves, wedding rings, obvious hearts, or infinity symbols. It must work at 24px, 48px, as a favicon, and as an iPhone Home Screen icon.
>
> Produce a consistent design system: tokens, spacing, type scale, components, states, and interaction patterns. The result should feel like a real premium consumer app from a strong product design team, with strong UX, restraint, clarity, emotional warmth, and excellent mobile ergonomics.
>
> Amorae. Two hearts, one faith.

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 2.0 | 2026-09-21 | Split out of the combined spec. Added a real token system (hex color, spacing, radius, elevation, motion), a typography scale table, a component list with states, a grouped screen inventory, a state-pattern matrix, concrete accessibility criteria, and a UX writing library. Moved the master prompt to an appendix. |
