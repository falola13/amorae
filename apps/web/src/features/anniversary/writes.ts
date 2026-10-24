import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { anniversaryApi as api } from "./api";

export const anniversaryWrites = {
  writeLetter: defineWrite({
    mutationKey: ["anniversary", "letter"],
    mutationFn: (text: string) => api.writeLetter(text),
    invalidates: [keys.anniversary],
    idempotent: "the letter is replaced whole; the same text kept twice is the same letter.",
  }),
};
