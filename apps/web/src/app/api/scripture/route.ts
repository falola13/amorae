import { NextResponse, type NextRequest } from "next/server";

import { errorBody } from "@/lib/api/envelope";
import { parseReference } from "@/lib/scripture/parse";

// Scripture lookup, proxied rather than called from the browser so bible-api.com
// never learns whose IP is praying about what.

// KJV: the phrasing this couple's churches actually use ("The LORD is my
// shepherd", not WEB's "Yahweh"), and public domain, which is what makes the
// text storable on a prayer point rather than merely displayable.
const TRANSLATION = "kjv";
const TRANSLATION_NAME = "King James Version";

// Cached hard: the text of Romans 8:28 is not going to change.
export const revalidate = 604800;

const PRIMARY = "https://bible-api.com";
// Fallback when the primary is rate-limited, down, or just doesn't have a reference it should —
// keyed by book number (1-66) and chapter rather than a name, so parseReference's numbering is
// the only place either source's book identity has to agree with ours.
const FALLBACK = "https://api.getbible.net/v2/kjv";
const TIMEOUT_MS = 6_000;
// The scripture field's own cap (schemas.ts), so nothing longer can be typed anyway.
const MAX_REF = 60;
// A whole chapter this long is a lot to drop into a prayer point — ask for a verse instead.
const MAX_WHOLE_CHAPTER_VERSES = 40;

type Upstream = {
  reference?: string;
  text?: string;
  translation_name?: string;
  error?: string;
};
type GetBibleVerse = { verse: number; text: string };
type GetBibleChapter = { verses?: GetBibleVerse[] };

const clean = (text: string) => text.replace(/\s+/g, " ").trim();

const invalidReference = () =>
  NextResponse.json(
    errorBody(
      "invalid_reference",
      'That doesn’t look like a book, chapter and verse. Try something like "John 3:16" or "Psalms 23".',
    ),
    { status: 400 },
  );

const unavailable = () =>
  NextResponse.json(errorBody("lookup_unavailable", "Couldn't reach the verse text just now."), {
    status: 503,
  });

const notFound = () =>
  NextResponse.json(errorBody("reference_not_found", "No verse found for that."), { status: 404 });

/** true when the primary's failure is worth trying the fallback for, rather than giving up. */
const shouldFallBack = (status: number) => status === 404 || status === 429 || status >= 500;

async function fetchPrimary(canonical: string): Promise<Response | null> {
  const target = `${PRIMARY}/${encodeURIComponent(canonical)}?translation=${TRANSLATION}`;
  try {
    return await fetch(target, { signal: AbortSignal.timeout(TIMEOUT_MS), next: { revalidate } });
  } catch {
    // Offline, DNS failure, or the timeout firing — same as a source that's down.
    return null;
  }
}

async function fetchFallback(bookNumber: number, chapter: number): Promise<GetBibleChapter | null> {
  const target = `${FALLBACK}/${bookNumber}/${chapter}.json`;
  try {
    const res = await fetch(target, {
      signal: AbortSignal.timeout(TIMEOUT_MS),
      next: { revalidate },
    });
    if (!res.ok) return null;
    return (await res.json().catch(() => null)) as GetBibleChapter | null;
  } catch {
    return null;
  }
}

export async function GET(request: NextRequest) {
  const ref = (request.nextUrl.searchParams.get("ref") ?? "").trim();
  if (!ref || ref.length > MAX_REF) return invalidReference();

  const parsed = parseReference(ref);
  if (!parsed) return invalidReference();

  const primaryRes = await fetchPrimary(parsed.canonical);
  if (primaryRes && primaryRes.ok) {
    const body = (await primaryRes.json().catch(() => ({}))) as Upstream;
    if (body.text && !body.error) {
      return NextResponse.json({
        data: {
          // The tidied reference, so "romans 8 28" comes back "Romans 8:28".
          reference: body.reference ?? parsed.canonical,
          // Verses arrive with their line breaks; a prayer point wants a sentence.
          text: clean(body.text),
          translation: body.translation_name ?? TRANSLATION_NAME,
        },
      });
    }
    // 200 with no usable text reads the same as "not found" for fallback purposes.
  } else if (primaryRes && !shouldFallBack(primaryRes.status)) {
    return unavailable();
  }
  // primaryRes === null (network/timeout) or a status worth retrying: fall through.

  const chapter = await fetchFallback(parsed.bookNumber, parsed.chapter);
  if (!chapter?.verses?.length) return primaryRes ? notFound() : unavailable();

  let verses = chapter.verses;
  if (parsed.verseStart !== undefined) {
    const end = parsed.verseEnd ?? parsed.verseStart;
    verses = verses.filter((v) => v.verse >= parsed.verseStart! && v.verse <= end);
    if (verses.length === 0) return notFound();
  } else if (verses.length > MAX_WHOLE_CHAPTER_VERSES) {
    return NextResponse.json(
      errorBody(
        "chapter_too_long",
        `That chapter has ${verses.length} verses — add one, like "${parsed.book} ${parsed.chapter}:1".`,
      ),
      { status: 400 },
    );
  }

  return NextResponse.json({
    data: {
      reference: parsed.canonical,
      text: clean(verses.map((v) => v.text).join(" ")),
      translation: TRANSLATION_NAME,
    },
  });
}
