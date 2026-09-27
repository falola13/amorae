"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { User } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { MAX_PHOTO_BYTES } from "@/features/together/writes";
import { coupleApi } from "./api";

// Same ten-megabyte cap Q-06 chose for memory photos (writes.ts), re-exported
// here so the profile screen doesn't reach across features for one constant.
export { MAX_PHOTO_BYTES };

/** Ask for permission, upload, then record it — the three steps only work together. */
async function putMyPhoto(photo: File): Promise<User> {
  const ticket = await coupleApi.photoTicket();
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
  return coupleApi.attachPhoto();
}

// The screen shows upload failures itself ("Photos aren't set up yet" or a
// generic retry), so both opt out of the global error toast. A File cannot
// survive the offline queue, so this is online-only like the memory photo writes.
export function useUploadMyPhoto() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: putMyPhoto,
    networkMode: "always",
    meta: { handlesError: true },
    onSuccess: (user) => {
      qc.setQueryData(keys.me, user);
      qc.invalidateQueries({ queryKey: keys.me });
      qc.invalidateQueries({ queryKey: keys.couple });
    },
  });
}
export function useRemoveMyPhoto() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.removePhoto,
    meta: { handlesError: true },
    onSuccess: (user) => {
      qc.setQueryData(keys.me, user);
      qc.invalidateQueries({ queryKey: keys.me });
      qc.invalidateQueries({ queryKey: keys.couple });
    },
  });
}

export const useMe = () => useQuery({ queryKey: keys.me, queryFn: coupleApi.me });
export const useCouple = () => useQuery({ queryKey: keys.couple, queryFn: coupleApi.couple });
export const useEndedCouples = () =>
  useQuery({ queryKey: keys.endedCouples, queryFn: coupleApi.endedCouples });

// Irreversible and starts a clock, so it's online-only (networkMode: always), not a resumable offline write.
export function useLeaveCouple() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.leaveCouple,
    networkMode: "always",
    meta: { handlesError: true },
    onSuccess: (ended) => {
      // Everything cached belonged to the ended space; drop rather than refetch (would mostly 404).
      qc.removeQueries();
      qc.setQueryData(keys.endedCouples, ended);
    },
  });
}

export function useCreateCouple() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.create,
    onSuccess: (c) => qc.setQueryData(keys.couple, c),
  });
}
// The join form shows a wrong code inline, so it opts out of the global error toast.
export function useJoinCouple() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.join,
    meta: { handlesError: true },
    onSuccess: (c) => qc.setQueryData(keys.couple, c),
  });
}
export function useOnboarding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.onboarding,
    onSuccess: (c) => qc.setQueryData(keys.couple, c),
  });
}
// The profile form maps field errors onto its inputs, so it opts out of the global error toast.
export function useUpdateProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.updateMe,
    meta: { handlesError: true },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.me });
      qc.invalidateQueries({ queryKey: keys.couple });
    },
  });
}
// Shows "wrong password" / "email taken" on the form itself, so no global toast.
export function useChangeEmail() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.changeEmail,
    meta: { handlesError: true },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.me });
      qc.invalidateQueries({ queryKey: keys.couple });
    },
  });
}
// The delete sheet shows a wrong confirmation or password on its input, so no global toast.
export const useDeleteAccount = () =>
  useMutation({ mutationFn: coupleApi.deleteMe, meta: { handlesError: true } });
export const useChangePassword = () =>
  useMutation({ mutationFn: coupleApi.changePassword, meta: { handlesError: true } });
export const useForgotPassword = () =>
  useMutation({ mutationFn: coupleApi.forgotPassword, meta: { handlesError: true } });
export const useResetPassword = () =>
  useMutation({
    mutationFn: (v: { token: string; new_password: string }) =>
      coupleApi.resetPassword(v.token, v.new_password),
    meta: { handlesError: true },
  });

export function useRegenerateInvite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.regenerateInvite,
    onSuccess: (c) => qc.setQueryData(keys.couple, c),
  });
}
// The couple sheet saves the couple and your role; both answer with the whole couple.
export function useUpdateCouple() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.updateCouple,
    meta: { handlesError: true },
    onSuccess: (c) => qc.setQueryData(keys.couple, c),
  });
}
export function useUpdateRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: coupleApi.updateRole,
    meta: { handlesError: true },
    onSuccess: (c) => qc.setQueryData(keys.couple, c),
  });
}
