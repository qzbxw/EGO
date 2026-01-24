import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import type { User } from '$lib/types';
import { clearUserSessions } from './sessions.svelte.ts';

// Helper to get initial value from localStorage
function getStorageItem(key: string): string | null {
	if (browser) {
		return localStorage.getItem(key);
	}
	return null;
}

// Initialize reactive state
let user = $state<User | null>(browser ? (() => {
    try {
        const val = localStorage.getItem('user');
        return val ? JSON.parse(val) : null;
    } catch { return null; }
})() : null);

let accessToken = $state<string | null>(getStorageItem('accessToken'));
let refreshToken = $state<string | null>(getStorageItem('refreshToken'));

export const auth = {
	get user() {
		return user;
	},
	get accessToken() {
		return accessToken;
	},
	get refreshToken() {
		return refreshToken;
	}
};

export function setAuthData(userData: User, access: string, refresh: string) {
	if (browser) {
		localStorage.setItem('user', JSON.stringify(userData));
		localStorage.setItem('accessToken', access);
		localStorage.setItem('refreshToken', refresh);
		user = userData;
		accessToken = access;
		refreshToken = refresh;
	}
}

export function setAccessToken(token: string) {
	if (browser) {
		accessToken = token;
		localStorage.setItem('accessToken', token);
	}
}

export function clearAuthData() {
	if (browser) {
		user = null;
		accessToken = null;
		refreshToken = null;
		localStorage.removeItem('user');
		localStorage.removeItem('accessToken');
		localStorage.removeItem('refreshToken');
		clearUserSessions();
	}
}

export function logout() {
	clearAuthData();
	try {
		if (browser) goto('/login');
	} catch {}
}

export function initAuthStore() {
	if (browser) {
		// Sync auth state across tabs
		window.addEventListener('storage', (event) => {
			if (event.key === 'accessToken') {
				accessToken = event.newValue;
				if (!event.newValue) {
					// Token removed in another tab
					user = null;
					refreshToken = null;
				}
			}
			if (event.key === 'refreshToken') {
				refreshToken = event.newValue;
			}
			if (event.key === 'user') {
				try {
					user = event.newValue ? JSON.parse(event.newValue) : null;
				} catch {
					user = null;
				}
			}
		});
	}
}
