/**
 * Утилита для динамического импорта компонентов и модулей
 * Позволяет разбить код на chunks и загружать их по мере необходимости
 */

interface DynamicImportOptions {
	timeout?: number;
	onError?: (error: Error) => void;
	preload?: boolean;
}

/**
 * Создает lazy-loaded версию модуля
 * @param importFn - функция для импорта модуля (() => import(...))
 * @param options - опции загрузки
 */
export async function dynamicImport<T>(
	importFn: () => Promise<{ default: T }>,
	options: DynamicImportOptions = {}
): Promise<T> {
	const { timeout = 30000, onError } = options;

	try {
		const timeoutPromise = new Promise<never>((_, reject) =>
			setTimeout(() => reject(new Error('Dynamic import timeout')), timeout)
		);

		const module = await Promise.race([importFn(), timeoutPromise]);
		return module.default;
	} catch (error) {
		const err = error instanceof Error ? error : new Error(String(error));
		onError?.(err);
		throw err;
	}
}

/**
 * Preload модуль без использования
 * Полезно для предварительной загрузки перед использованием
 */
export async function preloadModule(importFn: () => Promise<unknown>): Promise<void> {
	try {
		await importFn();
	} catch (error) {
		console.warn('[preloadModule] Failed to preload:', error);
	}
}

/**
 * Кеш для загруженных модулей
 */
const moduleCache = new Map<string, Promise<{ default: unknown }>>();

/**
 * Загружает модуль с кешированием
 */
export function cachedDynamicImport<T>(
	key: string,
	importFn: () => Promise<{ default: T }>
): Promise<T> {
	if (!moduleCache.has(key)) {
		moduleCache.set(key, importFn() as Promise<{ default: unknown }>);
	}

	return moduleCache.get(key)!.then((module) => module.default as T);
}

/**
 * Очищает кеш модулей (для devtools/hot reload)
 */
export function clearModuleCache(): void {
	moduleCache.clear();
}

export async function preloadBatch(
	importFns: Array<() => Promise<unknown>>,
	options: { concurrency?: number } = {}
): Promise<void> {
	const { concurrency = 3 } = options;
	const chunks = [];

	for (let i = 0; i < importFns.length; i += concurrency) {
		chunks.push(importFns.slice(i, i + concurrency));
	}

	for (const chunk of chunks) {
		await Promise.all(chunk.map((fn) => preloadModule(fn).catch(() => {})));
	}
}

/**
 * Intelligent preload - загружает модули на основе приоритета
 */
export async function smartPreload(
	modules: Array<{
		key: string;
		importFn: () => Promise<unknown>;
		priority?: 'high' | 'normal' | 'low';
	}>
): Promise<void> {
	const grouped = {
		high: modules.filter((m) => m.priority === 'high'),
		normal: modules.filter((m) => m.priority === 'normal'),
		low: modules.filter((m) => m.priority === 'low')
	};

	// Загружаем high-priority сразу
	await Promise.all(grouped.high.map((m) => preloadModule(m.importFn).catch(() => {})));

	// Normal в следующем микротаске
	queueMicrotask(async () => {
		await Promise.all(grouped.normal.map((m) => preloadModule(m.importFn).catch(() => {})));
	});

	// Low в idle time
	if (window.requestIdleCallback) {
		window.requestIdleCallback(() => {
			grouped.low.forEach((m) => {
				preloadModule(m.importFn).catch(() => {});
			});
		});
	}
}

/**
 * Создает retry-logic для импорта с fallback
 */
export async function importWithFallback<T>(
	primaryFn: () => Promise<{ default: T }>,
	fallbackFn: () => Promise<{ default: T }>,
	options: { maxRetries?: number } = {}
): Promise<T> {
	const { maxRetries = 2 } = options;

	for (let i = 0; i < maxRetries; i++) {
		try {
			return await dynamicImport(primaryFn);
		} catch (error) {
			if (i === maxRetries - 1) {
				try {
					return await dynamicImport(fallbackFn);
				} catch (fallbackError) {
					throw new Error(`Failed to import both primary and fallback: ${error}, ${fallbackError}`);
				}
			}
		}
	}

	throw new Error('Import failed');
}

/**
 * Создает promise-based loader для использования в компонентах
 * Реактивный способ загружать компоненты при необходимости
 */
export function createDeferredImport<T>(importFn: () => Promise<{ default: T }>) {
	let promise: Promise<T> | null = null;
	let loaded = false;
	let error: Error | null = null;

	return {
		load: async () => {
			if (loaded) return promise;

			if (!promise) {
				promise = dynamicImport(importFn)
					.then((module) => {
						loaded = true;
						return module;
					})
					.catch((err) => {
						error = err;
						throw err;
					});
			}

			return promise;
		},
		isLoaded: () => loaded,
		hasError: () => error !== null,
		getError: () => error
	};
}
