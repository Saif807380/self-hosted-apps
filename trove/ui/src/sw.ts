/// <reference lib="webworker" />
// Trove's service worker. It keeps what the generated one did (precache the
// app, offline navigation, cache covers and fonts, update itself) and adds
// web push for sale alerts. Built by vite-plugin-pwa (injectManifest).

import { clientsClaim } from 'workbox-core'
import { ExpirationPlugin } from 'workbox-expiration'
import { cleanupOutdatedCaches, createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'
import { CacheFirst } from 'workbox-strategies'

declare let self: ServiceWorkerGlobalScope

// ── App shell ──

precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()
registerRoute(new NavigationRoute(createHandlerBoundToURL('/index.html')))

registerRoute(
  ({ url }) => url.pathname.startsWith('/uploads/'),
  new CacheFirst({
    cacheName: 'uploads-cache',
    plugins: [new ExpirationPlugin({ maxEntries: 500, maxAgeSeconds: 60 * 60 * 24 * 30 })],
  }),
)
registerRoute(
  ({ url }) => url.pathname.startsWith('/fonts/'),
  new CacheFirst({
    cacheName: 'fonts-cache',
    plugins: [new ExpirationPlugin({ maxEntries: 10, maxAgeSeconds: 60 * 60 * 24 * 365 })],
  }),
)

// registerType: 'autoUpdate' — a new version takes over straight away.
self.skipWaiting()
clientsClaim()

// ── Sale alerts ──

/** What the server sends; see backend/internal/push.Notification. */
interface SalePush {
  title: string
  body: string
  url: string
  icon?: string
  tag?: string
}

self.addEventListener('push', event => {
  const data = parsePush(event.data)
  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: data.icon || '/icon-192.png',
      badge: '/icon-192.png',
      tag: data.tag, // a newer alert for the same game replaces the older one
      data: { url: data.url },
    }),
  )
})

function parsePush(data: PushMessageData | null): SalePush {
  const fallback = { title: 'Trove', body: '', url: '/games?tab=wishlist' }
  if (!data) return fallback
  try {
    return { ...fallback, ...(data.json() as Partial<SalePush>) }
  } catch {
    return { ...fallback, body: data.text() }
  }
}

// Tapping an alert opens the game in Trove, reusing an open window if any.
self.addEventListener('notificationclick', event => {
  event.notification.close()
  const target = new URL(event.notification.data?.url ?? '/games', self.location.origin).href
  event.waitUntil((async () => {
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
    const open = windows.find(w => new URL(w.url).origin === self.location.origin)
    if (open) {
      await open.focus()
      await open.navigate(target)
      return
    }
    await self.clients.openWindow(target)
  })())
})
