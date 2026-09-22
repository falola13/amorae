import { MOCK_TODAY } from "@/lib/api/mock/clock";
import { isMockApi } from "@/lib/config";

// In mock mode the demo data is anchored to one evening so every screen
// agrees on "tonight" and "this week". With the real API this is the clock.
// Always a fresh Date, so a caller mutating it can't shift everyone's "today".
export const today = (): Date => (isMockApi ? new Date(MOCK_TODAY) : new Date());
