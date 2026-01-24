/**
 * Утилита для работы с Intersection Observer
 * Позволяет отслеживать видимость элементов для lazy loading
 */

interface ObserverOptions {
	rootMargin?: string;
	threshold?: number | number[];
	root?: Element | null;
}

interface IntersectionCallback {
	(isVisible: boolean, entry: IntersectionObserverEntry): void;
}

/**
 * Создает и управляет Intersection Observer
 */
export function createObserver(
	element: Element,
	callback: IntersectionCallback,
	options: ObserverOptions = {}
) {
	const { rootMargin = '100px', threshold = 0.01, root = null } = options;

	const observer = new IntersectionObserver(
		(entries) => {
			entries.forEach((entry) => {
				callback(entry.isIntersecting, entry);
			});
		},
		{
			root,
			rootMargin,
			threshold
		}
	);

	observer.observe(element);

	return () => {
		observer.disconnect();
	};
}

/**
 * Hook для Svelte - легко использовать lazy loading
 */
export function lazyLoad(node: Element, callback: () => void, options: ObserverOptions = {}) {
	let called = false;

	const disconnect = createObserver(
		node,
		(isVisible) => {
			if (isVisible && !called) {
				called = true;
				callback();
			}
		},
		options
	);

	return {
		destroy() {
			disconnect();
		}
	};
}

/**
 * Utilities для lazy loading контента
 */
export const lazyLoadConfig = {
	eager: {
		rootMargin: '0px',
		threshold: 1.0
	},
	normal: {
		rootMargin: '100px',
		threshold: 0.01
	},
	lazy: {
		rootMargin: '500px',
		threshold: 0
	}
};

/**
 * Проверяет поддержку Intersection Observer
 */
export function supportsIntersectionObserver(): boolean {
	return typeof window !== 'undefined' && 'IntersectionObserver' in window;
}

/**
 * Batch observe для нескольких элементов одновременно
 */
export function observeMany(
	elements: Element[],
	callback: (element: Element, isVisible: boolean) => void,
	options: ObserverOptions = {}
) {
	const { rootMargin = '100px', threshold = 0.01, root = null } = options;

	const observer = new IntersectionObserver(
		(entries) => {
			entries.forEach((entry) => {
				callback(entry.target, entry.isIntersecting);
			});
		},
		{
			root,
			rootMargin,
			threshold
		}
	);

	elements.forEach((el) => observer.observe(el));

	return () => {
		observer.disconnect();
	};
}
