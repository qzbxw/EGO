import { browser } from '$app/environment';

let globalLoading = $state(false);
let loadingText = $state('Загрузка...');
let loadingVariant = $state<'brain' | 'sparkles' | 'zap' | 'pulse'>('brain');
export const loadingStore = {
	get isLoading() {
		return globalLoading;
	},
	get text() {
		return loadingText;
	},
	get variant() {
		return loadingVariant;
	}
};
export function showLoading(text?: string, variant?: 'brain' | 'sparkles' | 'zap' | 'pulse') {
	if (browser) {
		globalLoading = true;
		if (text) loadingText = text;
		if (variant) loadingVariant = variant;
	}
}
export function hideLoading() {
	if (browser) {
		globalLoading = false;
	}
}
export async function withLoading<T>(
	fn: () => Promise<T>,
	text?: string,
	variant?: 'brain' | 'sparkles' | 'zap' | 'pulse'
): Promise<T> {
	try {
		showLoading(text, variant);
		return await fn();
	} finally {
		hideLoading();
	}
}
export async function withLoadingAndErrorHandling<T>(
	fn: () => Promise<T>,
	text?: string,
	variant?: 'brain' | 'sparkles' | 'zap' | 'pulse',
	onError?: (error: Error) => void
): Promise<T | null> {
	try {
		showLoading(text, variant);
		return await fn();
	} catch (error) {
		console.error('Operation failed:', error);
		if (onError) {
			onError(error as Error);
		}
		return null;
	} finally {
		hideLoading();
	}
}
