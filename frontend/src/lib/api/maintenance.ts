import { browser } from '$app/environment';
let BASE_URL = import.meta.env.VITE_PUBLIC_EGO_BACKEND_URL || '/api';
try {
  if (browser && location.port === '5173') {
    if (typeof BASE_URL === 'string' && BASE_URL.startsWith('http://localhost')) {
      BASE_URL = '/api';
    }
  }
} catch {}
export interface MaintenanceStatus {
	maintenance: boolean;
	message: string;
}
export async function checkMaintenanceStatus(): Promise<MaintenanceStatus> {
	if (!browser) {
		return { maintenance: false, message: '' };
	}
	try {
		let bypassToken = getStoredBypassToken();
		if (!bypassToken) {
			try {
				const params = new URLSearchParams(window.location.search);
				const urlToken = params.get('bypass_token') || params.get('token');
				if (urlToken) {
					bypassToken = urlToken;
					console.log('🔧 checkMaintenanceStatus: found bypass token in URL (not storing yet)');
				}
			} catch {}
		}
		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
		};
		if (bypassToken) {
			headers['X-Bypass-Token'] = bypassToken;
		}
		const response = await fetch(`${BASE_URL}/maintenance/status`, {
			method: 'GET',
			headers,
		});
		if (response.ok) {
			const data = await response.json();
			const maintenance = (typeof data.maintenance === 'boolean') ? data.maintenance : !!data.maintenance_enabled;
			return {
				maintenance,
				message: data.message || 'Sorry, Service is currently unavailable :(',
			};
		}
		if (response.status === 503 || response.status === 502) {
			try {
				const data = await response.json();
				if (data.maintenance) {
					return {
						maintenance: true,
						message: data.message || 'Sorry, Service is currently unavailable :(',
					};
				}
			} catch {
				return {
					maintenance: true,
					message: 'Sorry, Service is currently unavailable :(',
				};
			}
		}
		if (response.status === 403) {
			return { maintenance: false, message: '' };
		}
		if (response.status === 404 || response.status === 401) {
			try {
				const testHeaders: Record<string, string> = { 'Content-Type': 'application/json' };
				if (bypassToken) testHeaders['X-Bypass-Token'] = bypassToken;
				const testResponse = await fetch(`${BASE_URL}/me`, {
					method: 'GET',
					headers: testHeaders,
				});
				if (testResponse.status === 503) {
					try {
						const data = await testResponse.json();
						if (data.maintenance) {
							return {
								maintenance: true,
								message: data.message || 'Sorry, Service is currently unavailable :(',
							};
						}
					} catch {
						return {
							maintenance: true,
							message: 'Sorry, Service is currently unavailable :(',
						};
					}
				}
			} catch {
			}
		}
		return { maintenance: false, message: '' };
	} catch (error) {
		console.warn('Failed to check maintenance status (network/CORS). Treat as non-maintenance on client:', error);
		return { maintenance: false, message: '' };
	}
}
export function getBypassToken(): string | null {
    if (!browser) return null;
    let token: string | null = null;
    try {
        const raw = window.location.search || '';
        const m = raw.match(/[?&](?:bypass_token|token)=([^&#]+)/);
        if (m && m[1]) {
            token = decodeURIComponent(m[1]);
        }
    } catch {}
    if (!token) {
        const urlParams = new URLSearchParams(window.location.search);
        token = urlParams.get('bypass_token') || urlParams.get('token');
    }
    if (!token && window.location.hash) {
        const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : window.location.hash;
        const hashParams = new URLSearchParams(hash.startsWith('?') ? hash.slice(1) : hash);
        token = hashParams.get('bypass_token') || hashParams.get('token');
    }
    if (token) {
        console.log('🔧 Found bypass token in URL, storing immediately:', `${token.substring(0, 20)}...`);
        storeBypassToken(token);
    }
    return token;
}
export function storeBypassToken(token: string): void {
    if (!browser) return;
    localStorage.setItem('maintenance_bypass_token', token);
    try {
        document.cookie = `maintenance_bypass_token=${encodeURIComponent(token)}; Path=/; Max-Age=259200; SameSite=Lax`;
    } catch {}
}
export function getStoredBypassToken(): string | null {
    if (!browser) return null;
    const fromLs = localStorage.getItem('maintenance_bypass_token');
    if (fromLs) return fromLs;
    try {
        const match = document.cookie
            .split('; ')
            .find((row) => row.startsWith('maintenance_bypass_token='));
        if (match) {
            const value = decodeURIComponent(match.split('=')[1] || '');
            if (value) return value;
        }
    } catch {}
    return null;
}
export function clearBypassToken(): void {
    if (!browser) return;
    localStorage.removeItem('maintenance_bypass_token');
    try {
        document.cookie = 'maintenance_bypass_token=; Path=/; Max-Age=0; SameSite=Lax';
    } catch {}
}
export async function validateBypassToken(token: string): Promise<boolean> {
    if (!browser || !token) {
        console.log('🔧 validateBypassToken: invalid input', { browser, hasToken: !!token });
        return false;
    }
    try {
        const endpoint = `${BASE_URL}/maintenance/validate`;
        console.log('🔧 validateBypassToken: testing token', { 
            token: `${token.substring(0, 20)}...`, 
            endpoint
        });
        const response = await fetch(endpoint, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'X-Bypass-Token': token,
            },
        });
        console.log('🔧 validateBypassToken: response', { 
            status: response.status, 
            statusText: response.statusText,
            ok: response.ok
        });
        if (response.ok) {
            try {
                const data = await response.json();
                const isValid = !!data?.valid;
                console.log('🔧 validateBypassToken: result', { isValid, status: response.status });
                return isValid;
            } catch {
                console.log('🔧 validateBypassToken: invalid JSON response, treat as invalid');
                return false;
            }
        }
        console.log('🔧 validateBypassToken: non-OK status, treat as invalid');
        return false;
    } catch (error) {
        console.warn('🔧 validateBypassToken: network/CORS error, treat as invalid', error);
        return false;
    }
}