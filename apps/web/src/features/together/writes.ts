import type { EventInput, GoalInput } from "@/lib/api/schemas";
import type { ChallengeDay, JournalEntry, Memory, Milestone } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { togetherApi as api } from "./api";

// Every write the Together feature makes, declared once (see lib/query/mutations.ts).
export const togetherWrites = {
  saveEvent: defineWrite({
    mutationKey: ["events", "save"],
    mutationFn: ({ id, input }: { id?: string; input: EventInput }) => (id ? api.updateEvent(id, input) : api.createEvent(input)),
    invalidates: [keys.events],
  }),
  completeEvent: defineWrite({
    mutationKey: ["events", "complete"],
    mutationFn: ({ id, done }: { id: string; done: boolean }) => api.completeEvent(id, done),
    invalidates: [keys.events],
  }),
  checklist: defineWrite({
    mutationKey: ["events", "checklist"],
    mutationFn: ({ id, item, done }: { id: string; item: string; done: boolean }) => api.checklist(id, item, done),
    invalidates: [keys.events],
  }),
  createGoal: defineWrite({
    mutationKey: ["goals", "create"],
    mutationFn: (g: GoalInput) => api.createGoal(g),
    invalidates: [keys.goals],
  }),
  addProgress: defineWrite({
    mutationKey: ["goals", "progress"],
    mutationFn: ({ id, amount }: { id: string; amount: number }) => api.progress(id, amount),
    invalidates: [keys.goals],
  }),
  challengeDay: defineWrite({
    mutationKey: ["challenge", "day"],
    mutationFn: ({ n, patch }: { n: number; patch: Partial<ChallengeDay> }) => api.challengeDay(n, patch),
    invalidates: [keys.challenge],
  }),
  addJournal: defineWrite({
    mutationKey: ["journal", "add"],
    mutationFn: ({ tag, text }: { tag: JournalEntry["tag"]; text: string }) => api.addJournal(tag, text),
    invalidates: [keys.journal],
  }),
  sendAppreciation: defineWrite({
    mutationKey: ["appreciations", "send"],
    mutationFn: (text: string) => api.sendAppreciation(text),
    invalidates: [keys.appreciations],
  }),
  undoAppreciation: defineWrite({
    mutationKey: ["appreciations", "undo"],
    mutationFn: (id: string) => api.undoAppreciation(id),
    invalidates: [keys.appreciations],
  }),
  addMemory: defineWrite({
    mutationKey: ["memories", "add"],
    mutationFn: (m: Omit<Memory, "id">) => api.addMemory(m),
    invalidates: [keys.memories],
  }),
  addMilestone: defineWrite({
    mutationKey: ["milestones", "add"],
    mutationFn: (m: Omit<Milestone, "id">) => api.addMilestone(m),
    invalidates: [keys.milestones],
  }),
};
