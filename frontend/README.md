# Amorae, frontend

A private space for two people. Next.js 15 + TypeScript + Tailwind + TanStack Query, built as an iPhone-first installable PWA from the Amorae design canvas.

## Run it

```
npm install
npm run dev        # http://localhost:3000
npm run build && npm start
```

Log in with any email and a password of 10+ characters. Data is mocked in `src/lib/api.ts` and persisted in localStorage, so the app works end to end (and offline) before the Go API exists.

## Where things are

- `src/app/(onboarding)/` welcome, sign up, log in, create couple, join, invite, install, notifications
- `src/app/(app)/` signed-in shell with the tab bar: Home, Together, Prayers, History, Settings
- `src/components/ui.tsx` the design system as components (buttons, fields, rows, switch, sheet, toast, empty and error states)
- `src/components/Icon.tsx` the outline icon set and the Amorae mark
- `src/lib/api.ts` mock API with the same surface as the Go endpoints, plus the offline queue
- `src/lib/hooks.ts` TanStack Query hooks, network state, install and push helpers
- `tailwind.config.ts` colour, type and motion tokens, one to one with the design system board
- `public/sw.js` service worker: app shell offline, cache-first assets, push handlers
- `public/manifest.webmanifest`, `public/icons`, `public/splash` the PWA icon and launch-screen pack

## Wiring the Go API

Replace the function bodies in `src/lib/api.ts` with `fetch` calls (`credentials: 'include'` for the HTTP-only cookie). The hooks and screens do not change. Set `NEXT_PUBLIC_API_URL` and `NEXT_PUBLIC_VAPID_PUBLIC_KEY` in `.env.local`, then finish `usePush()` in `hooks.ts` by subscribing with the VAPID key and POSTing to `/notifications/subscribe`.

Settings has a "Try the week states" switch so you can preview the three Home states (praying, your week, waiting). Remove it once the scheduler drives the week.

## PWA notes

- Install first, then ask for notifications: iOS only allows Web Push from an installed app.
- Safe areas come from `viewport-fit=cover` plus the `--safe-top` / `--safe-bottom` variables in `globals.css`.
- The tab bar hides while the keyboard is open; compose screens keep Cancel and Save in the top bar.
- Inputs are never below 16px, so Safari does not zoom on focus.
- `prefers-reduced-motion` turns every animation into an instant state change.
