"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import { coupleApi } from "./api";

export const useMe = () => useQuery({ queryKey: keys.me, queryFn: coupleApi.me });
export const useCouple = () => useQuery({ queryKey: keys.couple, queryFn: coupleApi.couple });

export function useCreateCouple() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: coupleApi.create, onSuccess: (c) => qc.setQueryData(keys.couple, c) });
}
// The join form shows a wrong code inline, so it opts out of the global error toast.
export function useJoinCouple() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: coupleApi.join, meta: { handlesError: true }, onSuccess: (c) => qc.setQueryData(keys.couple, c) });
}
export function useOnboarding() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: coupleApi.onboarding, onSuccess: (c) => qc.setQueryData(keys.couple, c) });
}
// The profile form maps field errors onto its inputs, so it opts out of the global error toast.
export function useUpdateProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.updateMe,
    meta: { handlesError: true },
    onSuccess: () => { qc.invalidateQueries({ queryKey: keys.me }); qc.invalidateQueries({ queryKey: keys.couple }); },
  });
}
// Shows "wrong password" / "email taken" on the form itself, so no global toast.
export function useChangeEmail() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.changeEmail,
    meta: { handlesError: true },
    onSuccess: () => { qc.invalidateQueries({ queryKey: keys.me }); qc.invalidateQueries({ queryKey: keys.couple }); },
  });
}
export const useDeleteAccount = () => useMutation({ mutationFn: coupleApi.deleteMe });
