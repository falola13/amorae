// The mock's "now", split from the mock store so lib/today.ts can read it
// without pulling the store and its seed data into every page bundle.

/** Tuesday 29 September 2026, evening: anchored so the demo data is coherent. */
export const MOCK_TODAY = new Date(2026, 8, 29, 19, 10);
