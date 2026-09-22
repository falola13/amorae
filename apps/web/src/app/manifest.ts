import type { MetadataRoute } from "next";

import { brand } from "@/lib/brand";

// Served at /manifest.webmanifest and linked from <head> automatically.
export default function manifest(): MetadataRoute.Manifest {
  return {
    id: "/",
    name: brand.name,
    short_name: brand.name,
    description: brand.description,
    // Launch into Home. Signed-out users are bounced to /welcome by src/proxy.ts.
    start_url: "/?source=pwa",
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
    shortcuts: [
      { name: "Open prayer", url: "/prayers/mode?source=shortcut", icons: [{ src: "/icons/shortcut-96.png", sizes: "96x96" }] },
      { name: "Add an event", url: "/together/events/new?source=shortcut", icons: [{ src: "/icons/shortcut-96.png", sizes: "96x96" }] },
    ],
  };
}
