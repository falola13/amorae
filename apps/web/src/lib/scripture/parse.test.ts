import { describe, expect, it } from "vitest";

import { parseReference } from "./parse";

describe("parseReference", () => {
  it("parses a plain book, chapter and verse", () => {
    expect(parseReference("Genesis 1:5")).toMatchObject({
      book: "Genesis",
      bookNumber: 1,
      chapter: 1,
      verseStart: 5,
      canonical: "Genesis 1:5",
    });
  });

  it("is case-insensitive and tolerates a bare space instead of a colon", () => {
    expect(parseReference("genesis 1 5")).toMatchObject({
      book: "Genesis",
      chapter: 1,
      verseStart: 5,
      canonical: "Genesis 1:5",
    });
  });

  it("accepts an abbreviation with a trailing period and a period separator", () => {
    expect(parseReference("Gen. 1.5")).toMatchObject({
      book: "Genesis",
      chapter: 1,
      verseStart: 5,
      canonical: "Genesis 1:5",
    });
  });

  it("parses a numbered book with a digit prefix and an en dash range", () => {
    expect(parseReference("1 cor 13:4–7")).toMatchObject({
      book: "1 Corinthians",
      bookNumber: 46,
      chapter: 13,
      verseStart: 4,
      verseEnd: 7,
      canonical: "1 Corinthians 13:4-7",
    });
  });

  it("accepts a roman-numeral prefix", () => {
    expect(parseReference("I John 4:8")).toMatchObject({
      book: "1 John",
      bookNumber: 62,
      chapter: 4,
      verseStart: 8,
      canonical: "1 John 4:8",
    });
  });

  it("accepts an ordinal-word prefix", () => {
    expect(parseReference("First Corinthians 13:4")).toMatchObject({
      book: "1 Corinthians",
      canonical: "1 Corinthians 13:4",
    });
  });

  it("accepts a glued digit-abbreviation form", () => {
    expect(parseReference("1Co 13:4")).toMatchObject({
      book: "1 Corinthians",
      canonical: "1 Corinthians 13:4",
    });
  });

  it("distinguishes the Gospel of John from the epistles", () => {
    expect(parseReference("John 3:16")).toMatchObject({ book: "John", bookNumber: 43 });
    expect(parseReference("1 John 4:8")).toMatchObject({ book: "1 John", bookNumber: 62 });
    expect(parseReference("2 Jn 1:1")).toMatchObject({ book: "2 John", bookNumber: 63 });
    expect(parseReference("3 John 1:2")).toMatchObject({ book: "3 John", bookNumber: 64 });
  });

  it("handles a multi-word book name, full or abbreviated", () => {
    expect(parseReference("Song of Solomon 2:4")).toMatchObject({
      book: "Song of Solomon",
      bookNumber: 22,
      canonical: "Song of Solomon 2:4",
    });
    expect(parseReference("Song of Songs 2:4")).toMatchObject({ book: "Song of Solomon" });
    expect(parseReference("SOS 2:4")).toMatchObject({ book: "Song of Solomon" });
  });

  it("parses a chapter-only reference (whole chapter)", () => {
    expect(parseReference("Ps 23")).toMatchObject({
      book: "Psalms",
      bookNumber: 19,
      chapter: 23,
      verseStart: undefined,
      canonical: "Psalms 23",
    });
    expect(parseReference("Psalm 23")).toMatchObject({ book: "Psalms", canonical: "Psalms 23" });
    expect(parseReference("Psalms 23")).toMatchObject({ book: "Psalms", canonical: "Psalms 23" });
  });

  it("accepts v/vv as a separator", () => {
    expect(parseReference("Genesis 1v5")).toMatchObject({ chapter: 1, verseStart: 5 });
    expect(parseReference("Ps 23 vv 1-3")).toMatchObject({
      book: "Psalms",
      chapter: 23,
      verseStart: 1,
      verseEnd: 3,
    });
  });

  it("tolerates extra spaces, smart punctuation and non-breaking spaces", () => {
    expect(parseReference("  Genesis   1:5  ")).toMatchObject({ chapter: 1, verseStart: 5 });
    expect(parseReference("Genesis 1:5")).toMatchObject({ chapter: 1, verseStart: 5 });
    expect(parseReference("1 Corinthians 13:4–7")).toMatchObject({
      book: "1 Corinthians",
      verseStart: 4,
      verseEnd: 7,
    });
  });

  it("does not confuse Isaiah with the roman numeral I", () => {
    expect(parseReference("Isaiah 1:5")).toMatchObject({ book: "Isaiah", bookNumber: 23 });
  });

  it("does not confuse Jonah with the abbreviation Jon", () => {
    expect(parseReference("Jonah 1:17")).toMatchObject({ book: "Jonah", bookNumber: 32 });
    expect(parseReference("Jon 1:17")).toMatchObject({ book: "Jonah", bookNumber: 32 });
  });

  it("returns null for junk", () => {
    expect(parseReference("")).toBeNull();
    expect(parseReference("   ")).toBeNull();
    expect(parseReference("hello world")).toBeNull();
    expect(parseReference("42")).toBeNull();
    expect(parseReference("Genesis")).toBeNull();
    expect(parseReference("Genesis 1:5 extra")).toBeNull();
    expect(parseReference("Notabook 1:1")).toBeNull();
    expect(parseReference("Genesis 0:5")).toBeNull();
    expect(parseReference("Genesis 1:5-3")).toBeNull();
  });
});
