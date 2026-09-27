// A pure parser for Bible references, typed the way people actually type them: full names,
// common abbreviations, roman or arabic numerals for the numbered books, and separators from
// ":" to a bare space to "v"/"vv" — plus whatever smart punctuation a phone keyboard hands back.
// No book knowledge lives anywhere else: the API route and the client preview both call this.

export interface ParsedReference {
  /** Canonical book name, numeral included for numbered books ("1 Corinthians"). */
  book: string;
  /** 1 (Genesis) through 66 (Revelation), the order every source agrees on. */
  bookNumber: number;
  chapter: number;
  verseStart?: number;
  verseEnd?: number;
  /** "Genesis 1:5", "1 Corinthians 13:4-7", "Psalms 23" — what gets looked up and stored. */
  canonical: string;
}

type BookDef = {
  number: number;
  name: string;
  /** Set only for a book that needs a leading 1/2/3 — its aliases never include the numeral. */
  num?: 1 | 2 | 3;
  aliases: string[];
};

// Full name always included as an alias so a typo-free name never depends on the abbreviation list.
const book = (number: number, name: string, aliases: string[], num?: 1 | 2 | 3): BookDef => ({
  number,
  name,
  num,
  aliases: [name, ...aliases],
});

const BOOKS: BookDef[] = [
  book(1, "Genesis", ["gen", "ge"]),
  book(2, "Exodus", ["exod", "exo", "ex"]),
  book(3, "Leviticus", ["lev", "le"]),
  book(4, "Numbers", ["num", "nu", "nm"]),
  book(5, "Deuteronomy", ["deut", "dt", "de"]),
  book(6, "Joshua", ["josh", "jos"]),
  book(7, "Judges", ["judg", "jdg", "jg"]),
  book(8, "Ruth", ["ru"]),
  book(9, "Samuel", ["sam", "sa"], 1),
  book(10, "Samuel", ["sam", "sa"], 2),
  book(11, "Kings", ["kgs", "ki"], 1),
  book(12, "Kings", ["kgs", "ki"], 2),
  book(13, "Chronicles", ["chron", "chr", "ch"], 1),
  book(14, "Chronicles", ["chron", "chr", "ch"], 2),
  book(15, "Ezra", ["ezr"]),
  book(16, "Nehemiah", ["neh"]),
  book(17, "Esther", ["esth", "est"]),
  book(18, "Job", ["jb"]),
  book(19, "Psalms", ["psalm", "psa", "ps", "pss"]),
  book(20, "Proverbs", ["prov", "prv", "pr"]),
  book(21, "Ecclesiastes", ["eccles", "eccl", "ecc"]),
  book(22, "Song of Solomon", ["song of songs", "song", "sos"]),
  book(23, "Isaiah", ["isa", "is"]),
  book(24, "Jeremiah", ["jer", "je"]),
  book(25, "Lamentations", ["lam", "la"]),
  book(26, "Ezekiel", ["ezek", "eze", "ezk"]),
  book(27, "Daniel", ["dan", "da", "dn"]),
  book(28, "Hosea", ["hos", "ho"]),
  book(29, "Joel", ["joe", "jl"]),
  book(30, "Amos", ["am"]),
  book(31, "Obadiah", ["obad", "ob"]),
  book(32, "Jonah", ["jnh", "jon"]),
  book(33, "Micah", ["mic", "mc"]),
  book(34, "Nahum", ["nah", "na"]),
  book(35, "Habakkuk", ["hab"]),
  book(36, "Zephaniah", ["zeph", "zep", "zp"]),
  book(37, "Haggai", ["hag", "hg"]),
  book(38, "Zechariah", ["zech", "zec", "zc"]),
  book(39, "Malachi", ["mal", "ml"]),
  book(40, "Matthew", ["matt", "mt"]),
  book(41, "Mark", ["mrk", "mk"]),
  book(42, "Luke", ["luk", "lk"]),
  book(43, "John", ["jhn", "jn"]),
  book(44, "Acts", ["act", "ac"]),
  book(45, "Romans", ["rom", "ro", "rm"]),
  book(46, "Corinthians", ["cor", "co"], 1),
  book(47, "Corinthians", ["cor", "co"], 2),
  book(48, "Galatians", ["gal", "ga"]),
  book(49, "Ephesians", ["eph"]),
  book(50, "Philippians", ["phil", "php"]),
  book(51, "Colossians", ["col"]),
  book(52, "Thessalonians", ["thess", "thes", "th"], 1),
  book(53, "Thessalonians", ["thess", "thes", "th"], 2),
  book(54, "Timothy", ["tim", "ti"], 1),
  book(55, "Timothy", ["tim", "ti"], 2),
  book(56, "Titus", ["tit"]),
  book(57, "Philemon", ["philem", "phlm", "phm"]),
  book(58, "Hebrews", ["heb"]),
  book(59, "James", ["jas", "jm"]),
  book(60, "Peter", ["pet", "pt", "pe"], 1),
  book(61, "Peter", ["pet", "pt", "pe"], 2),
  book(62, "John", ["jhn", "jn"], 1),
  book(63, "John", ["jhn", "jn"], 2),
  book(64, "John", ["jhn", "jn"], 3),
  book(65, "Jude", ["jud"]),
  book(66, "Revelation", ["revelations", "rev", "re"]),
];

// Every (book, alias) pair, longest alias first so "song of solomon" is tried before "song",
// and "psalms" before "ps". Aliases are compared without periods and case-insensitively.
const PAIRS: { entry: BookDef; alias: string }[] = BOOKS.flatMap((entry) =>
  entry.aliases.map((alias) => ({ entry, alias: alias.toLowerCase() })),
).sort((a, b) => b.alias.length - a.alias.length);

const WORD_PREFIX = /^(first|second|third|1st|2nd|3rd|iii|ii|i)\b\.?\s*/i;
const DIGIT_PREFIX = /^([123])\.?\s*/;
const WORD_PREFIX_NUM: Record<string, 1 | 2 | 3> = {
  first: 1,
  "1st": 1,
  i: 1,
  second: 2,
  "2nd": 2,
  ii: 2,
  third: 3,
  "3rd": 3,
  iii: 3,
};

/** Smart punctuation and phone-keyboard spacing, flattened to plain ASCII the parser can match. */
const normalize = (input: string): string =>
  input
    .normalize("NFKC")
    .replace(/[‘’‛]/g, "'")
    .replace(/[“”]/g, '"')
    .replace(/[–—]/g, "-") // en/em dash -> hyphen
    .replace(/[   ]/g, " ") // non-breaking spaces -> plain space
    .replace(/\s+/g, " ")
    .trim()
    .replace(/[.\s]+$/, "");

const stripNumeralPrefix = (s: string): { rest: string; num?: 1 | 2 | 3 } => {
  const word = WORD_PREFIX.exec(s);
  if (word) {
    return { rest: s.slice(word[0].length), num: WORD_PREFIX_NUM[word[1].toLowerCase()] };
  }
  const digit = DIGIT_PREFIX.exec(s);
  if (digit) return { rest: s.slice(digit[0].length), num: Number(digit[1]) as 1 | 2 | 3 };
  return { rest: s };
};

/** Longest alias at the start of `rest` whose book matches `num` (undefined for a plain book). */
const matchBook = (
  rest: string,
  num: 1 | 2 | 3 | undefined,
): { entry: BookDef; len: number } | null => {
  const lower = rest.toLowerCase();
  for (const { entry, alias } of PAIRS) {
    if (entry.num !== num) continue;
    if (!lower.startsWith(alias)) continue;
    let len = alias.length;
    if (lower[len] === ".") len += 1;
    const next = lower[len];
    if (next && /[a-z]/.test(next)) continue; // e.g. "job" inside a longer word
    return { entry, len };
  }
  return null;
};

// Chapter, then an optional verse and range, separated by any mix of space/./:/v/vv:
// "1:5", "1.5", "1 5", "1v5", "1 vv 4-7". The dash itself is normalized to "-" up front.
const LOCATION = /^(\d{1,3})(?:[\s.:vV]+(\d{1,3})(?:\s*-\s*(\d{1,3}))?)?$/;

export function parseReference(input: string): ParsedReference | null {
  if (!input) return null;
  const normalized = normalize(input);
  if (!normalized) return null;

  const { rest: afterPrefix, num } = stripNumeralPrefix(normalized);
  const hit = matchBook(afterPrefix, num);
  if (!hit) return null;

  const location = afterPrefix.slice(hit.len).trim();
  const m = LOCATION.exec(location);
  if (!m) return null;

  const chapter = Number(m[1]);
  const verseStart = m[2] ? Number(m[2]) : undefined;
  const verseEnd = m[3] ? Number(m[3]) : undefined;
  if (chapter < 1 || chapter > 176) return null;
  if (verseStart !== undefined && verseStart < 1) return null;
  if (verseEnd !== undefined && verseEnd < verseStart!) return null;

  const book = hit.entry.num ? `${hit.entry.num} ${hit.entry.name}` : hit.entry.name;
  const canonical =
    verseStart === undefined
      ? `${book} ${chapter}`
      : verseEnd === undefined || verseEnd === verseStart
        ? `${book} ${chapter}:${verseStart}`
        : `${book} ${chapter}:${verseStart}-${verseEnd}`;

  return { book, bookNumber: hit.entry.number, chapter, verseStart, verseEnd, canonical };
}
