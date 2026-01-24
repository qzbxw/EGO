import { memoryStore } from './memory.svelte.ts';
import { browser } from '$app/environment';

// Only initialize reactive state on browser
let theme = $state<'dark' | 'light'>('dark');
let backgroundSpheres = $state<boolean>(true);

function persist() {
	if (!browser) return;
	
	try {
		localStorage.setItem('pref-theme', theme);
		localStorage.setItem('pref-background-spheres', backgroundSpheres ? '1' : '0');
		localStorage.setItem('pref-memory-enabled', memoryStore.memoryEnabled ? '1' : '0');
	} catch (error) {
		console.error('[Preferences] Failed to persist preferences to localStorage:', error);
	}

	// Also save to cookies for SSR
	try {
		document.cookie = `pref-theme=${theme}; path=/; max-age=31536000; SameSite=Lax`;
		document.cookie = `pref-background-spheres=${backgroundSpheres ? '1' : '0'}; path=/; max-age=31536000; SameSite=Lax`;
		document.cookie = `pref-memory-enabled=${memoryStore.memoryEnabled ? '1' : '0'}; path=/; max-age=31536000; SameSite=Lax`;
	} catch (error) {
		console.error('[Preferences] Failed to persist preferences to cookies:', error);
	}
}
export const preferencesStore = {
	get theme() {
		return theme;
	},
	get backgroundSpheres() {
		return backgroundSpheres;
	},
	get memoryEnabled() {
		return memoryStore.memoryEnabled;
	}
};
export function loadPreferences(ssrData?: {
	theme?: 'dark' | 'light';
	backgroundSpheres?: boolean;
	memoryEnabled?: boolean;
}) {
	if (!browser) return;
	
	try {
		// Use SSR data if available, otherwise load from localStorage
		if (ssrData) {
			if (ssrData.theme) theme = ssrData.theme;
			if (ssrData.backgroundSpheres !== undefined) backgroundSpheres = ssrData.backgroundSpheres;
			if (ssrData.memoryEnabled !== undefined) memoryStore.setMemoryEnabled(ssrData.memoryEnabled);
		} else {
			const t = localStorage.getItem('pref-theme');
			if (t === 'light' || t === 'dark') theme = t;
			const b = localStorage.getItem('pref-background-spheres');
			if (b === '0' || b === '1') backgroundSpheres = b === '1';
			const m = localStorage.getItem('pref-memory-enabled');
			if (m === '0' || m === '1') memoryStore.setMemoryEnabled(m === '1');
		}

		try {
			const root = document.documentElement;
			if (theme === 'light') {
				root.classList.add('theme-light');
			} else {
				root.classList.remove('theme-light');
			}
		} catch (error) {
			console.error('[Preferences] Failed to apply theme to document:', error);
		}
	} catch (error) {
		console.error('[Preferences] Failed to load preferences from localStorage:', error);
	}
}
export function setTheme(next: 'dark' | 'light') {
	if (!browser) return;
	theme = next;
	persist();
	try {
		const root = document.documentElement;
		if (next === 'light') {
			root.classList.add('theme-light');
		} else {
			root.classList.remove('theme-light');
		}
	} catch (error) {
		console.error('[Preferences] Failed to apply theme to document:', error);
	}
}
export function setBackgroundSpheres(enabled: boolean) {
	if (!browser) return;
	backgroundSpheres = enabled;
	persist();
}
export function setMemoryEnabled(enabled: boolean) {
	if (!browser) return;
	memoryStore.setMemoryEnabled(enabled);
	persist();
}
