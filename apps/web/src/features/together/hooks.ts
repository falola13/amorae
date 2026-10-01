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
export const useActiveChallenges = () =>
  useQuery({ queryKey: keys.challengeActive, queryFn: api.challengeActive });
export const useChallengeById = (id: string) =>
  useQuery({
    queryKey: keys.challengeById(id),
    queryFn: () => api.challengeById(id),
    enabled: !!id,
  });
export const usePastChallenges = () =>
  useQuery({ queryKey: keys.challengePast, queryFn: api.challengePast });
export const useJournal = () => useQuery({ queryKey: keys.journal, queryFn: api.journal });
export const useAppreciations = () =>
  useQuery({ queryKey: keys.appreciations, queryFn: api.appreciations });
export const useMemories = () => useQuery({ queryKey: keys.memories, queryFn: api.memories });
export const useMilestones = () => useQuery({ queryKey: keys.milestones, queryFn: api.milestones });

// Writes: each is defined once in ./writes.ts. Errors, offline pausing and
// resuming after a restart are handled centrally (lib/query).
export const useSaveEvent = () => useWrite(w.saveEvent);
export const usePatchEvent = () => useWrite(w.patchEvent);
export const useCompleteEvent = () => useWrite(w.completeEvent);
export const useEventOutcome = () => useWrite(w.eventOutcome);
export const useDeleteEvent = () => useWrite(w.deleteEvent);
export const useChecklist = () => useWrite(w.checklist);
export const useCreateGoal = () => useWrite(w.createGoal);
export const useUpdateGoal = () => useWrite(w.updateGoal);
export const useAddProgress = () => useWrite(w.addProgress);
export const useChallengeTemplates = () =>
  useQuery({ queryKey: keys.challengeTemplates, queryFn: api.challengeTemplates });
export const useStartChallenge = () => useWrite(w.startChallenge);
export const useStartCustomChallenge = () => useWrite(w.startCustomChallenge);
export const useUpdateChallenge = () => useWrite(w.updateChallenge);
export const useChallengePlan = () => useWrite(w.challengePlan);
export const useLeaveChallenge = () => useWrite(w.leaveChallenge);
export const useChallengeDay = () => useWrite(w.challengeDay);
export const useChallengeReflection = () => useWrite(w.challengeReflection);
export const useAddJournal = () => useWrite(w.addJournal);
export const useUpdateJournal = () => useWrite(w.updateJournal);
export const useDeleteJournal = () => useWrite(w.deleteJournal);
export const useSendAppreciation = () => useWrite(w.sendAppreciation);
export const useUndoAppreciation = () => useWrite(w.undoAppreciation);
export const useAddMemory = () => useWrite(w.addMemory);
export const useAddMemoryWithPhoto = () => useWrite(w.addMemoryWithPhoto);
export const useRemovePhoto = () => useWrite(w.removePhoto);
export const useUploadPhoto = () => useWrite(w.uploadPhoto);
export const useDeleteMemory = () => useWrite(w.deleteMemory);
export const useUpdateMemory = () => useWrite(w.updateMemory);
export const useDeleteMilestone = () => useWrite(w.deleteMilestone);
export const useNudge = () => useWrite(w.nudge);
export const useAddMilestone = () => useWrite(w.addMilestone);
