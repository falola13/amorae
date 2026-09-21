import type { Metadata, Viewport } from "next";
import { Inter, Manrope } from "next/font/google";
import type { ReactNode } from "react";

import { ServiceWorkerRegistrar } from "@/components/pwa/service-worker-registrar";
import { brand } from "@/lib/brand";
import { startupImages } from "@/lib/pwa/startup-images";

import "./globals.css";

// Manrope for headings: the closest open font to the outlined wordmark.
const manrope = Manrope({
  subsets: ["latin"],
  variable: "--font-manrope",
  display: "swap",
});

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

// The PWA <head>. The manifest link comes from app/manifest.ts automatically.
export const metadata: Metadata = {
  applicationName: brand.name,
  title: {
    template: `%s · ${brand.name}`,
    default: brand.name,
  },
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
  // Next emits the standard mobile-web-app-capable tag for `capable`; iOS
  // still reads the apple- prefixed one before it will use startup images.
  other: { "apple-mobile-web-app-capable": "yes" },
  formatDetection: { telephone: false },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  // Lets the app draw edge to edge on notched phones; globals.css pads the
  // body by the safe-area insets so nothing sits under the notch or home bar.
  viewportFit: "cover",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: brand.colors.background },
    { media: "(prefers-color-scheme: dark)", color: brand.colors.darkBackground },
  ],
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={`${manrope.variable} ${inter.variable}`}>
      <body className="flex min-h-dvh flex-col bg-bg font-sans text-fg antialiased">
        {children}
        <ServiceWorkerRegistrar />
      </body>
    </html>
  );
}
