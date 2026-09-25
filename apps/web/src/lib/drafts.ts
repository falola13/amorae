// Unsent writing kept on this device, so a phone locking or the OS
// reclaiming a backgrounded PWA doesn't take a half-written prayer with it.
// Storage can be missing or refuse (private mode, full), so every call is
// best-effort: a draft that can't be kept is the old behaviour, not an error.

const PREFIX = "amorae:draft:";

export function readDraft<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(PREFIX + key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
}

export function writeDraft(key: string, value: unknown): void {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value));
  } catch {
    // Best-effort; see above.
  }
}

export function clearDraft(key: string): void {
  try {
    localStorage.removeItem(PREFIX + key);
  } catch {
    // Best-effort; see above.
  }
}
