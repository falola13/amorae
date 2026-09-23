// "Now", in one place, so every screen agrees on "tonight" and "this week",
// and a test can pin it. Always a fresh Date, so a caller mutating it can't
// shift everyone's "today".
export const today = (): Date => new Date();
