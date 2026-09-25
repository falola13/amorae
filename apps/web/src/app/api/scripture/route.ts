import { NextResponse, type NextRequest } from "next/server";

import { errorBody } from "@/lib/api/envelope";

// Scripture lookup, proxied rather than called from the browser so bible-api.com
// never learns whose IP is praying about what. WEB is public domain, so the text
// can be stored on a prayer point without a licence.

// Cached hard: the text of Romans 8:28 is not going to change.
export const revalidate = 604800;

const SOURCE = "https://bible-api.com";
const TIMEOUT_MS = 6_000;
// The scripture field's own cap (schemas.ts), so nothing longer can be typed anyway.
const MAX_REF = 60;
// Books, numbers, and the punctuation a reference is made of. Not a parser —
// bible-api decides what resolves; this only keeps junk out of the URL.
const REFERENCE = /^[\p{L}\d][\p{L}\d .,:;–—-]*$/u;

type Upstream = {
  reference?: string;
  text?: string;
  translation_name?: string;
  error?: string;
};

export async function GET(request: NextRequest) {
  const ref = (request.nextUrl.searchParams.get("ref") ?? "").trim();
  if (!ref || ref.length > MAX_REF || !REFERENCE.test(ref)) {
    return NextResponse.json(
      errorBody("invalid_reference", "That doesn't look like a book, chapter and verse."),
      { status: 400 },
    );
  }

  const target = `${SOURCE}/${encodeURIComponent(ref)}?translation=web`;
  let res: Response;
  try {
    res = await fetch(target, {
      signal: AbortSignal.timeout(TIMEOUT_MS),
      next: { revalidate },
    });
  } catch {
    // Offline, or the source is down. Neither is the couple's problem: the
    // reference they typed is still theirs to keep, so this never fails a save.
    return NextResponse.json(
      errorBody("lookup_unavailable", "Couldn't reach the verse text just now."),
      { status: 503 },
    );
  }

  if (res.status === 404) {
    return NextResponse.json(errorBody("reference_not_found", "No verse found for that."), {
      status: 404,
    });
  }
  if (!res.ok) {
    return NextResponse.json(
      errorBody("lookup_unavailable", "Couldn't reach the verse text just now."),
      { status: 503 },
    );
  }

  const body = (await res.json().catch(() => ({}))) as Upstream;
  if (!body.text || body.error) {
    return NextResponse.json(errorBody("reference_not_found", "No verse found for that."), {
      status: 404,
    });
  }

  return NextResponse.json({
    data: {
      // The tidied reference, so "romans 8 28" comes back "Romans 8:28".
      reference: body.reference ?? ref,
      // Verses arrive with their line breaks; a prayer point wants a sentence.
      text: body.text.replace(/\s+/g, " ").trim(),
      translation: body.translation_name ?? "World English Bible",
    },
  });
}
