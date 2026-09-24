import { apiPath, http } from "@/lib/api/http";
import type {
  Appreciation,
  Challenge,
  ChallengeDay,
  Event,
  Goal,
  JournalEntry,
  Memory,
  Milestone,
} from "@/lib/api/types";
import type { EventInput, GoalInput } from "@/lib/api/schemas";

export const togetherApi = {
  events: () => http.get<Event[]>("/events").then((r) => r.data),
  event: (id: string) => http.get<Event>(apiPath`/events/${id}`).then((r) => r.data),
  createEvent: (e: EventInput) => http.post<Event>("/events", e).then((r) => r.data),
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
  createGoal: (g: GoalInput) => http.post<Goal>("/goals", g).then((r) => r.data),
  updateGoal: (id: string, patch: Partial<GoalInput> & { done?: boolean }) =>
    http.patch<Goal>(apiPath`/goals/${id}`, patch).then((r) => r.data),
  progress: (id: string, amount: number) =>
    http.post<Goal>(apiPath`/goals/${id}/progress`, { amount }).then((r) => r.data),

  challenge: () => http.get<Challenge>("/challenges/current").then((r) => r.data),
  challengeDay: (n: number, patch: Partial<ChallengeDay>) =>
    http.patch<Challenge>(apiPath`/challenges/current/days/${n}`, patch).then((r) => r.data),

  journal: () => http.get<JournalEntry[]>("/journal").then((r) => r.data),
  addJournal: (tag: JournalEntry["tag"], text: string) =>
    http.post<JournalEntry>("/journal", { tag, text }).then((r) => r.data),

  appreciations: () => http.get<Appreciation[]>("/appreciations").then((r) => r.data),
  sendAppreciation: (text: string) =>
    http.post<Appreciation>("/appreciations", { text }).then((r) => r.data),
  undoAppreciation: (id: string) =>
    http.delete(apiPath`/appreciations/${id}`).then(() => undefined),

  memories: () => http.get<Memory[]>("/memories").then((r) => r.data),
  addMemory: (m: Omit<Memory, "id">) => http.post<Memory>("/memories", m).then((r) => r.data),

  milestones: () => http.get<Milestone[]>("/milestones").then((r) => r.data),
  addMilestone: (m: Omit<Milestone, "id">) =>
    http.post<Milestone>("/milestones", m).then((r) => r.data),
};
