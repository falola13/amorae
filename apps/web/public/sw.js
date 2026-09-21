// Amorae service worker.
//
// The one rule: never cache anything personal. Every page is rendered for
// the signed-in user behind an httpOnly session cookie, so a cached page
// could show one person's data to whoever opens the app next on a shared
// device, even after logout. What is safe to cache is only what is
// identical for every visitor:
//
//   /_next/static/*        content-hashed JS/CSS/fonts  → cache first (immutable)
//   /icons /splash /brand  brand assets, the manifest   → stale-while-revalidate
//   page navigations       personalised HTML            → network only; if the
//                                                          network is down, show /offline
//   everything else        POSTs (Server Actions), RSC  → not intercepted at all
//                          payloads, cross-origin
//
// Bump VERSION whenever this file or the /offline page changes. The browser
// only installs a new worker when these bytes change, and installing is what
// refreshes the precached offline page and clears old caches.

const VERSION = "v1";
const PRECACHE = `amorae-precache-${VERSION}`;
const STATIC_CACHE = `amorae-static-${VERSION}`;
const ASSET_CACHE = `amorae-assets-${VERSION}`;
const OFFLINE_URL = "/offline";

// Hashed chunks from old deploys are never requested again, so the static
// cache is trimmed to a ceiling instead of growing forever.
const MAX_STATIC_ENTRIES = 200;

const BRAND_ASSET = /^\/(icons|splash|brand)\//;

self.addEventListener("install", (event) => {
  event.waitUntil(precacheOfflinePage());
  // Safe to take over straight away: pages are never served from cache, and
  // static files are content-hashed, so old and new deploys can't mix.
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keep = new Set([PRECACHE, STATIC_CACHE, ASSET_CACHE]);
      const names = await caches.keys();
      await Promise.all(names.filter((n) => n.startsWith("amorae-") && !keep.has(n)).map((n) => caches.delete(n)));
      // Navigation preload starts the page request in parallel with booting
      // this worker, so the worker adds no latency to navigations.
      if (self.registration.navigationPreload) {
        await self.registration.navigationPreload.enable();
      }
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  if (request.mode === "navigate") {
    event.respondWith(navigate(event));
    return;
  }
  if (url.pathname.startsWith("/_next/static/")) {
    event.respondWith(cacheFirst(request));
    return;
  }
  if (BRAND_ASSET.test(url.pathname) || url.pathname === "/manifest.webmanifest") {
    event.respondWith(staleWhileRevalidate(event, request));
  }
  // Anything else falls through to the network untouched.
});

// The offline page plus every static file it needs (CSS, fonts, JS), so it
// renders fully styled with no network at all.
async function precacheOfflinePage() {
  const cache = await caches.open(PRECACHE);
  const response = await fetch(OFFLINE_URL, { cache: "reload" });
  if (!response.ok) throw new Error(`precache ${OFFLINE_URL}: HTTP ${response.status}`);

  // Asset paths also appear JSON-escaped inside Next's inline flight data
  // (\"/_next/static/...\"), hence stopping at backslashes and HTML entities.
  const html = await response.clone().text();
  const assets = new Set(html.match(/\/_next\/static\/[^"'\s)\\<>&]+/g) ?? []);
  assets.add("/icons/favicon.svg");

  await cache.put(OFFLINE_URL, response);
  // One by one, not addAll: a single missing asset should cost the offline
  // page a font, not fail the whole install and leave no worker at all.
  await Promise.allSettled([...assets].map((asset) => cache.add(asset)));
}

async function navigate(event) {
  try {
    const preloaded = await event.preloadResponse;
    if (preloaded) return preloaded;
    return await fetch(event.request);
  } catch {
    // Only reached when the network itself failed. A 500 from the server
    // is a real response and is shown as-is.
    const offline = await caches.match(OFFLINE_URL, { cacheName: PRECACHE });
    return offline ?? Response.error();
  }
}

async function cacheFirst(request) {
  const cached = await caches.match(request, { cacheName: STATIC_CACHE });
  if (cached) return cached;

  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(STATIC_CACHE);
    await cache.put(request, response.clone());
    await trim(cache, MAX_STATIC_ENTRIES);
  }
  return response;
}

async function staleWhileRevalidate(event, request) {
  const cache = await caches.open(ASSET_CACHE);
  const cached = await cache.match(request);

  const refresh = fetch(request)
    .then(async (response) => {
      if (response.ok) await cache.put(request, response.clone());
      return response;
    })
    .catch(() => undefined);

  if (cached) {
    // Serve the cached copy now, and let the refresh finish in the background.
    event.waitUntil(refresh);
    return cached;
  }
  return (await refresh) ?? Response.error();
}

// Cache keys come back in insertion order, so the oldest go first.
async function trim(cache, max) {
  const keys = await cache.keys();
  await Promise.all(keys.slice(0, Math.max(0, keys.length - max)).map((key) => cache.delete(key)));
}
