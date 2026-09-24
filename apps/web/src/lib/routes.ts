// Every in-app URL, built in one place. Pages link with `routes.goal(id)`
// instead of hand-writing "/together/goals/" + id, so renaming a route or a
// query parameter is one edit, and the compiler finds every caller.

const q = (params: Record<string, string | number | boolean | undefined>) => {
  const search = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== false) search.set(k, v === true ? "1" : String(v));
  }
  const s = search.toString();
  return s ? `?${s}` : "";
};
const seg = encodeURIComponent;

export const routes = {
  home: "/",

  // Signed out
  welcome: "/welcome",
  login: (opts: { next?: string; expired?: boolean } = {}) =>
    `/login${q({ next: opts.next, reason: opts.expired ? "expired" : undefined })}`,
  register: "/register",
  forgot: "/forgot",
  offline: "/offline",
  terms: "/terms",
  privacy: "/privacy",

  // Onboarding
  couple: (opts: { fresh?: boolean } = {}) => `/couple${q({ fresh: opts.fresh })}`,
  invite: "/invite",
  join: (opts: { code?: string } = {}) => `/join${q({ code: opts.code })}`,
  install: "/install",
  notificationsSetup: "/notifications",

  // Prayers
  prayers: "/prayers",
  prayer: (id: string) => `/prayers/${seg(id)}`,
  prayersSet: "/prayers/set",
  prayersDone: "/prayers/done",
  prayerEdit: (id?: string) => `/prayers/edit${q({ id })}`,
  prayerMode: (opts: { at?: number; quiet?: boolean } = {}) =>
    `/prayers/mode${q({ at: opts.at, quiet: opts.quiet })}`,
  history: "/history",
  historyWeek: (id: string) => `/history/${seg(id)}`,

  // Together
  together: "/together",
  calendar: "/together/calendar",
  events: "/together/events",
  event: (id: string) => `/together/events/${seg(id)}`,
  // `on` prefills the date. Tapping a day in the calendar and then adding
  // something should put it on that day, not on today.
  eventNew: (opts: { edit?: string; on?: string } = {}) =>
    `/together/events/new${q({ edit: opts.edit, on: opts.on })}`,
  goals: "/together/goals",
  goal: (id: string) => `/together/goals/${seg(id)}`,
  goalNew: "/together/goals/new",
  journal: "/together/journal",
  appreciation: "/together/appreciation",
  memories: "/together/memories",
  milestones: "/together/milestones",
  challenges: "/together/challenges",

  // Settings
  settings: "/settings",
  settingsProfile: "/settings/profile",
  settingsCouple: "/settings/couple",
  settingsPastSpace: "/settings/past-space",
  settingsNotifications: "/settings/notifications",
  settingsDevices: "/settings/devices",
} as const;
