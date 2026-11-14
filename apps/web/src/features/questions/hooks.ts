"use client";

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import { useWrite } from "@/lib/query/mutations";
import { questionsApi } from "./api";
import { questionWrites } from "./writes";

/** Today's question, and what I'm allowed to see of it. */
export const useTodayQuestion = () =>
  useQuery({ queryKey: keys.questionToday, queryFn: questionsApi.today });

export function usePastQuestions() {
  return useInfiniteQuery({
    queryKey: keys.questionsPast,
    queryFn: ({ pageParam }: { pageParam?: string }) => questionsApi.past(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next ?? undefined,
  });
}

export const useAnswerQuestion = () => useWrite(questionWrites.answer);
