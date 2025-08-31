import { build, files, version } from '$service-worker';
const sw = self as unknown as ServiceWorkerGlobalScope;
const isDev = sw.location.hostname === 'localhost' || sw.location.hostname === '127.0.0.1';
const CACHE_NAME = `ego-cache-${version || 'dev'}`;
const safeBuild = Array.isArray(build) ? build : [];
const safeFiles = Array.isArray(files) ? files : [];
const ASSETS_TO_CACHE = Array.from(
    new Set([...safeBuild, ...safeFiles].filter((x): x is string => typeof x === 'string'))
);
if (isDev) {
    sw.addEventListener('install', (event) => {
        event.waitUntil(sw.skipWaiting());
    });
    sw.addEventListener('activate', (event) => {
        event.waitUntil(sw.clients.claim());
    });
} else {
    sw.addEventListener('install', (event) => {
        event.waitUntil(
            (async () => {
                try {
                    const cache = await caches.open(CACHE_NAME);
                    if (ASSETS_TO_CACHE.length > 0) {
                        await cache.addAll(ASSETS_TO_CACHE);
                    }
                } catch (e) {
                    console.warn('[SW] install cache error', e);
                } finally {
                    await sw.skipWaiting();
                }
            })()
        );
    });
    sw.addEventListener('activate', (event) => {
        event.waitUntil(
            (async () => {
                const keys = await caches.keys();
                await Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)));
                await sw.clients.claim(); 
            })()
        );
    });
    sw.addEventListener('fetch', (event) => {
        if (isDev) return;
        const { request } = event;
        if (request.method !== 'GET' || !request.url.startsWith('http')) {
            return;
        }
        const url = new URL(request.url);
        const isSameOrigin = url.origin === self.location.origin;
        if (request.mode === 'navigate') {
            event.respondWith(
                (async () => {
                    try {
                        return await fetch(request);
                    } catch {
                        const cachedResponse = await caches.match('/');
                        if (cachedResponse) return cachedResponse;
                        return new Response('Вы оффлайн', { status: 408, headers: { 'Content-Type': 'text/plain' } });
                    }
                })()
            );
            return;
        }
        if (!isSameOrigin || url.port === '8080') {
            event.respondWith(
                (async () => {
                    try {
                        return await fetch(request);
                    } catch {
                        return new Response('Offline', { status: 503 });
                    }
                })()
            );
            return;
        }
        const pathname = url.pathname;
        const isDynamicAPI = /^(?:\/(?:api\/)?(auth|sessions|logs|me|ws|user))(\/|$)/.test(pathname);
        if (isDynamicAPI) {
            event.respondWith(
                (async () => {
                    try {
                        return await fetch(request);
                    } catch {
                        return new Response('Offline', { status: 503 });
                    }
                })()
            );
            return;
        }
        const assetDestinations = new Set(['script', 'style', 'image', 'font', 'manifest']);
        if (assetDestinations.has(request.destination)) {
            event.respondWith(
                (async () => {
                    const cached = await caches.match(request);
                    if (cached) return cached;
                    const response = await fetch(request);
                    if (response && response.status === 200) {
                        const cache = await caches.open(CACHE_NAME);
                        cache.put(request, response.clone());
                    }
                    return response;
                })()
            );
            return;
        }
        event.respondWith(fetch(request));
    });
    sw.addEventListener('message', (event) => {
        if (isDev) return;
        if (event.data && event.data.type === 'SKIP_WAITING') {
            sw.skipWaiting();
        }
    });
}