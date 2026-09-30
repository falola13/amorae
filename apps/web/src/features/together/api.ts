import { apiPath, http } from "@/lib/api/http";
import type {
  Appreciation,
  Challenge,
  ChallengeDayPatch,
  ChallengePast,
  ChallengeTemplate,
  StartChallengeInput,
  Event,
  Goal,
  JournalEntry,
  Memory,
  PhotoTicket,
  Milestone,
} from "@/lib/api/types";
import type { EventInput, GoalInput } from "@/lib/api/schemas";

/** A create's idempotency key, as a request header. The API stores the reply
 *  against it, so a queued write replayed after a dropped response gets the
 *  first answer back rather than making a second row (FR-PWA-009). */
export type MemoryText = Pick<Memory, "title" | "date" | "location" | "note">;

const withKey = (key?: string) => (key ? { headers: { "Idempotency-Key": key } } : undefined);

export const togetherApi = {
  events: () => http.get<Event[]>("/events").then((r) => r.data),
  event: (id: string) => http.get<Event>(apiPath`/events/${id}`).then((r) => r.data),
  createEvent: (e: EventInput, key?: string) =>
    http.post<Event>("/events", e, withKey(key)).then((r) => r.data),
  updateEvent: (id: string, e: Partial<EventInput>) =>
    http.patch<Event>(apiPath`/events/${id}`, e).then((r) => r.data),
  completeEvent: (id: string, done: boolean) =>
    (done
      ? http.post<Event>(apiPath`/events/${id}/complete`)
      : http.delete<Event>(apiPath`/events/${id}/complete`)
    ).then((r) => r.data),
  deleteEvent: (id: string) => http.delete(apiPath`/events/${id}`).then(() => undefined),
  checklist: (id: string, item: string, done: boolean) =>
    http.patch<Event>(apiPath`/events/${id}/checklist/${item}`, { done }).then((r) => r.data),

  goals: () => http.get<Goal[]>("/goals").then((r) => r.data),
  goal: (id: string) => http.get<Goal>(apiPath`/goals/${id}`).then((r) => r.data),
  createGoal: (g: GoalInput, key?: string) =>
    http.post<Goal>("/goals", g, withKey(key)).then((r) => r.data),
  updateGoal: (id: string, patch: Partial<GoalInput> & { done?: boolean }) =>
    http.patch<Goal>(apiPath`/goals/${id}`, patch).then((r) => r.data),
  progress: (id: string, amount: number, key?: string) =>
    http.post<Goal>(apiPath`/goals/${id}/progress`, { amount }, withKey(key)).then((r) => r.data),

  challenge: () => http.get<Challenge>("/challenges/current").then((r) => r.data),
  challengeTemplates: () =>
    http.get<ChallengeTemplate[]>("/challenges/templates").then((r) => r.data),
  challengePast: () => http.get<ChallengePast[]>("/challenges/past").then((r) => r.data),
  challengeById: (id: string) =>
    http.get<Challenge>(apiPath`/challenges/${id}`).then((r) => r.data),
  startChallenge: (input: StartChallengeInput) =>
    http.post<Challenge>("/challenges", input).then((r) => r.data),
  leaveChallenge: () => http.delete("/challenges/current").then(() => undefined),
  challengeDay: (n: number, patch: ChallengeDayPatch) =>
    http.patch<Challenge>(apiPath`/challenges/current/days/${n}`, patch).then((r) => r.data),
  /** Only once it is over; empty text removes it. */
  challengeReflection: (id: string, text: string) =>
    http.put<Challenge>(apiPath`/challenges/${id}/reflection`, { text }).then((r) => r.data),

  journal: () => http.get<JournalEntry[]>("/journal").then((r) => r.data),
  addJournal: (tag: JournalEntry["tag"], text: string, key?: string) =>
    http.post<JournalEntry>("/journal", { tag, text }, withKey(key)).then((r) => r.data),
  updateJournal: (id: string, tag: JournalEntry["tag"], text: string) =>
    http.patch<JournalEntry>(apiPath`/journal/${id}`, { tag, text }).then((r) => r.data),
  deleteJournal: (id: string) => http.delete(apiPath`/journal/${id}`).then(() => undefined),

  appreciations: () => http.get<Appreciation[]>("/appreciations").then((r) => r.data),
  sendAppreciation: (text: string, key?: string) =>
    http.post<Appreciation>("/appreciations", { text }, withKey(key)).then((r) => r.data),
  undoAppreciation: (id: string) =>
    http.delete(apiPath`/appreciations/${id}`).then(() => undefined),

  memories: () => http.get<Memory[]>("/memories").then((r) => r.data),
  addMemory: (m: Omit<Memory, "id">, key?: string) =>
    http.post<Memory>("/memories", m, withKey(key)).then((r) => r.data),
  photoTicket: (id: string) =>
    http.post<PhotoTicket>(apiPath`/memories/${id}/photo/ticket`).then((r) => r.data),
  /** Say the upload happened. No body: the server chose the only name it could go to. */
  attachPhoto: (id: string) => http.put<Memory>(apiPath`/memories/${id}/photo`).then((r) => r.data),
  removePhoto: (id: string) =>
    http.delete<Memory>(apiPath`/memories/${id}/photo`).then((r) => r.data),
  deleteMemory: (id: string) => http.delete(apiPath`/memories/${id}`).then(() => undefined),
  // Only the four fields: the API refuses unknown ones, and photo_url is one.
  updateMemory: (id: string, m: MemoryText) =>
    http.put<Memory>(apiPath`/memories/${id}`, m).then((r) => r.data),

  milestones: () => http.get<Milestone[]>("/milestones").then((r) => r.data),
  addMilestone: (m: Omit<Milestone, "id">, key?: string) =>
    http.post<Milestone>("/milestones", m, withKey(key)).then((r) => r.data),
  deleteMilestone: (id: string) => http.delete(apiPath`/milestones/${id}`).then(() => undefined),

  // How many more may be sent today, so the limit shows before it's reached.
  nudge: () => http.post<{ left: number }>("/nudge", {}).then((r) => r.data),
};
