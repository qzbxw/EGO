import type { ThemeName, ChatMode } from '$lib/types';
import { browser } from '$app/environment';

export type { ThemeName, ChatMode };

export function getAppLogo(theme: ThemeName): string {
	return theme === 'light' ? '/logolight.png' : '/logodark.png';
}

export function getModeLogo(mode: ChatMode, theme: ThemeName): string {
	const suffix = theme === 'light' ? 'light' : 'dark';
	if (mode === 'deeper') return `/deepermodelogo${suffix}.png`;
	if (mode === 'research') return `/researchmodelogo${suffix}.png`;
	return `/agentmodelogo${suffix}.png`;
}

/**
 * Centralized configuration for backend URLs
 * Uses environment variables as single source of truth
 */

function getBaseApiUrl(): string {
	const envUrl = import.meta.env.VITE_PUBLIC_EGO_BACKEND_URL as string | undefined;

	// Use environment variable if set and absolute
	if (envUrl && /^https?:\/\//i.test(envUrl)) {
		const trimmed = envUrl.replace(/\/+$/, '');
		// Ensure it ends with /api
		return trimmed.endsWith('/api') ? trimmed : `${trimmed}/api`;
	}

	// Default: use relative path (relies on proxy or same origin)
	return '/api';
}

function getPyBaseUrl(): string {
	const envUrl = import.meta.env.VITE_PUBLIC_EGO_PY_BACKEND_URL as string | undefined;

	// Use environment variable if set and absolute
	if (envUrl && /^https?:\/\//i.test(envUrl)) {
		return envUrl.replace(/\/+$/, '');
	}

	// Default: use relative path
	return '/py';
}

function getWebSocketUrl(token: string, bypassToken?: string): string {
	const baseUrl = getBaseApiUrl();
	const qs = new URLSearchParams({ token });
	if (bypassToken) {
		qs.set('bypass_token', bypassToken);
	}

	// If baseUrl is absolute HTTP(S), convert to WS(S)
	if (/^https?:\/\//i.test(baseUrl)) {
		const wsBase = baseUrl.replace(/^http/i, 'ws');
		return `${wsBase}/ws?${qs.toString()}`;
	}

	// Otherwise use current location protocol to determine ws/wss
	if (browser) {
		const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
		const host = location.host;
		return `${protocol}://${host}${baseUrl}/ws?${qs.toString()}`;
	}

	// Fallback (should not reach here in normal flow)
	return `/api/ws?${qs.toString()}`;
}

export const backendConfig = {
	/**
	 * Base URL for REST API endpoints
	 * Examples: '/api', 'https://api.example.com/api'
	 */
	get apiBaseUrl(): string {
		return getBaseApiUrl();
	},

	/**
	 * Base URL for Python backend endpoints
	 * Examples: '/py', 'https://py.example.com'
	 */
	get pyBaseUrl(): string {
		return getPyBaseUrl();
	},

	/**
	 * Get WebSocket URL with token
	 * @param token - Authentication token
	 * @param bypassToken - Optional maintenance bypass token
	 * @returns Full WebSocket URL
	 */
	getWebSocketUrl(token: string, bypassToken?: string): string {
		return getWebSocketUrl(token, bypassToken);
	}
};
