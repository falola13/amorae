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

/**
 * Ten megabytes, the cap Q-06 chose. Checked before anything is sent, so
 * somebody choosing a 40MB photo is told at once rather than after a long
 * upload that was never going to be accepted.
 */
export const MAX_PHOTO_BYTES = 10 * 1024 * 1024;

/**
 * Put one file where a memory's picture goes.
 *
 * Three calls that only make sense together: ask for permission, use it, and
 * record that it was used. Written once because both the composer and a
 * memory that already exists need exactly this, and a second copy would be
 * the copy that forgets the last step.
 */
async function putPhoto(memoryID: string, photo: File): Promise<Memory> {
  const ticket = await api.photoTicket(memoryID);
  const form = new FormData();
  // Exactly what the server signed, then the file. Nothing added, nothing
  // renamed — the signature covers this list.
  for (const [key, value] of Object.entries(ticket.fields)) form.append(key, value);
  form.append("file", photo);

  const upload = await fetch(ticket.upload_url, { method: "POST", body: form });
  if (!upload.ok) {
    // Cloudinary says why in the body, and its reasons are specific enough
    // to act on — a wrong cloud name and a refused signature look identical
    // from a status code alone.
    throw new Error(`photo upload refused (${upload.status}): ${await upload.text()}`);
  }
  return api.attachPhoto(memoryID);
}

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
  // Keeping a moment and attaching a picture to it are one act to the person
  // doing it, and three steps underneath: create the memory, ask for
  // permission to upload, put the file where the permission points, then say
  // it landed.
  //
  // The steps fail separately, and for a while this reported the whole thing
  // as failed when only the last part was — leaving the moment saved, the
  // sheet open, and the text still in it, so pressing the button again made
  // a second copy of a memory that had been kept the first time.
  //
  // So a failed upload is not a failed write. The moment was kept; say so,
  // and say the picture did not arrive, which is the one thing left to do
  // something about.
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
        // Kept out of the caller's way but not thrown away: whatever
        // Cloudinary refused is the only clue to why, and the screen can
        // only say that it happened.
        console.error("photo upload failed", err);
        return { memory: saved, photoFailed: true };
      }
    },
    invalidates: [keys.memories],
    // A create with no key of its own: replayed after a dropped response it
    // would make a second one. Online-only until the endpoint takes an
    // idempotency key (FR-PWA-009). A File would not survive the queue
    // either.
    onlineOnly: true,
  }),
  // Adding a picture to a moment already kept — the way back from an upload
  // that failed, and the way to change one's mind about which photo it was.
  uploadPhoto: defineWrite({
    mutationKey: ["memories", "photo", "upload"],
    mutationFn: ({ id, photo }: { id: string; photo: File }) => putPhoto(id, photo),
    invalidates: [keys.memories],
    // A File cannot be put in the queue and taken out again later, so this
    // is a write that happens now or says it did not.
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
  addMemory: defineWrite({
    mutationKey: ["memories", "add"],
    mutationFn: (m: Omit<Memory, "id">) => api.addMemory(m),
    invalidates: [keys.memories],
    onlineOnly: true,
  }),
  // A create with no key of its own: replayed after a dropped response it would
  // make a second one. Online-only until the endpoint takes an idempotency key
  // (FR-PWA-009), which is a smaller loss than silent duplicates.
  deleteMilestone: defineWrite({
    mutationKey: ["milestones", "delete"],
    mutationFn: (id: string) => api.deleteMilestone(id),
    invalidates: [keys.milestones],
    idempotent: "deleting something already gone leaves the same nothing behind.",
  }),
  addMilestone: defineWrite({
    mutationKey: ["milestones", "add"],
    mutationFn: (m: Omit<Milestone, "id">) => api.addMilestone(m),
    invalidates: [keys.milestones],
    onlineOnly: true,
  }),
};
