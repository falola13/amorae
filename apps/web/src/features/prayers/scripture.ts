export type Scripture = { reference: string; text: string; translation: string };

/** What went wrong, in words the person who typed the reference can act on. */
export type LookupFailure = { kind: "not-found" | "unavailable"; message: string };

export type LookupResult =
  | { ok: true; scripture: Scripture }
  | { ok: false; failure: LookupFailure };

// Looks a reference up through our own route (app/api/scripture), never the
// source directly. Never throws: a verse that will not load must not be able
// to stop a prayer being saved.
export async function lookupScripture(ref: string): Promise<LookupResult> {
  try {
    const res = await fetch(`/api/scripture?ref=${encodeURIComponent(ref)}`);
    const body = await res.json().catch(() => null);
    if (res.ok && body?.data?.text) return { ok: true, scripture: body.data as Scripture };
    return {
      ok: false,
      failure: {
        kind: res.status === 404 || res.status === 400 ? "not-found" : "unavailable",
        message: body?.error?.message ?? "Couldn't look that up just now.",
      },
    };
  } catch {
    return {
      ok: false,
      failure: { kind: "unavailable", message: "Couldn't look that up while you're offline." },
    };
  }
}
