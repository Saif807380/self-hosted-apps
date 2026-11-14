/// <reference types="vite/client" />
import { useCallback, useEffect, useState } from 'react'
import { gamingApi } from '@/services/gaming'

/**
 * unsupported: this browser (or the dev server, which has no service worker)
 *   can't do web push. unavailable: the server has no VAPID keys. blocked:
 *   notifications were denied in browser settings.
 */
export type PushState = 'loading' | 'unsupported' | 'unavailable' | 'blocked' | 'off' | 'on'

/** Sale-alert subscription for this browser. */
export function usePushAlerts() {
  const [state, setState] = useState<PushState>('loading')
  const [publicKey, setPublicKey] = useState('')
  // Why alerts are unsupported here, in the browser's own words when known.
  const [detail, setDetail] = useState('')

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const next = await detect()
      if (!cancelled) {
        setPublicKey(next.publicKey)
        setDetail(next.detail ?? '')
        setState(next.state)
      }
    })().catch(() => !cancelled && setState('unavailable'))
    return () => { cancelled = true }
  }, [])

  const enable = useCallback(async () => {
    if (await Notification.requestPermission() !== 'granted') {
      setState('blocked')
      return
    }
    const reg = await navigator.serviceWorker.ready
    const existing = await reg.pushManager.getSubscription()
    if (existing && keyDiffers(existing, publicKey)) await existing.unsubscribe()
    const sub = await subscribe(reg, publicKey)
    await register(sub)
    setState('on')
  }, [publicKey])

  const disable = useCallback(async () => {
    const sub = await currentSubscription()
    if (sub) {
      await gamingApi.unsubscribePush(sub.endpoint).catch(() => {})
      await sub.unsubscribe()
    }
    setState('off')
  }, [])

  return { state, detail, enable, disable, sendTest: gamingApi.sendTestPush }
}

// How long to wait for the service worker. It's registered only once the
// page has fully loaded, so on a first visit it isn't there yet when this
// runs; in the dev server there's none at all.
const SW_WAIT_MS = 8000

async function detect(): Promise<{ state: PushState; publicKey: string; detail?: string }> {
  const missing = missingApi()
  if (missing) return { state: 'unsupported', publicKey: '', detail: missing }
  const sw = await startServiceWorker()
  if (typeof sw === 'string') return { state: 'unsupported', publicKey: '', detail: sw }
  const config = await gamingApi.pushConfig()
  const publicKey = config.publicKey ?? ''
  if (!config.enabled || !publicKey) return { state: 'unavailable', publicKey }
  if (Notification.permission === 'denied') return { state: 'blocked', publicKey }

  const sub = await currentSubscription()
  if (!sub) return { state: 'off', publicKey }
  // Trust an existing subscription. Its key can't be relied on to check it
  // against the server's (Firefox for Android reports none), so a changed
  // server key is handled when alerts are turned on instead.
  // Re-register on every visit in case the server dropped it.
  await register(sub).catch(() => {})
  return { state: 'on', publicKey }
}

function missingApi(): string {
  if (!('serviceWorker' in navigator)) {
    return 'This browser has service workers turned off (a private window, or Firefox set to delete site data when closed).'
  }
  if (!('PushManager' in window)) return "This browser doesn't support web push."
  if (!('Notification' in window)) return "This browser doesn't support notifications."
  return ''
}

/**
 * Returns the active service worker registration, or why there isn't one.
 * The page registers it once fully loaded (registerSW.js); if that hasn't
 * happened, register here so a failure comes back with the browser's reason.
 */
async function startServiceWorker(): Promise<ServiceWorkerRegistration | string> {
  if (!import.meta.env.PROD) return 'Alerts only work in a production build; the dev server has no service worker.'
  if (!(await navigator.serviceWorker.getRegistration())) {
    try {
      await navigator.serviceWorker.register('/sw.js', { scope: '/' })
    } catch (e) {
      return `The service worker couldn't start: ${e instanceof Error ? e.message : String(e)}`
    }
  }
  const ready = await Promise.race([
    navigator.serviceWorker.ready,
    new Promise<null>(resolve => setTimeout(() => resolve(null), SW_WAIT_MS)),
  ])
  return ready ?? "The service worker didn't finish starting. Reload the page and try again."
}

async function currentSubscription() {
  const reg = await navigator.serviceWorker.getRegistration()
  return reg?.pushManager.getSubscription() ?? null
}

async function subscribe(reg: ServiceWorkerRegistration, publicKey: string): Promise<PushSubscription> {
  const options = { userVisibleOnly: true, applicationServerKey: base64UrlToBytes(publicKey) }
  try {
    return await reg.pushManager.subscribe(options)
  } catch (e) {
    // InvalidStateError: a subscription made with a different (old) server
    // key is still there. Replace it.
    const existing = await reg.pushManager.getSubscription()
    if (!(e instanceof DOMException && e.name === 'InvalidStateError') || !existing) throw e
    await existing.unsubscribe()
    return reg.pushManager.subscribe(options)
  }
}

// The endpoint this browser last registered, so a replacement can retire it
// on the server rather than leave a duplicate behind.
const ENDPOINT_KEY = 'trove:push-endpoint'

async function register(sub: PushSubscription) {
  const json = sub.toJSON()
  await gamingApi.subscribePush({
    endpoint: sub.endpoint,
    p256dh: json.keys?.p256dh ?? '',
    auth: json.keys?.auth ?? '',
  })
  try {
    const previous = localStorage.getItem(ENDPOINT_KEY)
    if (previous && previous !== sub.endpoint) await gamingApi.unsubscribePush(previous).catch(() => {})
    localStorage.setItem(ENDPOINT_KEY, sub.endpoint)
  } catch {
    // Storage blocked: duplicates are cleaned up when the push service
    // reports them gone.
  }
}

/** True only when the subscription's key is known and isn't the server's. */
function keyDiffers(sub: PushSubscription, publicKey: string): boolean {
  const current = sub.options?.applicationServerKey
  if (!current) return false
  const a = new Uint8Array(current)
  const b = base64UrlToBytes(publicKey)
  return a.length !== b.length || a.some((v, i) => v !== b[i])
}

function base64UrlToBytes(value: string): Uint8Array<ArrayBuffer> {
  const base64 = (value + '='.repeat((4 - (value.length % 4)) % 4)).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  const bytes = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i)
  return bytes
}
