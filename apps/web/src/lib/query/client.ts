"use client";

import { MutationCache, QueryClient } from "@tanstack/react-query";

import { isApiError } from "@/lib/api/errors";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { notify } from "@/lib/store/toast";

declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: {
      /** The caller shows this mutation's errors itself (e.g. inline form errors), so skip the toast. */
      handlesError?: boolean;
    };
  }
}

/**
 * What to actually say when a write fails.
 *
 * A validation error's own message is "Some fields are invalid." — true, and
 * no use to anybody. The sentence worth reading is in `fields`: the API
 * already writes one per field, in words a person can act on ("This is before
 * it starts."). Showing the wrapper instead of the contents made every
 * validation failure in the app read the same and explain nothing.
 *
 * Several fields are joined rather than one picked, because a form that is
 * wrong twice should say so once.
 */
function readableMessage(error: unknown): string {
  if (!isApiError(error)) return GENERIC_ERROR_MESSAGE;
  const fields = Object.values(error.fields ?? {}).filter(Boolean);
  return fields.length > 0 ? fields.join(" ") : error.message;
}

/**
 * The app's QueryClient. Two policies live here, once, instead of in every
 * page:
 *
 * - Every failed mutation tells the user, via a toast with the API's
 *   user-safe message. A page that shows errors inline opts out with
 *   `meta: { handlesError: true }`. A 401 is skipped: the app shell is
 *   already sending the user to /login.
 * - Offline: queries serve what's cached (offlineFirst). Mutations use the
 *   default "online" mode, so a change made offline is *paused*, not failed:
 *   its optimistic update shows immediately and React Query sends it when
 *   the connection returns. Paused changes survive the app being closed
 *   (lib/query/persist.ts). A write marked `onlineOnly` opts out and fails
 *   instead (lib/query/mutations.ts).
 */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    mutationCache: new MutationCache({
      onError: (error, _variables, _context, mutation) => {
        if (mutation.meta?.handlesError) return;
        if (isApiError(error) && error.status === 401) return;
        notify(readableMessage(error));
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: true,
        networkMode: "offlineFirst",
        // Retry server hiccups, never a 4xx: asking again won't change the answer.
        retry: (count, error) => !(isApiError(error) && error.status < 500) && count < 2,
      },
    },
  });
}
