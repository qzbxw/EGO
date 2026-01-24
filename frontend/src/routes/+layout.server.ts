import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies }) => {
	// Read theme and preferences from cookies for SSR
	// Note: With adapter-static, this runs at BUILD time, not runtime.
	// Cookies won't be available during prerender, so we use defaults.
	// Maintenance checks are handled client-side in +layout.ts
	let theme: 'dark' | 'light' = 'dark';
	let backgroundSpheres = true;
	let memoryEnabled = false;

	try {
		theme = (cookies.get('pref-theme') || 'dark') as 'dark' | 'light';
		backgroundSpheres = cookies.get('pref-background-spheres') !== '0';
		memoryEnabled = cookies.get('pref-memory-enabled') === '1';
	} catch {
		// Cookies not available during prerender - use defaults
	}

	return {
		initialLocale: 'ru' as const,
		theme,
		backgroundSpheres,
		memoryEnabled
	};
};
