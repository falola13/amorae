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

const VERSION = "v2";
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

// ---------------------------------------------------------------------------
// Push
//
// The other half of cmd/worker. The payload it sends is
// { title, body, path, tag } and nothing else — no prayer text, no names
// beyond a partner's first name — because this is drawn on a lock screen,
// which is the one part of Amorae that is not private (FR-NOTF-005).

self.addEventListener("push", (event) => {
  event.waitUntil(show(readPush(event.data)));
});

// readPush never throws. A push with no payload, or one this version of the
// worker does not understand, still has to put *something* on the screen: the
// browser shows its own "This site has been updated in the background" notice
// otherwise, which is worse than a vague one of ours.
function readPush(data) {
  const fallback = { title: "Amorae", body: "Something is waiting for you.", path: "/", tag: "amorae" };
  if (!data) return fallback;
  try {
    const payload = data.json();
    return {
      title: payload.title || fallback.title,
      body: payload.body || fallback.body,
      // Only ever somewhere inside this app. Resolved against our own origin
      // and checked, rather than tested for a leading "/": "//evil.example"
      // starts with a slash and is read by browsers as https://evil.example,
      // so a prefix check would walk someone straight off the site.
      path: samePath(payload.path),
      tag: payload.tag || fallback.tag,
    };
  } catch {
    return fallback;
  }
}

function samePath(path) {
  if (typeof path !== "string") return "/";
  try {
    const url = new URL(path, self.location.origin);
    return url.origin === self.location.origin ? url.pathname + url.search : "/";
  } catch {
    return "/";
  }
}

function show({ title, body, path, tag }) {
  return self.registration.showNotification(title, {
    body,
    // One per category: a second "a moment to pray" replaces the first rather
    // than stacking up a list of them.
    tag,
    renotify: false,
    icon: "/icons/icon-192.png",
    badge: "/icons/badge-72.png",
    data: { path },
  });
}

// Tapping it goes where the notification is about — reusing the open tab if
// there is one, so people do not collect windows.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const path = event.notification.data?.path || "/";

  event.waitUntil(
    (async () => {
      const clients = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      for (const client of clients) {
        if (new URL(client.url).origin === self.location.origin) {
          await client.focus();
          if ("navigate" in client) await client.navigate(path);
          return;
        }
      }
      await self.clients.openWindow(path);
    })(),
  );
});
