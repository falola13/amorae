import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
import type { ReactNode } from "react";

import { ServiceWorkerRegistrar } from "@/components/pwa/service-worker-registrar";
import { Providers } from "@/components/providers";
import { brand } from "@/lib/brand";
import { startupImages } from "@/lib/pwa/startup-images";

import "./globals.css";

// One face across the product, per the design system: Manrope 400 to 700.
// Self-hosted (SIL OFL, src/fonts) so an installed app never waits on a
// third-party request and the offline shell still has its type.
const manrope = localFont({
  src: [{ path: "../fonts/manrope-latin-variable.woff2", weight: "200 800", style: "normal" }],
  variable: "--font-manrope",
  display: "swap",
});

export const metadata: Metadata = {
  applicationName: brand.name,
  title: { template: `%s · ${brand.name}`, default: brand.name },
  description: brand.description,
  icons: {
    icon: [
      { url: "/icons/favicon.ico", sizes: "48x48" },
      { url: "/icons/favicon.svg", type: "image/svg+xml" },
    ],
    // iOS rounds these itself, so they are square and opaque on purpose.
    apple: [
      { url: "/icons/apple-touch-icon.png", sizes: "180x180" },
      { url: "/icons/apple-touch-icon-167.png", sizes: "167x167" },
      { url: "/icons/apple-touch-icon-152.png", sizes: "152x152" },
      { url: "/icons/apple-touch-icon-120.png", sizes: "120x120" },
    ],
    other: [{ rel: "mask-icon", url: "/icons/safari-pinned-tab.svg", color: brand.colors.plum }],
  },
  appleWebApp: {
    capable: true,
    title: brand.name,
    statusBarStyle: "default",
    startupImage: startupImages,
  },
  other: { "apple-mobile-web-app-capable": "yes" },
  formatDetection: { telephone: false },
};

// Everything dynamic in this app ends up asking the Go API, which sits in
// Frankfurt with its database. Rendering in the United States and then
// fetching across the Atlantic is a second per screen that nobody chose; see
// the note in app/api/v1/[...path]/route.ts for the measurement. Static
// assets are unaffected — they are served from the edge either way.
export const preferredRegion = "fra1";

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  // One value would paint the status bar and the PWA's chrome linen whatever
  // the app underneath is doing. media queries here are how Next lets a
  // viewport answer per scheme.
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: brand.colors.background },
    { media: "(prefers-color-scheme: dark)", color: brand.colors.backgroundDark },
  ],
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={manrope.variable}>
      <body className="flex min-h-dvh flex-col bg-bg font-sans text-ink antialiased">
        <Providers>{children}</Providers>
        <ServiceWorkerRegistrar />
      </body>
    </html>
  );
}
