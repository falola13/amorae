import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { questionsApi } from "./api";

export const questionWrites = {
  answer: defineWrite({
    mutationKey: ["questions", "answer"],
    mutationFn: (text: string) => questionsApi.answer(text),
    // Answering is also what unseals their answer, so the archive moves too.
    invalidates: [keys.questionToday, keys.questionsPast],
    idempotent: "one answer per person per day, so a replay rewrites the same text.",
  }),
};
