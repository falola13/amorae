import { http } from "@/lib/api/http";
import type { Anniversary, AnniversaryLetter } from "@/lib/api/types";

export const anniversaryApi = {
  get: () => http.get<Anniversary>("/anniversary").then((r) => r.data),
  /** Writes this person's letter for the coming anniversary, or replaces it. */
  /** A raised glass: a nudge in the anniversary's own words, from the same allowance. */
  cheers: () => http.post<{ left: number }>("/nudge", { style: "cheers" }).then((r) => r.data),
  writeLetter: (text: string) =>
    http.put<AnniversaryLetter>("/anniversary/letter", { text }).then((r) => r.data),
};
