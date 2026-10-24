import { http } from "@/lib/api/http";
import type { Anniversary, AnniversaryLetter } from "@/lib/api/types";

export const anniversaryApi = {
  get: () => http.get<Anniversary>("/anniversary").then((r) => r.data),
  /** Writes this person's letter for the coming anniversary, or replaces it. */
  writeLetter: (text: string) =>
    http.put<AnniversaryLetter>("/anniversary/letter", { text }).then((r) => r.data),
};
