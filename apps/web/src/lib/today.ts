// One "now" so every screen agrees and a test can pin it. Always a fresh Date
// so a caller mutating it can't shift everyone else's "today".
export const today = (): Date => new Date();
