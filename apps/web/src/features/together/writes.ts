import type { EventInput, GoalInput } from "@/lib/api/schemas";
import type {
  Challenge,
  ChallengeDayPatch,
  Event,
  JournalEntry,
  Memory,
  Milestone,
  StartChallengeInput,
} from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { togetherApi as api, type MemoryText } from "./api";

/** Ten megabytes, the cap Q-06 chose. Checked before anything is sent. */
export const MAX_PHOTO_BYTES = 10 * 1024 * 1024;

/** Ask for permission, upload, then record it — the three steps only work together. */
async function putPhoto(memoryID: string, photo: File): Promise<Memory> {
  const ticket = await api.photoTicket(memoryID);
  const form = new FormData();
  // Exactly what the server signed: the signature covers this list.
  for (const [key, value] of Object.entries(ticket.fields)) form.append(key, value);
  form.append("file", photo);

  const upload = await fetch(ticket.upload_url, { method: "POST", body: form });
  if (!upload.ok) {
    // The body says why; the status alone cannot tell a wrong cloud name
    // from a refused signature.
    throw new Error(`photo upload refused (${upload.status}): ${await upload.text()}`);
  }
  return api.attachPhoto(memoryID);
}

// Every write the Together feature makes, declared once (see lib/query/mutations.ts).
// Why a client key makes a create safe to queue. One sentence, used by every
// write that needs it, because it is the same sentence each time.
const K =
  "the API keeps the reply against the key, so a second send replays the first answer instead of creating a second row.";

export const togetherWrites = {
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  saveEvent: defineWrite({
    mutationKey: ["events", "save"],
    mutationFn: ({
      id,
      input,
      idempotencyKey,
    }: {
      id?: string;
      input: EventInput;
      idempotencyKey?: string;
    }) => (id ? api.updateEvent(id, input) : api.createEvent(input, idempotencyKey)),
    invalidates: [keys.events],
    // Updating was always idempotent; creating is what needed the key.
    keyed: K,
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
    mutationFn: ({ idempotencyKey, ...g }: GoalInput & { idempotencyKey?: string }) =>
      api.createGoal(g, idempotencyKey),
    invalidates: [keys.goals],
    keyed: K,
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
    mutationFn: ({
      id,
      amount,
      idempotencyKey,
    }: {
      id: string;
      amount: number;
      idempotencyKey?: string;
    }) => api.progress(id, amount, idempotencyKey),
    invalidates: [keys.goals],
    // The one that most needed this: it adds an amount rather than setting
    // one, so a replay used to double-count. The key is what stops that.
    keyed: K,
  }),
  startChallenge: defineWrite({
    mutationKey: ["challenge", "start"],
    mutationFn: (input: StartChallengeInput) => api.startChallenge(input),
    invalidates: [keys.challenge, keys.timelineAll],
    // Starting a second one is refused by the server, so a replay after a
    // dropped response finds the one it already made.
    idempotent: "one challenge at a time, enforced by a unique index on the couple.",
  }),
  // The make-our-own form shows the server's field errors beside the fields, so no toast on top.
  startCustomChallenge: defineWrite({
    mutationKey: ["challenge", "start-custom"],
    mutationFn: (input: StartChallengeInput) => api.startChallenge(input),
    invalidates: [keys.challenge, keys.timelineAll],
    handlesError: true,
    onlineOnly: true,
  }),
  leaveChallenge: defineWrite({
    mutationKey: ["challenge", "leave"],
    mutationFn: () => api.leaveChallenge(),
    invalidates: [keys.challenge, keys.timelineAll],
    idempotent: "leaving one that has already gone leaves the same nothing behind.",
  }),
  challengeDay: defineWrite({
    mutationKey: ["challenge", "day"],
    mutationFn: ({ n, patch }: { n: number; patch: ChallengeDayPatch }) =>
      api.challengeDay(n, patch),
    invalidates: [keys.challenge, keys.timelineAll],
    idempotent: "a patch of one numbered day; the same patch twice is the same day.",
    // Same reasoning as the checklist: a day you mark should look marked.
    // Only this person's own mark moves — the partner's is theirs (DEC-30).
    optimistic: (qc, { n, patch }) => {
      qc.setQueryData<Challenge>(keys.challenge, (c) =>
        c ? { ...c, days: c.days.map((d) => (d.n === n ? { ...d, ...patch } : d)) } : c,
      );
    },
  }),
  challengeReflection: defineWrite({
    mutationKey: ["challenge", "reflection"],
    mutationFn: ({ id, text }: { id: string; text: string }) => api.challengeReflection(id, text),
    invalidates: [keys.challenge],
    idempotent: "the reflection is replaced, so the same text sent twice is the same reflection.",
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  addJournal: defineWrite({
    mutationKey: ["journal", "add"],
    mutationFn: ({
      tag,
      text,
      idempotencyKey,
    }: {
      tag: JournalEntry["tag"];
      text: string;
      idempotencyKey?: string;
    }) => api.addJournal(tag, text, idempotencyKey),
    invalidates: [keys.journal],
    keyed: K,
  }),
  // Author-only on the server; the partner's entries never offer it.
  updateJournal: defineWrite({
    mutationKey: ["journal", "update"],
    mutationFn: ({ id, tag, text }: { id: string; tag: JournalEntry["tag"]; text: string }) =>
      api.updateJournal(id, tag, text),
    invalidates: [keys.journal],
    idempotent: "the same tag and text set twice is the same entry.",
  }),
  deleteJournal: defineWrite({
    mutationKey: ["journal", "delete"],
    mutationFn: (id: string) => api.deleteJournal(id),
    invalidates: [keys.journal],
    idempotent: "deleting something already gone leaves the same nothing behind.",
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  sendAppreciation: defineWrite({
    mutationKey: ["appreciations", "send"],
    mutationFn: ({ text, idempotencyKey }: { text: string; idempotencyKey?: string }) =>
      api.sendAppreciation(text, idempotencyKey),
    invalidates: [keys.appreciations],
    keyed: K,
  }),
  // Only meaningful within seconds of sending, so it is never queued offline:
  // replayed hours later, it would delete a note the partner has already read.
  undoAppreciation: defineWrite({
    mutationKey: ["appreciations", "undo"],
    mutationFn: (id: string) => api.undoAppreciation(id),
    invalidates: [keys.appreciations],
    onlineOnly: true,
  }),
  // A failed upload is not a failed write: the moment is already saved, so
  // report that and say the photo did not arrive.
  addMemoryWithPhoto: defineWrite({
    mutationKey: ["memories", "add-with-photo"],
    mutationFn: async ({
      memory,
      photo,
    }: {
      memory: Omit<Memory, "id">;
      photo?: File | null;
    }): Promise<{ memory: Memory; photoFailed: boolean }> => {
      const saved = await api.addMemory(memory);
      if (!photo) return { memory: saved, photoFailed: false };
      try {
        return { memory: await putPhoto(saved.id, photo), photoFailed: false };
      } catch (err) {
        console.error("photo upload failed", err);
        return { memory: saved, photoFailed: true };
      }
    },
    invalidates: [keys.memories],
    // A create with no key of its own, and a File cannot survive the queue
    // (FR-PWA-009).
    onlineOnly: true,
  }),
  uploadPhoto: defineWrite({
    mutationKey: ["memories", "photo", "upload"],
    mutationFn: ({ id, photo }: { id: string; photo: File }) => putPhoto(id, photo),
    invalidates: [keys.memories],
    // A File cannot survive the offline queue.
    onlineOnly: true,
  }),
  removePhoto: defineWrite({
    mutationKey: ["memories", "photo", "remove"],
    mutationFn: (id: string) => api.removePhoto(id),
    invalidates: [keys.memories],
    idempotent: "removing a photo that is already gone leaves the same memory behind.",
  }),
  deleteMemory: defineWrite({
    mutationKey: ["memories", "delete"],
    mutationFn: (id: string) => api.deleteMemory(id),
    invalidates: [keys.memories],
    idempotent: "deleting something already gone leaves the same nothing behind.",
  }),
  // The words and the day; the photo has its own writes above.
  updateMemory: defineWrite({
    mutationKey: ["memories", "update"],
    mutationFn: ({ id, title, date, location, note }: MemoryText & { id: string }) =>
      api.updateMemory(id, { title, date, location, note }),
    invalidates: [keys.memories],
    idempotent: "the same fields set twice leave the same moment.",
  }),
  addMemory: defineWrite({
    mutationKey: ["memories", "add"],
    mutationFn: ({ idempotencyKey, ...m }: Omit<Memory, "id"> & { idempotencyKey?: string }) =>
      api.addMemory(m, idempotencyKey),
    invalidates: [keys.memories],
    keyed: K,
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  // Nothing to write and nothing to reply to, so there is nothing to queue
  // either: a thought delivered tomorrow is a different thought.
  nudge: defineWrite({
    mutationKey: ["nudge"],
    mutationFn: () => api.nudge(),
    invalidates: [],
    onlineOnly: true,
    // The screen says what happened, including the refusals — that they are
    // asleep, or that today's are spent.
    handlesError: true,
  }),
  deleteMilestone: defineWrite({
    mutationKey: ["milestones", "delete"],
    mutationFn: (id: string) => api.deleteMilestone(id),
    invalidates: [keys.milestones],
    idempotent: "deleting something already gone leaves the same nothing behind.",
  }),
  addMilestone: defineWrite({
    mutationKey: ["milestones", "add"],
    mutationFn: ({ idempotencyKey, ...m }: Omit<Milestone, "id"> & { idempotencyKey?: string }) =>
      api.addMilestone(m, idempotencyKey),
    invalidates: [keys.milestones],
    keyed: K,
  }),
};
