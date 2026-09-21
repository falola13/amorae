# Amorae Specification

**Product:** Amorae  
**Tagline:** Two hearts, one faith.  
**What it is:** A private, mobile-first PWA for two people to build their relationship: connection, planning, shared goals, faith, and memories.  
**Version:** 2.0  
**Status:** MVP definition, ready for build  
**Last updated:** 2026-09-21

---

## How this repo of docs is organized

The old single spec mixed product vision, engineering detail, and design detail in one file. That made it hard to hand a clean, focused document to either an engineer or a designer. It is now split into three files, each with one owner and one job.

| File | Owner | Read it when you need |
|------|-------|-----------------------|
| [`01-foundation.md`](./01-foundation.md) | Product | The shared truth: vision, positioning, pillars, principles, MVP scope, roadmap, and glossary. Both other docs assume you have read this. |
| [`02-engineering.md`](./02-engineering.md) | Engineering | Functional requirements with acceptance criteria, data model, API, backend architecture, the weekly prayer scheduler, security, PWA internals, AI, and non-functional targets. |
| [`03-design.md`](./03-design.md) | Design | Design principles, tokens (real hex, spacing, type scale), components with states, screen inventory, motion, accessibility, and UX writing. |

Each of the two working specs (engineering, design) opens with a short context recap so it stands on its own. The full product context lives in `01-foundation.md` and is the single source of truth. If vision or scope changes, change it there first.

---

## Working conventions

- **One source of truth.** Product facts live in `01-foundation.md`. Engineering and design docs reference it rather than restating it in detail.
- **Requirement language.** "Must" is required for MVP. "Should" is expected but can slip a release. "Later" is explicitly out of MVP scope.
- **Versioning.** Bump the version at the top of a file on any meaningful change and add a line to its Change Log. Keep a short note on what changed and why.
- **Decisions.** Non-obvious calls (why a table was dropped, why a limit exists) are recorded in the Decisions Log at the end of the relevant doc, not lost in chat.
- **Open questions.** Anything unresolved goes in the Open Questions section of the relevant doc so it does not get silently assumed.

---

## Change log

| Version | Date | Change |
|---------|------|--------|
| 2.0 | 2026-09-21 | Split the single spec into foundation / engineering / design. Converted vague feature lists into concrete requirements with acceptance criteria. Added real data model with columns, types, and enums. Added API conventions and example payloads. Added design tokens with hex values, type scale, and component states. Added non-functional targets, offline sync rules, and a scalability section. |
| 1.0 | (prior) | Original combined product requirements and UX design spec. |
