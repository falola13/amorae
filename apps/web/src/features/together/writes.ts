import type { EventInput, GoalInput } from "@/lib/api/schemas";
import type {
  Challenge,
  ChallengeDay,
  Event,
  JournalEntry,
  Memory,
  Milestone,
} from "@/lib/api/types";
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
  // Changing one thing about an event that already exists, from the event
  // itself. Separate from saveEvent because that one also creates, which is
  // what forces it online-only; a PATCH sets the fields it names to the
  // values it carries, so landing twice lands the same event.
  patchEvent: defineWrite({
    mutationKey: ["events", "patch"],
    mutationFn: ({ id, patch }: { id: string; patch: Partial<EventInput> }) =>
      api.updateEvent(id, patch),
    invalidates: [keys.events],
    idempotent: "sets the fields it names to the values it carries, however often it lands.",
    // The rows on an event save as you leave them, so the value should settle
    // the moment you do rather than blink back to the old one and forward
    // again. A refusal — an end before its start — rolls this back, and the
    // toast says which.
    optimistic: (qc, { id, patch }) => {
      const apply = (e: Event): Event => ({ ...e, ...patch }) as Event;
      qc.setQueryData<Event>(keys.event(id), (e) => (e ? apply(e) : e));
      qc.setQueryData<Event[]>(keys.events, (list) =>
        list?.map((e) => (e.id === id ? apply(e) : e)),
      );
    },
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
    // A checkbox that waits for a round trip is a checkbox somebody taps
    // twice.
    optimistic: (qc, { id, item, done }) => {
      const tick = (e: Event): Event => ({
        ...e,
        checklist: e.checklist.map((c) => (c.id === item ? { ...c, done } : c)),
      });
      qc.setQueryData<Event>(keys.event(id), (e) => (e ? tick(e) : e));
      qc.setQueryData<Event[]>(keys.events, (list) =>
        list?.map((e) => (e.id === id ? tick(e) : e)),
      );
    },
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
  startChallenge: defineWrite({
    mutationKey: ["challenge", "start"],
    mutationFn: (template: string) => api.startChallenge(template),
    invalidates: [keys.challenge],
    // Starting a second one is refused by the server, so a replay after a
    // dropped response finds the one it already made.
    idempotent: "one challenge at a time, enforced by a unique index on the couple.",
  }),
  leaveChallenge: defineWrite({
    mutationKey: ["challenge", "leave"],
    mutationFn: () => api.leaveChallenge(),
    invalidates: [keys.challenge],
    idempotent: "leaving one that has already gone leaves the same nothing behind.",
  }),
  challengeDay: defineWrite({
    mutationKey: ["challenge", "day"],
    mutationFn: ({ n, patch }: { n: number; patch: Partial<ChallengeDay> }) =>
      api.challengeDay(n, patch),
    invalidates: [keys.challenge],
    idempotent: "a patch of one numbered day; the same patch twice is the same day.",
    // Same reasoning as the checklist: a day you mark should look marked.
    // Only this person's own mark moves — the partner's is theirs (DEC-30).
    optimistic: (qc, { n, patch }) => {
      qc.setQueryData<Challenge>(keys.challenge, (c) =>
        c ? { ...c, days: c.days.map((d) => (d.n === n ? { ...d, ...patch } : d)) } : c,
      );
    },
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
  // Keeping a moment and attaching a picture to it are one act to the person
  // doing it, and three steps underneath: create the memory, ask for
  // permission to upload, put the file where the permission points, then say
  // it landed. One write so a half-finished upload cannot leave a memory that
  // claims a photo it has not got.
  addMemoryWithPhoto: defineWrite({
    mutationKey: ["memories", "add-with-photo"],
    mutationFn: async ({ memory, photo }: { memory: Omit<Memory, "id">; photo?: File | null }) => {
      const saved = await api.addMemory(memory);
      if (!photo) return saved;

      const ticket = await api.photoTicket(saved.id);
      const form = new FormData();
      // Exactly what the server signed, then the file. Nothing added, nothing
      // renamed — the signature covers this list.
      for (const [key, value] of Object.entries(ticket.fields)) form.append(key, value);
      form.append("file", photo);

      const upload = await fetch(ticket.upload_url, { method: "POST", body: form });
      if (!upload.ok) {
        // The moment is saved; only the picture failed. Say which, rather
        // than letting it read as having lost the whole thing.
        throw new Error("The moment was saved, but the photo didn’t upload.");
      }
      return api.attachPhoto(saved.id);
    },
    invalidates: [keys.memories],
    onlineOnly: true,
  }),
  removePhoto: defineWrite({
    mutationKey: ["memories", "photo", "remove"],
    mutationFn: (id: string) => api.removePhoto(id),
    invalidates: [keys.memories],
    idempotent: "removing a photo that is already gone leaves the same memory behind.",
  }),
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
