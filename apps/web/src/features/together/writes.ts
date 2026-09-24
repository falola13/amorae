import type { EventInput, GoalInput } from "@/lib/api/schemas";
import type { ChallengeDay, JournalEntry, Memory, Milestone } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { togetherApi as api } from "./api";

// Every write the Together feature makes, declared once (see lib/query/mutations.ts).
export const togetherWrites = {
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  saveEvent: defineWrite({
    mutationKey: ["events", "save"],
    mutationFn: ({ id, input }: { id?: string; input: EventInput }) =>
      id ? api.updateEvent(id, input) : api.createEvent(input),
    invalidates: [keys.events],
    // Updating is idempotent, creating is not, and this write does both.
    onlineOnly: true,
  }),
  deleteEvent: defineWrite({
    mutationKey: ["events", "delete"],
    mutationFn: (id: string) => api.deleteEvent(id),
    invalidates: [keys.events],
    idempotent: "deleting something already gone leaves the same nothing behind.",
  }),
  completeEvent: defineWrite({
    mutationKey: ["events", "complete"],
    mutationFn: ({ id, done }: { id: string; done: boolean }) => api.completeEvent(id, done),
    invalidates: [keys.events],
    idempotent: "sets the flag to a given value rather than toggling it.",
  }),
  checklist: defineWrite({
    mutationKey: ["events", "checklist"],
    mutationFn: ({ id, item, done }: { id: string; item: string; done: boolean }) =>
      api.checklist(id, item, done),
    invalidates: [keys.events],
    idempotent: "sets one item to a given value, so a replay lands the same state.",
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  createGoal: defineWrite({
    mutationKey: ["goals", "create"],
    mutationFn: (g: GoalInput) => api.createGoal(g),
    invalidates: [keys.goals],
    onlineOnly: true,
  }),
  // Adds an amount rather than setting one, so a replay double-counts it.
  // The endpoint should take the new total, not a delta, and then this can
  // queue like the rest (FR-PWA-009).
  updateGoal: defineWrite({
    mutationKey: ["goals", "update"],
    mutationFn: ({ id, patch }: { id: string; patch: { done?: boolean } }) =>
      api.updateGoal(id, patch),
    invalidates: [keys.goals],
    idempotent: "sets the fields it names to the values it carries, however often it lands.",
  }),
  addProgress: defineWrite({
    mutationKey: ["goals", "progress"],
    mutationFn: ({ id, amount }: { id: string; amount: number }) => api.progress(id, amount),
    invalidates: [keys.goals],
    onlineOnly: true,
  }),
  challengeDay: defineWrite({
    mutationKey: ["challenge", "day"],
    mutationFn: ({ n, patch }: { n: number; patch: Partial<ChallengeDay> }) =>
      api.challengeDay(n, patch),
    invalidates: [keys.challenge],
    idempotent: "a patch of one numbered day; the same patch twice is the same day.",
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  addJournal: defineWrite({
    mutationKey: ["journal", "add"],
    mutationFn: ({ tag, text }: { tag: JournalEntry["tag"]; text: string }) =>
      api.addJournal(tag, text),
    invalidates: [keys.journal],
    onlineOnly: true,
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  sendAppreciation: defineWrite({
    mutationKey: ["appreciations", "send"],
    mutationFn: (text: string) => api.sendAppreciation(text),
    invalidates: [keys.appreciations],
    onlineOnly: true,
  }),
  // Only meaningful within seconds of sending, so it is never queued offline:
  // replayed hours later, it would delete a note the partner has already read.
  undoAppreciation: defineWrite({
    mutationKey: ["appreciations", "undo"],
    mutationFn: (id: string) => api.undoAppreciation(id),
    invalidates: [keys.appreciations],
    onlineOnly: true,
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  addMemory: defineWrite({
    mutationKey: ["memories", "add"],
    mutationFn: (m: Omit<Memory, "id">) => api.addMemory(m),
    invalidates: [keys.memories],
    onlineOnly: true,
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  addMilestone: defineWrite({
    mutationKey: ["milestones", "add"],
    mutationFn: (m: Omit<Milestone, "id">) => api.addMilestone(m),
    invalidates: [keys.milestones],
    onlineOnly: true,
  }),
};
