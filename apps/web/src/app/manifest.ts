import type { MetadataRoute } from "next";

import { brand } from "@/lib/brand";

// Served at /manifest.webmanifest, and Next links it from <head>
// automatically. Written in TypeScript rather than as a static file so the
// colours come from the same constants as the theme-color meta tags.
export default function manifest(): MetadataRoute.Manifest {
  return {
    id: "/",
    name: brand.name,
    short_name: brand.name,
    description: brand.description,
    // Launch into the app, not the marketing page. Signed-out users are
    // bounced to /login by src/proxy.ts, so this is safe for both.
    start_url: "/dashboard?source=pwa",
    scope: "/",
    display: "standalone",
    orientation: "portrait",
    background_color: brand.colors.background,
    theme_color: brand.colors.background,
    lang: "en",
    dir: "ltr",
    categories: ["lifestyle"],
    icons: [
      { src: "/icons/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any" },
      { src: "/icons/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any" },
      { src: "/icons/icon-maskable-192.png", sizes: "192x192", type: "image/png", purpose: "maskable" },
      { src: "/icons/icon-maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
      { src: "/icons/badge-96.png", sizes: "96x96", type: "image/png", purpose: "monochrome" },
    ],
    // Shortcuts ("Open prayer" → /prayers, "Add an event" →
    // /together/events/new, icon /icons/shortcut-96.png) are designed but
    // left out until those routes exist: an installed app whose shortcuts
    // open a 404 is worse than one with no shortcuts.
  };
}
