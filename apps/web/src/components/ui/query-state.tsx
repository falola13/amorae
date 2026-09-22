"use client";

import type { UseQueryResult } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { ErrorState, Skeleton } from "./kit";

type AnyQuery = UseQueryResult<unknown, unknown>;
type DataOf<Q> = Q extends UseQueryResult<infer D, unknown> ? D : never;
type DataTuple<Qs extends readonly AnyQuery[]> = { [K in keyof Qs]: DataOf<Qs[K]> };

/**
 * The one place a screen decides between loading, failed and ready, so no
 * page can leave a failed request as a skeleton that spins forever.
 *
 *   <QueryState queries={[goal, couple]}>
 *     {(g, c) => <GoalDetail goal={g} partner={c.partner} />}
 *   </QueryState>
 *
 * Keep the page chrome (TopBar, Main, title) outside, so it stays put while
 * the content area switches. Cached data always wins: a background refetch
 * that fails keeps showing what was loaded.
 */
export function QueryState<const Qs extends readonly AnyQuery[]>({
  queries,
  loading,
  children,
}: {
  queries: Qs;
  /** Shown while loading. Defaults to a text skeleton. */
  loading?: ReactNode;
  children: (...data: DataTuple<Qs>) => ReactNode;
}) {
  const missing = queries.filter((q) => q.data === undefined);
  const retry = () => missing.forEach((q) => void q.refetch());

  const failed = missing.find((q) => q.isError);
  if (failed) {
    return (
      <ErrorState
        title="This didn’t load"
        text={isApiError(failed.error) ? failed.error.message : GENERIC_ERROR_MESSAGE}
        onRetry={retry}
      />
    );
  }
  if (missing.some((q) => q.fetchStatus === "paused")) {
    return <ErrorState title="You’re offline" text="This will load when you’re back online." onRetry={retry} />;
  }
  if (missing.length > 0) return <>{loading ?? <Skeleton />}</>;

  return <>{children(...(queries.map((q) => q.data) as DataTuple<Qs>))}</>;
}
