import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { anniversaryApi as api } from "./api";

export const anniversaryWrites = {
  // Like a nudge: only meaningful now, so never queued for later.
  cheers: defineWrite({
    mutationKey: ["anniversary", "cheers"],
    mutationFn: () => api.cheers(),
    invalidates: [],
    onlineOnly: true,
    // The glass still clinks on screen; the page says whether it reached them.
    handlesError: true,
  }),
  writeLetter: defineWrite({
    mutationKey: ["anniversary", "letter"],
    mutationFn: (text: string) => api.writeLetter(text),
    invalidates: [keys.anniversary],
    idempotent: "the letter is replaced whole; the same text kept twice is the same letter.",
  }),
};
