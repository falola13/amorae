/* Amorae service worker.
   - App shell and visited pages stay readable offline (network first, cache fallback).
   - Build assets, fonts and icons are cache first.
   - Push handlers are ready for the Go API's Web Push messages. */
const VERSION = 'amorae-v1';
const SHELL = ['/', '/together', '/prayers', '/prayers/mode', '/history', '/settings', '/manifest.webmanifest', '/icons/icon-192.png'];

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(VERSION).then((c) => c.addAll(SHELL).catch(() => undefined)).then(() => self.skipWaiting()));
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== VERSION).map((k) => caches.delete(k)))).then(() => self.clients.claim()),
  );
});

const isStatic = (url) => url.pathname.startsWith('/_next/static/') || url.pathname.startsWith('/icons/') || url.pathname.startsWith('/splash/') || url.pathname.startsWith('/brand/') || /\.(woff2?|png|svg|ico)$/.test(url.pathname);

self.addEventListener('fetch', (event) => {
  const req = event.request;
  if (req.method !== 'GET') return;
  const url = new URL(req.url);
  if (url.origin !== self.location.origin) return;
  if (url.pathname.startsWith('/api/')) return; // API calls are handled by the app (queued when offline)

  if (isStatic(url)) {
    event.respondWith(
      caches.match(req).then((hit) => hit || fetch(req).then((res) => {
        const copy = res.clone();
        caches.open(VERSION).then((c) => c.put(req, copy));
        return res;
      })),
    );
    return;
  }

  // Pages and RSC payloads: network first so content is fresh, cache so it survives offline.
  event.respondWith(
    fetch(req)
      .then((res) => {
        if (res.ok) {
          const copy = res.clone();
          caches.open(VERSION).then((c) => c.put(req, copy));
        }
        return res;
      })
      .catch(() => caches.match(req).then((hit) => hit || caches.match(url.pathname) || caches.match('/'))),
  );
});

self.addEventListener('push', (event) => {
  let data = { title: 'Amorae', body: 'A quiet moment for the two of you.', url: '/' };
  try { if (event.data) data = { ...data, ...event.data.json() }; } catch (_) {}
  event.waitUntil(
    self.registration.showNotification(data.title, { body: data.body, icon: '/icons/icon-192.png', badge: '/icons/badge-96.png', data: { url: data.url } }),
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const target = (event.notification.data && event.notification.data.url) || '/';
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((list) => {
      for (const client of list) { if ('focus' in client) { client.navigate(target); return client.focus(); } }
      return self.clients.openWindow(target);
    }),
  );
});
