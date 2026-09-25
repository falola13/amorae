"use client";

import type { UseQueryResult } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { Main } from "@/components/layout/screen";
import { GENERIC_ERROR_MESSAGE, NOT_AVAILABLE_CODE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { LoadProblem, Skeleton } from "./kit";

type AnyQuery = UseQueryResult<unknown, unknown>;
type DataOf<Q> = Q extends UseQueryResult<infer D, unknown> ? D : never;
type DataTuple<Qs extends readonly AnyQuery[]> = { [K in keyof Qs]: DataOf<Qs[K]> };

/**
 * Central loading/error/ready switch for a set of queries; each failure only
 * takes this component's space, not the whole screen. Cached data always wins
 * over a failed background refetch.
 */
export function QueryState<const Qs extends readonly AnyQuery[]>({
  queries,
  loading,
  frame = (notice) => notice,
  children,
}: {
  queries: Qs;
  /** Shown while loading. Defaults to a text skeleton. */
  loading?: ReactNode;
  /** Wraps a failure or offline notice, e.g. `inPage`. */
  frame?: (notice: ReactNode) => ReactNode;
  children: (...data: DataTuple<Qs>) => ReactNode;
}) {
  const missing = queries.filter((q) => q.data === undefined);
  const retry = () => missing.forEach((q) => void q.refetch());

  const failed = missing.find((q) => q.isError);
  if (failed) return <>{frame(<LoadFailure error={failed.error} onRetry={retry} />)}</>;
  if (missing.some((q) => q.fetchStatus === "paused")) {
    // Not an error: React Query loads it by itself when the connection returns.
    return (
      <>
        {frame(
          <LoadProblem
            tone="quiet"
            icon="offline"
            title="You’re offline"
            text="This will load when you’re back online."
          />,
        )}
      </>
    );
  }
  if (missing.length > 0) return <>{loading ?? <Skeleton />}</>;

  return <>{children(...(queries.map((q) => q.data) as DataTuple<Qs>))}</>;
}

/** Frame for a QueryState that is a whole page body: padded, and filling the height so the tab bar stays at the bottom. */
export const inPage = (notice: ReactNode) => (
  <Main>
    <div className="pt-4">{notice}</div>
  </Main>
);

/** The inline notice for a failed load, worded for its cause. Retry only where retrying can help. */
export function LoadFailure({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  if (isApiError(error) && error.code === NOT_AVAILABLE_CODE) {
    return (
      <LoadProblem
        tone="quiet"
        icon="clock"
        title="Not available yet"
        text="This part of Amorae is still being built."
      />
    );
  }
  if (isApiError(error) && error.status === 404) {
    // The API's own "not found" (deleted, or another couple's): asking again won't bring it back.
    return (
      <LoadProblem tone="quiet" icon="alert" title="This isn’t here anymore" text={error.message} />
    );
  }
  const message = isApiError(error) ? error.message : GENERIC_ERROR_MESSAGE;
  return (
    <LoadProblem
      tone="error"
      icon="alert"
      title="This didn’t load"
      text={message}
      onRetry={onRetry}
    />
  );
}
