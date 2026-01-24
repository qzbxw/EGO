import { browser } from '$app/environment';
import { _, locale, waitLocale } from 'svelte-i18n';
import { get as getStore } from 'svelte/store';
import '$lib/i18n';
import { redirect, isRedirect } from '@sveltejs/kit';
import { initAuthStore, auth, setAccessToken } from '$lib/stores/auth.svelte.ts';
import { connectionManager } from '$lib/connection-manager';
import { api, ApiError } from '$lib/api';
import { setInitialSessions } from '$lib/stores/sessions.svelte.ts';
import type { ChatSession, User } from '$lib/types';
import '../app.css';
export const load = async ({ data, url }) => {
	const defaultLocale = data.initialLocale || 'ru';
	const initLocale = browser ? window.localStorage.getItem('locale') || defaultLocale : defaultLocale;
	// Always set locale in load function to ensure SSR works correctly
	locale.set(initLocale);
	await waitLocale();
	if (browser) {
		try {
			const rawToken = url.searchParams.get('bypass_token') || url.searchParams.get('token');
			if (rawToken) {
				console.log(' Early capture: found bypass token in URL (not storing yet)');
			}
		} catch {}
		try {
			const isMaintenancePage = url.pathname === '/maintenance';
			if (!isMaintenancePage) {
				const {
					checkMaintenanceStatus,
					getStoredBypassToken,
					validateBypassToken,
					storeBypassToken
				} = await import('$lib/api/maintenance');
				let hasValidBypass = false;
				const rawToken = url.searchParams.get('bypass_token') || url.searchParams.get('token');
				if (rawToken) {
					hasValidBypass = await validateBypassToken(rawToken);
					if (hasValidBypass) {
						storeBypassToken(rawToken);
						try {
							const u = new URL(window.location.href);
							u.searchParams.delete('bypass_token');
							u.searchParams.delete('token');
							window.history.replaceState({}, '', u.toString());
						} catch {}
					}
				} else {
					hasValidBypass = !!getStoredBypassToken();
				}
				if (!hasValidBypass) {
					const status = await checkMaintenanceStatus();
					if (status.maintenance && !status.isChatOnly) {
						throw redirect(307, '/maintenance');
					}
				}
			}
		} catch (e: unknown) {
			if (isRedirect(e)) {
				throw e;
			}
		}
		try {
			const pathname = url.pathname;
			const isPublic = [
				'/',
				'/login',
				'/register',
				'/knowego',
				'/terms',
				'/privacy',
				'/maintenance'
			].includes(pathname);
		} catch (e: unknown) {
			if (isRedirect(e)) {
				throw e;
			}
		}
		initAuthStore();
		const pathname = url.pathname;
		const isProtectedRoute = ![
			'/login',
			'/register',
			'/',
			'/terms',
			'/privacy',
			'/knowego',
			'/maintenance'
		].includes(pathname);
		{
			const isMaintenancePage = url.pathname === '/maintenance';
			const { maintenanceStore } = await import('$lib/stores/maintenance-store.svelte.ts');
			const isMaintenanceActive = maintenanceStore.isMaintenanceActive;
			if (
				!isMaintenancePage &&
				!isMaintenanceActive &&
				auth.accessToken
			) {
				// No longer needed: connectionManager.connect();
			}
		}
		if (isProtectedRoute) {
			try {
				if (!auth.accessToken && auth.refreshToken) {
					const res = await api.post<{ access_token: string }>('/auth/refresh', {
						refresh_token: auth.refreshToken
					});
					if (res?.access_token) {
						setAccessToken(res.access_token);
					}
				}
				if (!auth.accessToken) {
					throw redirect(307, '/login');
				}
				const tryRefreshOnce = async () => {
					if (!auth.refreshToken) return false;
					try {
						const res = await api.post<{ access_token: string }>('/auth/refresh', {
							refresh_token: auth.refreshToken
						});
						if (res?.access_token) {
							setAccessToken(res.access_token);
							return true;
						}
					} catch {}
					return false;
				};
				try {
					await api.get<User>('/me');
				} catch (e: unknown) {
					if (e instanceof ApiError && e.status === 401 && (await tryRefreshOnce())) {
						await api.get<User>('/me');
					} else {
						throw e;
					}
				}
				let sessions: ChatSession[];
				try {
					sessions = await api.get<ChatSession[]>('/sessions');
				} catch (e: unknown) {
					if (e instanceof ApiError && e.status === 401 && (await tryRefreshOnce())) {
						sessions = await api.get<ChatSession[]>('/sessions');
					} else {
						throw e;
					}
				}
				setInitialSessions(sessions);
			} catch (err: unknown) {
				if (isRedirect(err)) {
					throw err;
				}
				if (err instanceof ApiError) {
					if (err.status === 503) {
						const urlToken = url.searchParams.get('bypass_token') || url.searchParams.get('token');
						const target = urlToken
							? `/maintenance?bypass_token=${encodeURIComponent(urlToken)}`
							: '/maintenance';
						throw redirect(307, target);
					}
					if (err.status === 401 || err.status === 403 || err.status === 502) {
						throw err;
					}
				}
				if (browser) {
					try {
						const { toast } = await import('svelte-sonner');
						toast.error((err as Error)?.message || getStore(_)('errors.generic'));
					} catch (_) {
						console.error('Load error:', (err as Error)?.message || err);
					}
				}
			}
		}
	}
	return data;
};
