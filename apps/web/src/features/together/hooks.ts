"use client";

import { useQuery } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import { useWrite } from "@/lib/query/mutations";
import { togetherApi as api } from "./api";
import { togetherWrites as w } from "./writes";

// Reads
export const useEvents = () => useQuery({ queryKey: keys.events, queryFn: api.events });
export const useEvent = (id: string) =>
  useQuery({ queryKey: keys.event(id), queryFn: () => api.event(id), enabled: !!id });
export const useGoals = () => useQuery({ queryKey: keys.goals, queryFn: api.goals });
export const useGoal = (id: string) =>
  useQuery({ queryKey: keys.goal(id), queryFn: () => api.goal(id), enabled: !!id });
export const useChallenge = () => useQuery({ queryKey: keys.challenge, queryFn: api.challenge });
export const useJournal = () => useQuery({ queryKey: keys.journal, queryFn: api.journal });
export const useAppreciations = () =>
  useQuery({ queryKey: keys.appreciations, queryFn: api.appreciations });
export const useMemories = () => useQuery({ queryKey: keys.memories, queryFn: api.memories });
export const useMilestones = () => useQuery({ queryKey: keys.milestones, queryFn: api.milestones });

// Writes: each is defined once in ./writes.ts. Errors, offline pausing and
// resuming after a restart are handled centrally (lib/query).
export const useSaveEvent = () => useWrite(w.saveEvent);
export const useCompleteEvent = () => useWrite(w.completeEvent);
export const useChecklist = () => useWrite(w.checklist);
export const useCreateGoal = () => useWrite(w.createGoal);
export const useAddProgress = () => useWrite(w.addProgress);
export const useChallengeDay = () => useWrite(w.challengeDay);
export const useAddJournal = () => useWrite(w.addJournal);
export const useSendAppreciation = () => useWrite(w.sendAppreciation);
export const useUndoAppreciation = () => useWrite(w.undoAppreciation);
export const useAddMemory = () => useWrite(w.addMemory);
export const useAddMilestone = () => useWrite(w.addMilestone);
