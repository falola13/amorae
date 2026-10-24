/**
 * The words the anniversary page brings with it: what each year is
 * traditionally marked with, questions for the evening, small plans for the
 * day, and a prayer. Written here rather than fetched — they are the page's
 * own voice, and the page should work offline on the day.
 */

import { yearsWord } from "./derive";

export interface YearSymbol {
  /** The traditional material for this year. */
  material: string;
  /** What it is said to stand for, and a small idea to act on it. */
  meaning: string;
  idea: string;
}

const SYMBOLS: Record<number, YearSymbol> = {
  1: {
    material: "Paper",
    meaning: "A blank page: everything still to be written, and the first lines already there.",
    idea: "Write each other a list of ten things from the first year.",
  },
  2: {
    material: "Cotton",
    meaning: "Two threads woven until they are one cloth — comfortable, everyday, strong.",
    idea: "Something soft for the home you share: new sheets, a throw, a quilt.",
  },
  3: {
    material: "Leather",
    meaning:
      "Shelter and protection. Leather softens with use and becomes more yours the longer you keep it — like three years of being known.",
    idea: "A leather journal for the next year's prayers, or a strap with your date stamped inside.",
  },
  4: {
    material: "Fruit & flowers",
    meaning: "What is planted has begun to bear fruit.",
    idea: "Plant something together and watch it grow through year five.",
  },
  5: {
    material: "Wood",
    meaning: "Roots deep enough to weather a storm.",
    idea: "A carved keepsake box for the letters you write each other.",
  },
  6: {
    material: "Iron",
    meaning: "Strength that has been tested and held.",
    idea: "Cook a meal together in one cast-iron pan.",
  },
  7: {
    material: "Wool",
    meaning: "Warmth, and the comfort of being home.",
    idea: "Something warm to share on cold evenings.",
  },
  8: {
    material: "Bronze",
    meaning: "Two metals made into one that lasts.",
    idea: "Something to keep on a shelf for another eight years.",
  },
  9: {
    material: "Pottery",
    meaning: "Shaped by many hands, and stronger for the fire.",
    idea: "A pottery class, side by side at the wheel.",
  },
  10: {
    material: "Tin",
    meaning: "Preserving what matters, keeping it bright.",
    idea: "A time capsule to open on your twentieth.",
  },
  11: { material: "Steel", meaning: "Unbending.", idea: "Something that will outlast you both." },
  12: {
    material: "Silk",
    meaning: "Fine, and stronger than it looks.",
    idea: "Something beautiful to wear.",
  },
  13: {
    material: "Lace",
    meaning: "Intricate, delicate, made by patience.",
    idea: "Something handmade.",
  },
  14: { material: "Ivory", meaning: "Rare and treasured.", idea: "A piece of art for your home." },
  15: {
    material: "Crystal",
    meaning: "Clarity, and light passing through.",
    idea: "Two glasses to raise.",
  },
  20: {
    material: "China",
    meaning: "Fine, and set at the table every day.",
    idea: "Set the table properly tonight.",
  },
  25: {
    material: "Silver",
    meaning: "Precious, and polished by the years.",
    idea: "Renew your vows.",
  },
  30: {
    material: "Pearl",
    meaning: "Beauty formed slowly, layer by layer.",
    idea: "Somewhere by the sea.",
  },
  40: { material: "Ruby", meaning: "A love that still burns.", idea: "Dance." },
  50: {
    material: "Gold",
    meaning: "Everything you have built, gathered together.",
    idea: "Everyone you love in one room.",
  },
};

export const symbolFor = (years: number): YearSymbol | null => SYMBOLS[years] ?? null;

/** Questions for the evening — slow ones, best asked across a table. */
export const QUESTIONS = [
  "What moment this year made you feel most loved by me?",
  "When did you feel we were most on the same team?",
  "What is something I did this year that you never told me you noticed?",
  "Which prayer of ours this year are you most grateful was answered?",
  "What was the hardest week, and what got us through it?",
  "What is one thing you want us to do more of in the year ahead?",
  "What did you learn about me this year?",
  "Where do you hope we are on this day next year?",
  "What is a small, ordinary moment from this year you would live again?",
  "How can I love you better in the year ahead?",
  "What are you most proud of us for?",
  "If this year had a title, what would it be?",
  "What do you still remember about the day we started?",
  "What is one promise you want to make for the next year?",
];

export interface Idea {
  title: string;
  detail: string;
  /** A sensible time to suggest, HH:MM, when it has one. */
  time?: string;
}

/** Small plans for the day. Tapping one adds it to today's plans. */
export const IDEAS: Idea[] = [
  { title: "Breakfast in bed", detail: "Whoever wakes first.", time: "08:30" },
  {
    title: "Walk where it began",
    detail: "Go back to where you first met, or your first date.",
    time: "16:00",
  },
  { title: "Dinner, dressed up", detail: "Somewhere you have never been.", time: "19:30" },
  { title: "Read your letters aloud", detail: "To each other, slowly.", time: "21:00" },
  { title: "Pray over the year ahead", detail: "Together, with the prayer below.", time: "21:30" },
  { title: "Recreate an old photo", detail: "Same pose, years on." },
  { title: "Cook the first meal you shared", detail: "Or the closest you can manage." },
  { title: "Dance in the kitchen", detail: "The song from your first year." },
];

/** A prayer for the year ahead, in the couple's own names. */
export function anniversaryPrayer(years: number, me: string, partner: string): string[] {
  const next = years + 1;
  return [
    `Father, thank you for ${years === 1 ? "this first year" : `these ${yearsWord(years).toLowerCase()}`} — for ${me} and ${partner}, and for every day you kept us.`,
    "Thank you for the prayers you answered, and for the ones you answered differently than we asked.",
    `Bind us together like a cord of three strands as we step into year ${next}. Make our home a place of peace, our words gentle, and our love patient.`,
    "Where we have hurt each other, heal us. Where we have grown apart, draw us close. Where we have been faithful, keep us faithful.",
    "Let our love for each other point back to you. In Jesus' name,",
  ];
}
