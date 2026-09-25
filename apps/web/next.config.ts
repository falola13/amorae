import type { NextConfig } from "next";

// API_URL and COOKIE_SECURE are read from process.env at request time inside
// src/lib/env.ts (a server-only module), never routed through this file's
// `env` key — that key inlines values into the build, which would bake a
// build-time API_URL into the bundle instead of letting each deployed
// container point at its own API via a runtime env var.
const securityHeaders = [
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
];

// The one value that *should* be baked in: which deploy this bundle came from.
// An open app compares it with /api/version to learn a newer deploy exists —
// sw.js only changes when its VERSION is bumped by hand, so a new worker is
// not a signal that the app changed.
const buildId = process.env.VERCEL_GIT_COMMIT_SHA ?? `local-${Date.now()}`;

const nextConfig: NextConfig = {
  env: { NEXT_PUBLIC_BUILD_ID: buildId },
  // Required by the Dockerfile: produces .next/standalone, a minimal
  // server.js plus only the node_modules files each page actually needs.
  output: "standalone",
  poweredByHeader: false,
  async headers() {
    return [
      { source: "/:path*", headers: securityHeaders },
      {
        // The service worker must never be served stale from an HTTP cache,
        // or users stay on an old worker. It may only load same-origin code.
        source: "/sw.js",
        headers: [
          { key: "Content-Type", value: "application/javascript; charset=utf-8" },
          { key: "Cache-Control", value: "no-cache, no-store, must-revalidate" },
          { key: "Content-Security-Policy", value: "default-src 'self'; script-src 'self'" },
        ],
      },
    ];
  },
};

export default nextConfig;
