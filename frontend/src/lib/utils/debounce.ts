/**
 * Debounce utility to limit function execution rate
 * @param fn Function to debounce
 * @param delayMs Delay in milliseconds
 * @returns Debounced function
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
	fn: T,
	delayMs: number
): (...args: Parameters<T>) => void {
	let timeoutId: ReturnType<typeof setTimeout> | null = null;

	return function (this: unknown, ...args: Parameters<T>) {
		if (timeoutId) {
			clearTimeout(timeoutId);
		}

		timeoutId = setTimeout(() => {
			fn.apply(this, args);
			timeoutId = null;
		}, delayMs);
	};
}

/**
 * Debounce with immediate option - executes immediately on first call,
 * then debounces subsequent calls
 */
export function debounceImmediate<T extends (...args: unknown[]) => unknown>(
	fn: T,
	delayMs: number
): (...args: Parameters<T>) => void {
	let timeoutId: ReturnType<typeof setTimeout> | null = null;
	let isImmediate = true;

	return function (this: unknown, ...args: Parameters<T>) {
		if (isImmediate) {
			fn.apply(this, args);
			isImmediate = false;
		}

		if (timeoutId) {
			clearTimeout(timeoutId);
		}

		timeoutId = setTimeout(() => {
			isImmediate = true;
			timeoutId = null;
		}, delayMs);
	};
}
