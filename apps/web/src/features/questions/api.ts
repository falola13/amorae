import { http } from "@/lib/api/http";
import type { QuestionEntry, QuestionPage } from "@/lib/api/types";

export const questionsApi = {
  today: () => http.get<QuestionEntry>("/questions/today").then((r) => r.data),
  // PUT: answering again the same day is an edit, not a second answer.
  answer: (text: string) =>
    http.put<QuestionEntry>("/questions/today", { text }).then((r) => r.data),
  past: (before?: string) =>
    http.get<QuestionPage>("/questions", { params: { before, limit: 20 } }).then((r) => r.data),
};
