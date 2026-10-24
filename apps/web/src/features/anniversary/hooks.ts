"use client";

import { useQuery } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import { useWrite } from "@/lib/query/mutations";
import { anniversaryApi as api } from "./api";
import { anniversaryWrites as w } from "./writes";

/** A 404 here is "no anniversary yet" (isAbsence), not a failure. */
export const useAnniversary = () => useQuery({ queryKey: keys.anniversary, queryFn: api.get });

export const useWriteLetter = () => useWrite(w.writeLetter);
export const useCheers = () => useWrite(w.cheers);
