import { browser } from '$app/environment';
import { _ } from 'svelte-i18n';
import { get } from 'svelte/store';
import { memoryStore } from '$lib/stores/memory.svelte.ts';
import { backendConfig } from '$lib/config';
import { auth } from '$lib/stores/auth.svelte.ts';

export type UserSettings = { llm_provider: string; llm_model?: string; api_key?: string };

export class ApiError extends Error {
	status: number;
	constructor(message: string, status: number) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

export class MaintenanceModeError extends ApiError {
	bypassToken: string | null;
	constructor(bypassToken: string | null = null) {
		super('Service is in maintenance mode', 503);
		this.name = 'MaintenanceModeError';
		this.bypassToken = bypassToken;
	}
}
async function requestWithBase(
	base: string,
	path: string,
	options: RequestInit = {},
	customFetch?: typeof fetch,
	allowRetry401 = true
) {
	const { auth } = await import('$lib/stores/auth.svelte.ts');
	try {
		const { memoryEnabled } = memoryStore;
		window.__egoMemoryEnabled = memoryEnabled;
	} catch (error) {
		console.error('[API] Failed to access memoryStore:', error);
	}
	const isAuthEndpoint = path.startsWith('/auth/');
	const fetcher = customFetch || (browser ? window.fetch : fetch);
	const headers = new Headers(options.headers || {});
	const token = auth.accessToken;
	const userId = auth.user?.id ? String(auth.user.id) : null;
	if (token) {
		headers.set('Authorization', `Bearer ${token}`);
	}
	if (userId) {
		headers.set('X-User-ID', userId);
	}
	let bypassToken: string | null = null;
	try {
		const { getStoredBypassToken, storeBypassToken } = await import('$lib/api/maintenance');
		bypassToken = getStoredBypassToken();
		if (!bypassToken && browser) {
			try {
				const params = new URLSearchParams(window.location.search);
				const urlToken = params.get('bypass_token') || params.get('token');
				if (urlToken) {
					bypassToken = urlToken;
					storeBypassToken(urlToken);
					console.log('🔧 Captured bypass token directly from URL for API request');
				}
			} catch (error) {
				console.error('[API] Failed to capture bypass token from URL:', error);
			}
		}
		if (bypassToken) {
			console.log(
				'🔧 Adding bypass token to API request:',
				`${bypassToken.substring(0, 20)}...`,
				'for path:',
				path
			);
			headers.set('X-Bypass-Token', bypassToken);
		} else {
			console.log('🔧 No bypass token found for API request:', path);
		}
	} catch (error) {
		console.log('🔧 Error getting bypass token for API request:', error);
	}
	try {
		const { memoryEnabled } = memoryStore;
		headers.set('X-Memory-Enabled', memoryEnabled ? '1' : '0');
	} catch (error) {
		console.error('[API] Failed to set memory-enabled header:', error);
	}
	const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData;
	if (!headers.has('Content-Type') && options.body && !isFormData) {
		headers.set('Content-Type', 'application/json');
	}
	const requestOptions: RequestInit = { ...options, headers };
	let response = await fetcher(`${base}${path}`, requestOptions);
	try {
		const ct = response.headers.get('Content-Type') || response.headers.get('content-type') || '';
		if (response.ok && /text\/html/i.test(ct)) {
			throw new ApiError('Unexpected HTML response from API', 502);
		}
	} catch (e) {
		if (e instanceof ApiError) throw e;
	}
	if (response.status === 503) {
		let bypassToken: string | null = null;
		try {
			const { getStoredBypassToken } = await import('$lib/api/maintenance');
			bypassToken = getStoredBypassToken();
		} catch (error) {
			console.error('[API] Failed to check bypass token for 503 response:', error);
		}

		// Throw specialized error instead of navigating directly
		// The UI layer (e.g., +layout.svelte) will handle the navigation
		if (!bypassToken) {
			console.log('🔧 API returned 503; service is in maintenance mode');
			throw new MaintenanceModeError(null);
		} else {
			console.log('🔧 503 received but bypass token present; continuing');
		}
	}
	if (response.status === 401 && !isAuthEndpoint && base === backendConfig.apiBaseUrl) {
		try {
			const { auth, setAccessToken } = await import('$lib/stores/auth.svelte.ts');
			if (allowRetry401 && auth.refreshToken) {
				const refreshHeaders: Record<string, string> = {
					'Content-Type': 'application/json'
				};
				if (bypassToken) {
					refreshHeaders['X-Bypass-Token'] = bypassToken;
				}
				const refreshResp = await fetcher(`${backendConfig.apiBaseUrl}/auth/refresh`, {
					method: 'POST',
					headers: refreshHeaders,
					body: JSON.stringify({ refresh_token: auth.refreshToken })
				});
				if (refreshResp.ok) {
					let newToken: string | null = null;
					try {
						const data = await refreshResp.json();
						newToken = data?.access_token || null;
					} catch (error) {
						console.error('[API] Failed to parse refresh token response:', error);
					}
					if (newToken) {
						setAccessToken(newToken);
						headers.set('Authorization', `Bearer ${newToken}`);
						const retryOptions: RequestInit = { ...requestOptions, headers };
						const retry = await fetcher(`${base}${path}`, retryOptions);
						if (!retry.ok) {
							console.warn('[API] Retry request failed after token refresh:', {
								status: retry.status,
								path
							});
							response = retry;
						} else {
							return retry;
						}
					}
				} else {
					console.warn('[API] Token refresh failed, logging out:', {
						status: refreshResp.status,
						statusText: refreshResp.statusText
					});
					const { logout } = await import('$lib/stores/auth.svelte.ts');
					logout();
				}
			} else if (!isAuthEndpoint && !auth.refreshToken) {
				// No refresh token available, must log out
				const { logout } = await import('$lib/stores/auth.svelte.ts');
				logout();
			}
		} catch (error) {
			console.error('[API] Exception during token refresh:', error);
		}
		let msg = 'Session expired';
		try {
			msg = get(_)('errors.session_expired');
		} catch (error) {
			console.error('[API] Failed to get localized session expired message:', error);
		}
		throw new ApiError(msg, 401);
	}
	if (!response.ok) {
		let errorMessage = get(_)('errors.generic');
		try {
			const errorData = await response.json();
			errorMessage = errorData.error || errorData.message || errorData.detail || errorMessage;
		} catch (error) {
			console.error('[API] Failed to parse error response body:', error);
		}
		throw new ApiError(errorMessage, response.status);
	}
	return response;
}
async function request(path: string, options: RequestInit = {}, customFetch?: typeof fetch) {
	return requestWithBase(backendConfig.apiBaseUrl, path, options, customFetch);
}
export const api = {
	async get<T>(path: string, customFetch?: typeof fetch): Promise<T> {
		const response = await request(path, { method: 'GET' }, customFetch);
		return response.json();
	},
	async getBlob(path: string, customFetch?: typeof fetch): Promise<Blob> {
		const response = await request(path, { method: 'GET' }, customFetch);
		const contentType =
			response.headers.get('Content-Type') ||
			response.headers.get('content-type') ||
			'application/octet-stream';
		const data = await response.arrayBuffer();
		return new Blob([data], { type: contentType });
	},
	async post<T, Body = Record<string, unknown>>(
		path: string,
		data: Body,
		customFetch?: typeof fetch
	): Promise<T> {
		const response = await request(
			path,
			{
				method: 'POST',
				body: JSON.stringify(data)
			},
			customFetch
		);
		const text = await response.text();
		return text ? JSON.parse(text) : (null as T);
	},
	async patch<T, Body = Record<string, unknown>>(
		path: string,
		data: Body,
		customFetch?: typeof fetch
	): Promise<T> {
		const response = await request(
			path,
			{
				method: 'PATCH',
				body: JSON.stringify(data)
			},
			customFetch
		);
		const text = await response.text();
		return text ? JSON.parse(text) : (null as T);
	},
	async delete(path: string, customFetch?: typeof fetch): Promise<void> {
		await request(path, { method: 'DELETE' }, customFetch);
	},
	async postMultipart<T>(path: string, form: FormData, customFetch?: typeof fetch): Promise<T> {
		const response = await request(path, { method: 'POST', body: form }, customFetch);
		const text = await response.text();
		return text ? JSON.parse(text) : (null as T);
	},
	async getUserSettings(): Promise<UserSettings> {
		return this.get('/user/settings');
	},
	async updateUserSettings(settings: UserSettings) {
		return request('/user/settings', { method: 'PUT', body: JSON.stringify(settings) });
	},
	async deleteUserSettings() {
		return this.delete('/user/settings');
	}
};
export const apiPy = {
	async post<T, Body = Record<string, unknown>>(
		path: string,
		data: Body,
		customFetch?: typeof fetch
	): Promise<T> {
		const response = await requestWithBase(
			backendConfig.pyBaseUrl,
			path,
			{
				method: 'POST',
				body: JSON.stringify(data)
			},
			customFetch
		);
		const text = await response.text();
		return text ? JSON.parse(text) : (null as T);
	},
	async validateApiKey(provider: string, apiKey: string) {
		return this.post('/validate_api_key', { provider, api_key: apiKey });
	},
	async listModels(provider: string, apiKey: string) {
		return this.post('/list_models', { provider, api_key: apiKey });
	},
	async clearMemory() {
		const userId = auth.user?.id ? String(auth.user.id) : null;
		const response = await requestWithBase(backendConfig.pyBaseUrl, '/clear_memory', {
			method: 'POST',
			body: JSON.stringify({ user_id: userId })
		});
		return response.json();
	},
	async deleteSessionVectors(sessionUUID: string) {
		try {
			const userId = auth.user?.id ? String(auth.user.id) : null;
			const response = await requestWithBase(backendConfig.pyBaseUrl, '/delete_session_vectors', {
				method: 'POST',
				body: JSON.stringify({ user_id: userId, session_uuid: sessionUUID })
			});
			const text = await response.text();
			return text ? JSON.parse(text) : null;
		} catch {
			return null;
		}
	}
};
