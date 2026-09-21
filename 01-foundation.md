# Amorae: Product Foundation

**Version:** 2.0 | **Status:** MVP definition | **Owner:** Product | **Last updated:** 2026-09-21

This is the shared context both the engineering and design specs build on. If you only read one file, read this one. It defines what Amorae is, what it is not, and what ships in the first version.

---

## 1. Vision

Amorae is a private digital space for two people to intentionally build their relationship together.

Prayer and faith matter, but Amorae is not a prayer app. It supports the whole shared life of a couple: planning, goals, appreciation, journaling, memories, milestones, important dates, and faith.

The core idea, in one line:

> Two people, intentionally building a life together.

The product should feel private, calm, intimate, useful, and human. It should never feel like a public social network, a generic productivity tool, a dating app, or a church management system.

---

## 2. Positioning

**Primary statement:**

> Amorae is a private space for couples to connect, grow, plan, share experiences, practice their faith, and build memories together.

Prayer is a first-class feature, but it does not define the identity of the product. Everything (logo, navigation, content, architecture) should stay relevant whether the couple is praying, planning a date, working toward a goal, or saving a memory.

**Final framing the product should earn:**

> Your private space for the life you are building together.

The product should make the couple feel Amorae belongs to "us."

---

## 3. The five pillars

Every feature maps to one of five pillars. This is the mental model for the whole product, the navigation, and the roadmap.

| Pillar | Purpose | Feature area |
|--------|---------|--------------|
| **Connect** | Strengthen the emotional bond | Appreciation, shared journal, prompts, reflections |
| **Plan** | Organize what the couple does together | Events, couple calendar, dates, reminders, checklists |
| **Grow** | Improve together on purpose | Shared goals, challenges, monthly intentions |
| **Faith** | Support shared spiritual life | Weekly prayer, prayer history, scripture, devotion |
| **Remember** | Preserve the shared journey | Memories, milestones, important dates, event recaps |

---

## 4. Core principles

These are non-negotiable. Both engineering and design decisions are checked against them.

1. **Private by default.** Couple data belongs to the couple. No public surface, ever.
2. **A two-person experience.** The app always understands it is a shared space between two partners.
3. **Intentional, not addictive.** Encourage meaningful moments, not endless scrolling or streak pressure.
4. **Warm, not childish.** No excessive hearts, pink gradients, cartoon romance, or cliche imagery.
5. **Useful before decorative.** Every visual element should improve understanding, orientation, or tone.
6. **Human before AI.** AI is optional and never the defining interaction. The product must work fully with AI disabled.
7. **Faith without pressure.** Prayer invites reflection. It never uses guilt, manipulation, or religious pressure.
8. **Designed for real life.** It must work with limited time, weak network, and one free hand.

---

## 5. MVP scope

MVP ships the core loop of a couple pairing, praying weekly, planning shared life, and saving memories.

### In scope

| Area | Included |
|------|----------|
| Auth | Sign up, login, logout, password reset, session persistence |
| Pairing | Create couple, invite by code or link, join, two members max, couple profile |
| Faith | Weekly prayer with alternating setter, prayer points, independent completion, prayer history |
| Plan | Shared events, simple couple calendar, reminders |
| Grow | Shared goals, guided challenges |
| Connect | Appreciation notes, shared journal |
| Remember | Memories, milestones, important dates |
| Platform | Installable PWA, push notifications, offline reading, iPhone-first ergonomics |

### Out of scope for MVP (Later)

Public social feed, group prayer, public profiles, community features, voice or video calls, a full chat system, payments, large-scale gamification, an AI chatbot, complex financial management, and heavy analytics.

The first version should be narrow and deeply polished rather than broad and shallow.

---

## 6. Roadmap

Phases map to the pillars and to engineering modules, so teams can build and ship in vertical slices.

| Phase | Theme | Ships |
|-------|-------|-------|
| 1 | Foundation | Auth, couple pairing, PWA shell, Home, profile and settings |
| 2 | Faith | Weekly prayers, prayer points, completion, prayer history, push reminders |
| 3 | Together | Events, couple calendar, shared goals, challenges |
| 4 | Memories | Journal, appreciation, memories, milestones |
| 5 | Optional intelligence | AI prayer assistant, AI rewriting, reflections, verified scripture suggestions |

---

## 7. Glossary

Shared vocabulary so the two specs stay consistent.

| Term | Meaning |
|------|---------|
| **Couple** | The shared account container. Owns all shared resources. |
| **Member / Partner** | One of the two users in a couple. |
| **Setter** | The partner responsible for setting prayer points in a given week. Alternates weekly. |
| **Prayer week** | A weekly container of prayer points, starting Sunday in the couple's timezone. |
| **Prayer point** | A single item to pray about within a prayer week. |
| **Completion** | An individual partner marking a prayer point as prayed. Independent per partner. |
| **Together hub** | The shared-life section that groups events, calendar, goals, challenges, journal, and memories. |
| **Couple-scoped** | Any resource that belongs to a couple and is only visible to its two members. |

---

## 8. Decisions log

- **One couple per user in MVP.** Keeps pairing and authorization simple. The data model still uses a membership table, so this can be relaxed later without a rewrite. See engineering spec, scalability section.
- **Prayer stays first-class but not the identity.** Drives the five-pillar structure and the neutral logo direction (no crosses, praying hands, or doves).
- **AI is isolated and optional.** Lives behind an interface and a feature flag so it can be turned off or swapped without touching core features.
