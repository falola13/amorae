'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { api, flushQueue, readQueue } from './api';
import type { Event, Goal, JournalEntry, Memory, Milestone, NotificationPrefs, PrayerPoint } from './types';

export const keys = {
  me: ['me'] as const, couple: ['couple'] as const, week: ['week'] as const, history: ['history'] as const,
  events: ['events'] as const, goals: ['goals'] as const, challenge: ['challenge'] as const, journal: ['journal'] as const,
  appreciations: ['appreciations'] as const, memories: ['memories'] as const, milestones: ['milestones'] as const, prefs: ['prefs'] as const,
};

export const useMe = () => useQuery({ queryKey: keys.me, queryFn: api.me });
export const useCouple = () => useQuery({ queryKey: keys.couple, queryFn: api.couple });
export const useWeek = () => useQuery({ queryKey: keys.week, queryFn: api.currentWeek });
export const useHistory = () => useQuery({ queryKey: keys.history, queryFn: api.history });
export const useWeekById = (id: string) => useQuery({ queryKey: ['week', id], queryFn: () => api.week(id) });
export const useEvents = () => useQuery({ queryKey: keys.events, queryFn: api.events });
export const useEvent = (id: string) => useQuery({ queryKey: ['event', id], queryFn: () => api.event(id) });
export const useGoals = () => useQuery({ queryKey: keys.goals, queryFn: api.goals });
export const useGoal = (id: string) => useQuery({ queryKey: ['goal', id], queryFn: () => api.goal(id) });
export const useChallenge = () => useQuery({ queryKey: keys.challenge, queryFn: api.challenge });
export const useJournal = () => useQuery({ queryKey: keys.journal, queryFn: api.journal });
export const useAppreciations = () => useQuery({ queryKey: keys.appreciations, queryFn: api.appreciations });
export const useMemories = () => useQuery({ queryKey: keys.memories, queryFn: api.memories });
export const useMilestones = () => useQuery({ queryKey: keys.milestones, queryFn: api.milestones });
export const usePrefs = () => useQuery({ queryKey: keys.prefs, queryFn: api.prefs });

function useInvalidating<TArgs extends unknown[], R>(fn: (...args: TArgs) => Promise<R>, invalidate: ReadonlyArray<readonly string[]>) {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (args: TArgs) => fn(...args), onSettled: () => invalidate.forEach((k) => qc.invalidateQueries({ queryKey: k })) });
}

/** Optimistic: the check draws in immediately, the API call follows. */
export function useSetCompleted() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ pointId, done }: { pointId: string; done: boolean }) => api.setCompleted(pointId, done),
    onMutate: async ({ pointId, done }) => {
      await qc.cancelQueries({ queryKey: keys.week });
      const prev = qc.getQueryData<Awaited<ReturnType<typeof api.currentWeek>>>(keys.week);
      if (prev) qc.setQueryData(keys.week, { ...prev, myCompleted: done ? Array.from(new Set([...prev.myCompleted, pointId])) : prev.myCompleted.filter((x) => x !== pointId) });
      return { prev };
    },
    onError: (_e, _v, ctx) => { if (ctx?.prev) qc.setQueryData(keys.week, ctx.prev); },
    onSettled: () => qc.invalidateQueries({ queryKey: keys.week }),
  });
}

export const useSavePoints = () => useInvalidating((points: PrayerPoint[]) => api.savePoints(points), [keys.week]);
export const usePublish = () => useInvalidating(() => api.publish(), [keys.week, keys.history]);
export const useSaveReflection = () => useInvalidating((weekId: string, text: string) => api.saveReflection(weekId, text), [keys.history, keys.week]);
export const useDemoWeek = () => useInvalidating((mode: 'partner' | 'mine' | 'waiting') => api.setDemoWeek(mode), [keys.week]);

export const useSaveEvent = () => useInvalidating((e: Partial<Event> & { title: string; date: string }) => api.saveEvent(e), [keys.events]);
export const useToggleChecklist = () => useInvalidating((eventId: string, itemId: string, done: boolean) => api.toggleChecklist(eventId, itemId, done), [keys.events]);
export const useCompleteEvent = () => useInvalidating((id: string, done: boolean) => api.completeEvent(id, done), [keys.events]);
export const useSaveGoal = () => useInvalidating((g: Partial<Goal> & { title: string }) => api.saveGoal(g), [keys.goals]);
export const useAddProgress = () => useInvalidating((goalId: string, amount: number) => api.addProgress(goalId, amount), [keys.goals]);
export const useChallengeDay = () => useInvalidating((n: number, patch: { done?: boolean; skipped?: boolean }) => api.setChallengeDay(n, patch), [keys.challenge]);
export const useAddJournal = () => useInvalidating((tag: JournalEntry['tag'], text: string) => api.addJournal(tag, text), [keys.journal]);
export const useSendAppreciation = () => useInvalidating((text: string) => api.sendAppreciation(text), [keys.appreciations]);
export const useUndoAppreciation = () => useInvalidating((id: string) => api.undoAppreciation(id), [keys.appreciations]);
export const useAddMemory = () => useInvalidating((m: Omit<Memory, 'id'>) => api.addMemory(m), [keys.memories]);
export const useAddMilestone = () => useInvalidating((m: Omit<Milestone, 'id'>) => api.addMilestone(m), [keys.milestones]);
export const useSavePrefs = () => useInvalidating((p: Partial<NotificationPrefs>) => api.savePrefs(p), [keys.prefs]);
export const useUpdateProfile = () => useInvalidating((p: { name?: string; email?: string; timezone?: string }) => api.updateProfile(p), [keys.couple, keys.me]);

/* ---------- connectivity ---------- */

export type NetState = { online: boolean; pending: number; justSynced: number | null };

export function useNetwork(): NetState {
  const qc = useQueryClient();
  const [online, setOnline] = useState(true);
  const [pending, setPending] = useState(0);
  const [justSynced, setJustSynced] = useState<number | null>(null);
  useEffect(() => {
    setOnline(navigator.onLine); setPending(readQueue().length);
    const up = async () => {
      setOnline(true);
      const n = await flushQueue();
      setPending(0);
      if (n > 0) { setJustSynced(n); qc.invalidateQueries(); setTimeout(() => setJustSynced(null), 3500); }
    };
    const down = () => setOnline(false);
    const tick = () => setPending(readQueue().length);
    window.addEventListener('online', up); window.addEventListener('offline', down); window.addEventListener('storage', tick);
    const t = setInterval(tick, 1500);
    return () => { window.removeEventListener('online', up); window.removeEventListener('offline', down); window.removeEventListener('storage', tick); clearInterval(t); };
  }, [qc]);
  return { online, pending, justSynced };
}

/** iOS standalone detection and install prompt (Android) support. */
export function useInstall() {
  const [standalone, setStandalone] = useState(false);
  const [ios, setIos] = useState(false);
  const [prompt, setPrompt] = useState<null | (() => Promise<void>)>(null);
  useEffect(() => {
    setStandalone(window.matchMedia('(display-mode: standalone)').matches || (navigator as unknown as { standalone?: boolean }).standalone === true);
    setIos(/iphone|ipad|ipod/i.test(navigator.userAgent));
    const handler = (raw: globalThis.Event) => { const e = raw as globalThis.Event & { prompt?: () => Promise<void> }; e.preventDefault(); setPrompt(() => async () => { await e.prompt?.(); }); };
    window.addEventListener('beforeinstallprompt', handler);
    return () => window.removeEventListener('beforeinstallprompt', handler);
  }, []);
  return { standalone, ios, prompt };
}

export function usePush() {
  const [state, setState] = useState<'unsupported' | 'default' | 'granted' | 'denied'>('default');
  useEffect(() => { setState(!('Notification' in window) || !('serviceWorker' in navigator) ? 'unsupported' : Notification.permission); }, []);
  const request = async () => {
    if (!('Notification' in window)) return 'unsupported' as const;
    const p = await Notification.requestPermission();
    setState(p);
    if (p === 'granted') {
      try {
        const reg = await navigator.serviceWorker.ready;
        // With the Go API: subscribe with the VAPID public key and POST to /notifications/subscribe
        await reg.pushManager.getSubscription();
      } catch { /* subscription is wired once the API is live */ }
    }
    return p;
  };
  return { state, request };
}
