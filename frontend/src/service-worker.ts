import { build, files, version } from '$service-worker';

const sw = self as unknown as ServiceWorkerGlobalScope;
const isDev = sw.location.hostname === 'localhost' || sw.location.hostname === '127.0.0.1';

// Cache strategies
const CACHE_NAMES = {
	app: `ego-app-${version || 'dev'}`,
	images: `ego-images-v1`,
	api: `ego-api-v1`,
	static: `ego-static-${version || 'dev'}`
};

const safeBuild = Array.isArray(build) ? build : [];
const safeFiles = Array.isArray(files) ? files : [];
const ASSETS_TO_CACHE = Array.from(
	new Set([...safeBuild, ...safeFiles].filter((x): x is string => typeof x === 'string'))
);

// Image cache strategy (limit to 50 items, max 10MB)
const IMAGE_CACHE_CONFIG = {
	maxItems: 50,
	maxAge: 30 * 24 * 60 * 60 * 1000, // 30 days
	maxSize: 10 * 1024 * 1024 // 10MB
};

// API cache strategy (limit to 100 items, max 5MB)
const API_CACHE_CONFIG = {
	maxItems: 100,
	maxAge: 5 * 60 * 1000, // 5 minutes
	maxSize: 5 * 1024 * 1024 // 5MB
};

/**
 * Очистка кеша при превышении лимитов
 */
async function cleanupCache(cacheName: string, config: typeof IMAGE_CACHE_CONFIG) {
	const cache = await caches.open(cacheName);
	const keys = await cache.keys();

	if (keys.length > config.maxItems) {
		// Удаляем старые записи
		for (let i = 0; i < keys.length - config.maxItems; i++) {
			await cache.delete(keys[i]);
		}
	}
}

/**
 * Стратегия Cache-first для статических активов
 */
async function cacheFirstStrategy(request: Request): Promise<Response> {
	const cached = await caches.match(request);
	if (cached) return cached;

	try {
		const response = await fetch(request);
		if (response && response.status === 200) {
			const cache = await caches.open(CACHE_NAMES.static);
			cache.put(request, response.clone());
		}
		return response;
	} catch (error) {
		return new Response('Offline', { status: 503 });
	}
}

/**
 * Стратегия Network-first для API
 */
async function networkFirstStrategy(request: Request): Promise<Response> {
	try {
		const response = await fetch(request);
		if (response && response.status === 200) {
			const cache = await caches.open(CACHE_NAMES.api);
			cache.put(request, response.clone());
			await cleanupCache(CACHE_NAMES.api, API_CACHE_CONFIG);
		}
		return response;
	} catch (error) {
		const cached = await caches.match(request);
		return cached || new Response('Offline', { status: 503 });
	}
}

/**
 * Стратегия для изображений (cache-first с network update)
 */
async function imageStrategy(request: Request): Promise<Response> {
	const cached = await caches.match(request);

	// Если есть в кеше, вернуть и попытаться обновить в фоне
	if (cached) {
		// Фоновое обновление
		fetch(request).then((response) => {
			if (response && response.status === 200) {
				const cache = caches.open(CACHE_NAMES.images);
				cache.then((c) => {
					c.put(request, response.clone());
					cleanupCache(CACHE_NAMES.images, IMAGE_CACHE_CONFIG);
				});
			}
		});
		return cached;
	}

	// Не в кеше, пытаемся загрузить
	try {
		const response = await fetch(request);
		if (response && response.status === 200) {
			const cache = await caches.open(CACHE_NAMES.images);
			cache.put(request, response.clone());
			await cleanupCache(CACHE_NAMES.images, IMAGE_CACHE_CONFIG);
		}
		return response;
	} catch (error) {
		// Возвращаем placeholder если нет кеша
		return new Response(
			'<svg xmlns="http://www.w3.org/2000/svg" width="400" height="300"><rect fill="#f0f0f0" width="400" height="300"/></svg>',
			{ headers: { 'Content-Type': 'image/svg+xml' } }
		);
	}
}

// ===== Event listeners =====

if (isDev) {
	// Dev mode - быстрое обновление
	sw.addEventListener('install', (event) => {
		event.waitUntil(sw.skipWaiting());
	});
	sw.addEventListener('activate', (event) => {
		event.waitUntil(sw.clients.claim());
	});
} else {
	// Prod mode - кеширование активов при установке
	sw.addEventListener('install', (event) => {
		event.waitUntil(
			(async () => {
				try {
					const cache = await caches.open(CACHE_NAMES.app);
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
				const cacheNamesToKeep = Object.values(CACHE_NAMES);
				await Promise.all(
					keys.filter((k) => !cacheNamesToKeep.includes(k)).map((k) => caches.delete(k))
				);
				await sw.clients.claim();
			})()
		);
	});

	sw.addEventListener('fetch', (event) => {
		const { request } = event;
		if (request.method !== 'GET' || !request.url.startsWith('http')) {
			return;
		}

		const url = new URL(request.url);
		const isSameOrigin = url.origin === self.location.origin;

		// Навигационные запросы (HTML)
		if (request.mode === 'navigate') {
			event.respondWith(
				(async () => {
					try {
						return await fetch(request);
					} catch {
						const cachedResponse = await caches.match('/');
						if (cachedResponse) return cachedResponse;
						return new Response('Вы оффлайн', {
							status: 408,
							headers: { 'Content-Type': 'text/plain; charset=utf-8' }
						});
					}
				})()
			);
			return;
		}

		// Кросс-ориджинные запросы
		if (!isSameOrigin || url.port === '8080') {
			event.respondWith(networkFirstStrategy(request));
			return;
		}

		const pathname = url.pathname;

		// Динамические API запросы
		const isDynamicAPI = /^(?:\/(?:api\/)?(auth|sessions|logs|me|ws|user))(\/|$)/.test(pathname);
		if (isDynamicAPI) {
			event.respondWith(networkFirstStrategy(request));
			return;
		}

		// Изображения
		if (request.destination === 'image') {
			event.respondWith(imageStrategy(request));
			return;
		}

		// Статические активы (скрипты, стили, шрифты)
		const assetDestinations = new Set(['script', 'style', 'font', 'manifest']);
		if (assetDestinations.has(request.destination)) {
			event.respondWith(cacheFirstStrategy(request));
			return;
		}

		// По умолчанию - network-first
		event.respondWith(networkFirstStrategy(request));
	});

	sw.addEventListener('message', (event) => {
		if (event.data && event.data.type === 'SKIP_WAITING') {
			sw.skipWaiting();
		}
	});
}
