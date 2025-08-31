import { browser } from '$app/environment';
import { _ } from 'svelte-i18n';
import { get } from 'svelte/store';
import { goto } from '$app/navigation';
import { memoryStore } from '$lib/stores/memory.svelte.ts';
export type UserSettings = { llm_provider: string; llm_model?: string; api_key?: string };
export class ApiError extends Error {
	status: number;
	constructor(message: string, status: number) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}
let BASE_URL = import.meta.env.VITE_PUBLIC_EGO_BACKEND_URL || '/api';
let PY_BASE_URL = import.meta.env.VITE_PUBLIC_EGO_PY_BACKEND_URL || '/py';
try {
	if (browser) {
		const host = location.hostname;
		const isVercel = /\.vercel\.app$/i.test(host);
		if (isVercel && (BASE_URL === '/api' || BASE_URL === '' || BASE_URL === '/')) {
			BASE_URL = 'https://ego-backend.vercel.app';
		}
	}
} catch {
}
try {
	if (browser && location.port === '5173') {
		if (typeof BASE_URL === 'string' && BASE_URL.startsWith('http://localhost')) {
			BASE_URL = '/api';
		}
		if (typeof PY_BASE_URL === 'string' && PY_BASE_URL.startsWith('http://localhost')) {
			PY_BASE_URL = '/py';
		}
	}
} catch {
}
try {
	if (typeof BASE_URL === 'string' && /^https?:\/\//.test(BASE_URL)) {
		const trimmed = BASE_URL.replace(/\/+$/, '');
		const u = new URL(trimmed);
		const path = u.pathname.replace(/\/+$/, '');
		if (!path.endsWith('/api')) {
			BASE_URL = trimmed + '/api';
		}
	}
} catch {
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
		(window as any).__egoMemoryEnabled = memoryEnabled;
	} catch {
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
	try {
		const { getStoredBypassToken, storeBypassToken } = await import('$lib/api/maintenance');
		let bypassToken = getStoredBypassToken();
		if (!bypassToken && browser) {
			try {
				const params = new URLSearchParams(window.location.search);
				const urlToken = params.get('bypass_token') || params.get('token');
				if (urlToken) {
					bypassToken = urlToken;
					storeBypassToken(urlToken);
					console.log('🔧 Captured bypass token directly from URL for API request');
				}
			} catch {
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
	} catch {
	}
	const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData;
	if (!headers.has('Content-Type') && options.body && !isFormData) {
		headers.set('Content-Type', 'application/json');
	}
	const requestOptions: RequestInit = { ...options, headers };
	let response = await fetcher(`${base}${path}`, requestOptions);
	try {
		const ct =
			response.headers.get('Content-Type') || response.headers.get('content-type') || '';
		if (response.ok && /text\/html/i.test(ct)) {
			throw new ApiError('Unexpected HTML response from API', 502);
		}
	} catch (e) {
		if (e instanceof ApiError) throw e;
	}
	if (response.status === 503) {
		let hasBypass = false;
		try {
			const { getStoredBypassToken } = await import('$lib/api/maintenance');
			hasBypass = !!getStoredBypassToken();
		} catch {
		}
		if (!hasBypass) {
			console.log('🔧 API returned 503; redirecting to /maintenance (no bypass token)');
			try {
				if (browser && window.location.pathname !== '/maintenance') {
					const params = new URLSearchParams(window.location.search);
					const urlToken = params.get('bypass_token') || params.get('token');
					const target = urlToken
						? `/maintenance?bypass_token=${encodeURIComponent(urlToken)}`
						: '/maintenance';
					await goto(target);
				}
			} catch {
			}
		} else {
			console.log('🔧 503 received but bypass token present; skipping maintenance redirect');
		}
	}
	if (response.status === 401 && !isAuthEndpoint && base === BASE_URL) {
		try {
			const { auth, setAccessToken } = await import('$lib/stores/auth.svelte.ts');
			if (allowRetry401 && auth.refreshToken) {
				const refreshResp = await fetcher(`${BASE_URL}/auth/refresh`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ refresh_token: auth.refreshToken })
				});
				if (refreshResp.ok) {
					let newToken: string | null = null;
					try {
						const data = await refreshResp.json();
						newToken = data?.access_token || null;
					} catch {
					}
					if (newToken) {
						setAccessToken(newToken);
						headers.set('Authorization', `Bearer ${newToken}`);
						const retryOptions: RequestInit = { ...requestOptions, headers };
						const retry = await fetcher(`${base}${path}`, retryOptions);
						if (!retry.ok) {
							response = retry; 
						} else {
							return retry;
						}
					}
				}
			}
		} catch {
		}
		let msg = 'Session expired';
		try {
			msg = get(_)(
				'errors.session_expired'
			);
		} catch {
		}
		throw new ApiError(msg, 401);
	}
	if (!response.ok) {
		let errorMessage = get(_)(
			'errors.generic'
		);
		try {
			const errorData = await response.json();
			errorMessage = errorData.message || errorData.detail || errorMessage;
		} catch (e) {
		}
		throw new ApiError(errorMessage, response.status);
	}
	return response;
}
async function request(path: string, options: RequestInit = {}, customFetch?: typeof fetch) {
	return requestWithBase(BASE_URL, path, options, customFetch);
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
			PY_BASE_URL,
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
		let userId: string | null = null;
		try {
			const { auth } = await import('$lib/stores/auth.svelte.ts');
			userId = auth.user?.id ? String(auth.user.id) : null;
		} catch {
		}
		const response = await requestWithBase(
			PY_BASE_URL,
			'/clear_memory',
			{
				method: 'POST',
				body: JSON.stringify({ user_id: userId })
			}
		);
		const text = await response.text();
		return text ? JSON.parse(text) : null;
	},
	async deleteSessionVectors(sessionUUID: string) {
		try {
			let userId: string | null = null;
			try {
				const { auth } = await import('$lib/stores/auth.svelte.ts');
				userId = auth.user?.id ? String(auth.user.id) : null;
			} catch {
			}
			const response = await requestWithBase(
				PY_BASE_URL,
				'/delete_session_vectors',
				{
					method: 'POST',
					body: JSON.stringify({ user_id: userId, session_uuid: sessionUUID })
				}
			);
			const text = await response.text();
			return text ? JSON.parse(text) : null;
		} catch (e) {
			return null;
		}
	}
};