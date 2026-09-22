# Amorae brand

*Two hearts, one faith.* A private space for the life you are building together.

## Colours

| Token | Hex | Used for |
| --- | --- | --- |
| Plum | `#5B2A4A` | The mark, primary actions, links (light mode) |
| Background | `#F7F4EF` | Page background, PWA splash and theme colour |
| Ink | `#24201F` | Text, and dark-mode surfaces |
| Soft white | `#FFFDFA` | Cards, and text on plum |
| Plum tint (dark mode) | `#E9D5E1` | The mark and accent on dark backgrounds |

Code reads these from two places. Keep them in step:

- `apps/web/src/app/globals.css`: design tokens for the UI (`--color-*`).
- `apps/web/src/lib/brand.ts`: the manifest, theme-color meta tags and the
  Safari pinned-tab colour, which can't read CSS.

Type: everything is set in Manrope, the closest open font to the wordmark,
self-hosted from `apps/web/src/fonts`. The wordmark itself is outlined in the
SVGs, so it needs no font.

## The mark

- Clear space: half the mark's width on every side. The lockup SVGs already
  include it, so don't add extra margin around them.
- Minimum size is 16px. Below 32px use the heavier drawing, which
  `<Mark>` does automatically.
- Plum on the warm background, or soft white on plum. Use ink for
  single-colour print. No gradients, shadows or outlines.
- Never rotate it, close the gaps between the two arms, or place it inside a
  heart or ring.

## Where the assets live (`apps/web/public`)

| Path | Used by |
| --- | --- |
| `icons/icon-192.png`, `icon-512.png` | Android install, Chrome, desktop (purpose `any`) |
| `icons/icon-maskable-*.png` | Android adaptive icons (full bleed, mark inside the 80% safe zone) |
| `icons/icon-1024.png` | Master raster: store listings, social |
| `icons/apple-touch-icon*.png` | iPhone and iPad home screen (square and opaque on purpose) |
| `icons/favicon.ico`, `favicon.svg` | Browser tabs. The SVG lightens in dark mode |
| `icons/safari-pinned-tab.svg` | Safari pinned tabs |
| `icons/badge-72.png`, `badge-96.png` | Notification badge (`badge` in `showNotification`) |
| `icons/shortcut-96.png` | Manifest shortcuts (none are active yet, see `src/app/manifest.ts`) |
| `splash/*.png` | iOS launch screens, one per device. The list is in `src/lib/pwa/startup-images.ts` |
| `brand/lockup-*`, `brand/mark-*` | Logo files. `<Lockup>` and `<Mark>` in `src/components/brand/logo.tsx` |
