import { addSession, bringSessionToTop, updateSession } from '$lib/stores/sessions.svelte.ts';
import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import { toast } from 'svelte-sonner';
import { browser } from '$app/environment';
import { page } from '$app/stores';
import type { Page } from '@sveltejs/kit';
import { connectWebSocket, type EgoWebSocket } from './websocket';
import { wsStore } from './stores/websocket.svelte';
import { auth, setAccessToken } from './stores/auth.svelte';
import {
	streamStore,
	appendToStream,
	endStream,
	setThoughtHeader,
	setLastUserMessageLogId,
	setStreamError,
	stopStreamAsCancelled
} from './stores/stream.svelte';
import { api } from '$lib/api';
import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
import type { ChatSession } from './types';
let wsInstance: EgoWebSocket | null = null;
let reconnectAttempts = 0;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
let pingInterval: ReturnType<typeof setInterval> | null = null;
let pongTimeout: ReturnType<typeof setTimeout> | null = null;
let intentionallyClosed = false;
let reconnectToastId: string | number | undefined = undefined;
let suppressReconnectToastsUntilOpen = false;
let isPageVisible = true;
let isOnline = typeof navigator !== 'undefined' ? navigator.onLine : true;
let triedRefreshThisCycle = false; 
let isConnecting = false; 
let visibilityInitialized = false; 
let watchdogInterval: ReturnType<typeof setInterval> | null = null;
let lastStreamActivityMs = 0;
const WATCHDOG_INTERVAL_MS = 10000; 
const STREAM_STALL_THRESHOLD_MS = 180000; 
const MAX_RECONNECT_ATTEMPTS_VISIBLE = 10; 
const MAX_RECONNECT_ATTEMPTS_HIDDEN = 3; 
const RECONNECT_MAX_DELAY_MS = 10000; 
const RECONNECT_PAUSE_WHEN_HIDDEN_MS = 30000; 
const PING_INTERVAL_MS = 25000; 
const PONG_TIMEOUT_MS = 28000; 
function handlePong() {
	if (pongTimeout) clearTimeout(pongTimeout);
}
function isChatRoute(): boolean {
	try {
		const p = get(page) as Page;
		const id = p.route?.id;
		return typeof id === 'string' && id.startsWith('/chat');
	} catch {
		return false;
	}
}
function markStreamActivity() {
	lastStreamActivityMs = Date.now();
}
function ensureWatchdogRunning() {
	if (watchdogInterval) return;
	watchdogInterval = setInterval(async () => {
		try {
			if (streamStore.isDone) return;
			if (!lastStreamActivityMs) {
				lastStreamActivityMs = Date.now();
				return;
			}
			const since = Date.now() - lastStreamActivityMs;
			if (since > STREAM_STALL_THRESHOLD_MS) {
				console.warn(`[WS] Stream watchdog: detected stall after ${since}ms. Attempting graceful recovery...`);
				const logId = streamStore.currentLogId;
				if (logId && !streamStore.isRecovering) {
					streamStore.isRecovering = true;
					try {
						const resp = await api.get<{ final_response?: string }>(`/logs/${logId}`);
						if (resp?.final_response) {
							console.log('[WS] Watchdog recovered final response via REST API');
							streamStore.recoveredFromRest = true;
							appendToStream(resp.final_response);
							endStream();
							lastStreamActivityMs = Date.now();
							streamStore.isRecovering = false;
							return;
						}
					} catch (error) {
						console.warn('[WS] Watchdog failed to fetch final response:', error);
					}
					streamStore.isRecovering = false;
				}
				lastStreamActivityMs = Date.now();
			}
		} catch {
		}
	}, WATCHDOG_INTERVAL_MS);
}
function scheduleReconnect() {
	if (intentionallyClosed || !isOnline) {
		return;
	}
	const maxAttempts = isPageVisible ? MAX_RECONNECT_ATTEMPTS_VISIBLE : MAX_RECONNECT_ATTEMPTS_HIDDEN;
	if (reconnectAttempts >= maxAttempts) {
		console.log(`[WS] Max reconnect attempts reached (${maxAttempts}). Stopping reconnection.`);
		if (reconnectToastId) {
			toast.dismiss(reconnectToastId);
			reconnectToastId = undefined;
		}
		if (!isPageVisible) {
			setTimeout(() => {
				reconnectAttempts = 0;
				if (!wsStore.connection && auth.accessToken && !intentionallyClosed) {
					scheduleReconnect();
				}
			}, RECONNECT_PAUSE_WHEN_HIDDEN_MS);
		}
		return;
	}
	const delay = Math.min(Math.pow(1.8, reconnectAttempts) * 1000, RECONNECT_MAX_DELAY_MS);
	reconnectAttempts++;
	console.log(
		`[WS] Connection lost. Reconnecting in ${Math.round(delay / 100) / 10}s... (attempt ${
			reconnectAttempts
		}/${maxAttempts})`
	);
	if (isChatRoute() && isPageVisible && !suppressReconnectToastsUntilOpen) {
		const message = get(_)('toasts.ws_reconnect_attempt', {
			values: { attempt: reconnectAttempts }
		});
		const hideAction = {
			label: get(_)('toasts.action_hide'),
			onClick: () => {
				if (reconnectToastId) toast.dismiss(reconnectToastId);
				suppressReconnectToastsUntilOpen = true;
				reconnectToastId = undefined;
			}
		} as const;
		if (reconnectToastId) {
			toast.loading(message, { id: reconnectToastId, action: hideAction });
		} else {
			reconnectToastId = toast.loading(message, { duration: Infinity, action: hideAction });
		}
	}
	if (reconnectTimeout) clearTimeout(reconnectTimeout);
	reconnectTimeout = setTimeout(connect, delay);
}
async function attemptRefreshIfNeeded() {
	if (triedRefreshThisCycle || !auth.refreshToken) return false;
	triedRefreshThisCycle = true;
	try {
		const res = await api.post<{ access_token: string }>('/auth/refresh', {
			refresh_token: auth.refreshToken
		});
		if (res?.access_token) {
			setAccessToken(res.access_token);
			if (reconnectTimeout) clearTimeout(reconnectTimeout);
			reconnectAttempts = 0;
			connect();
			return true;
		}
	} catch {
	}
	return false;
}
function connect() {
	try {
		if (maintenanceStore && maintenanceStore.isMaintenanceActive) {
			console.log('[WS] Maintenance active; skipping WS connect');
			return;
		}
	} catch {}
	if (wsInstance || isConnecting || !auth.accessToken) {
		return;
	}
	intentionallyClosed = false;
	isConnecting = true;
	console.log(`[WS] Attempting to connect... (Attempt ${reconnectAttempts + 1})`);
	wsStore.setConnection(null);
	if (reconnectTimeout) clearTimeout(reconnectTimeout);
	if (pingInterval) clearInterval(pingInterval);
	if (pongTimeout) clearTimeout(pongTimeout);
	if (!reconnectToastId && isChatRoute() && !suppressReconnectToastsUntilOpen) {
		reconnectToastId = toast.loading(get(_)('toasts.connecting_server'), {
			duration: Infinity,
			action: {
				label: get(_)('toasts.action_hide'),
				onClick: () => {
					if (reconnectToastId) toast.dismiss(reconnectToastId);
					suppressReconnectToastsUntilOpen = true;
					reconnectToastId = undefined;
				}
			}
		});
	}
	ensureWatchdogRunning();
	wsInstance = connectWebSocket(auth.accessToken, {
		onOpen: () => {
			console.log('[WS] Connection established successfully.');
			wsStore.setConnection(wsInstance);
			const hadReconnectAttempts = reconnectAttempts > 0;
			reconnectAttempts = 0;
			triedRefreshThisCycle = false; 
			isConnecting = false;
			markStreamActivity();
			if (reconnectToastId) {
				toast.dismiss(reconnectToastId);
				reconnectToastId = undefined;
			}
			suppressReconnectToastsUntilOpen = false;
			if (hadReconnectAttempts && isChatRoute() && isPageVisible) {
				toast.success(get(_)('toasts.ws_reconnected'), { duration: 2000 });
			}
			if (!streamStore.isDone && streamStore.currentLogId) {
				const logId = streamStore.currentLogId;
				streamStore.isRecovering = true;
				let attempts = 0;
				const MAX_POLL_ATTEMPTS = 6;
				const poll = async () => {
					attempts++;
					try {
						const resp = await api.get<{ final_response?: string }>(`/logs/${logId}`);
						if (resp?.final_response) {
							streamStore.recoveredFromRest = true;
							appendToStream(resp.final_response);
							endStream();
							streamStore.isRecovering = false;
							return;
						}
					} catch {}
					if (streamStore.isDone) return;
					if (attempts < MAX_POLL_ATTEMPTS) {
						setTimeout(poll, 1500);
					} else {
						console.warn(
							'[WS] Final response not available after reconnect; marking stream as cancelled.'
						);
						streamStore.isRecovering = false;
						stopStreamAsCancelled();
					}
				};
				setTimeout(poll, 600);
			} else if (!streamStore.isDone && !streamStore.currentLogId) {
				console.warn('[WS] Active stream without currentLogId on reconnect; deferring cancel and attempting recovery.');
				connectionManager.triggerPollForFinalIfNeeded();
			}
			const sendPing = () => {
				if (!wsInstance || !wsStore.connection) return;
				try {
					wsInstance.ping();
				} catch {}
				if (pongTimeout) clearTimeout(pongTimeout);
				pongTimeout = setTimeout(() => {
					console.error('[WS] Pong not received in time. Closing connection to trigger reconnect.');
					try {
						wsInstance?.close();
					} catch {}
				}, PONG_TIMEOUT_MS);
			};
			sendPing();
			pingInterval = setInterval(sendPing, PING_INTERVAL_MS);
		},
		onClose: () => {
			console.log('[WS] Connection closed.');
			wsStore.setConnection(null);
			wsInstance = null;
			isConnecting = false;
			if (pingInterval) clearInterval(pingInterval);
			if (pongTimeout) clearTimeout(pongTimeout);
			markStreamActivity();
			if (!streamStore.isDone && isChatRoute() && isPageVisible) {
				toast.warning(get(_)('toasts.ws_reconnecting'), { duration: 3000 });
			}
			if (!isOnline && isChatRoute() && isPageVisible) {
				if (!reconnectToastId) {
					reconnectToastId = toast.loading(get(_)('toasts.offline_waiting'), {
						duration: Infinity
					});
				}
			}
			attemptRefreshIfNeeded();
			scheduleReconnect();
		},
		onSessionCreated: (newSession: ChatSession) => {
			if (!newSession || !newSession.uuid) return;
			addSession(newSession);
			try {
				const p = get(page) as Page;
				const pathname: string = p.url?.pathname || '';
				const isOnNew = typeof pathname === 'string' && pathname.endsWith('/chat/new');
				if (isOnNew || !streamStore.sessionUUID) {
					streamStore.sessionUUID = newSession.uuid;
					streamStore.newlyCreatedSessionUUID = newSession.uuid;
				}
			} catch {
				if (!streamStore.sessionUUID) {
					streamStore.sessionUUID = newSession.uuid;
					streamStore.newlyCreatedSessionUUID = newSession.uuid;
				}
			}
			markStreamActivity();
		},
		onLogCreated: ({ log_id, temp_id }) => {
			console.log('[ConnectionManager] Received log_created event:', { log_id, temp_id });
			setLastUserMessageLogId(temp_id, log_id);
			markStreamActivity();
		},
		onThoughtHeader: (header: string) => {
			markStreamActivity();
			setThoughtHeader(header);
		},
		onChunk: (text: string) => {
			markStreamActivity();
			appendToStream(text);
		},
		onDone: async () => {
			markStreamActivity();
			const activeSessionUUID = streamStore.sessionUUID;
			if (activeSessionUUID) {
				bringSessionToTop(activeSessionUUID);
			}
			try {
				if (!streamStore.textStream && streamStore.currentLogId) {
					const logId = streamStore.currentLogId;
					const resp = await api.get<{ final_response?: string }>(`/logs/${logId}`);
					if (resp?.final_response) {
						appendToStream(resp.final_response);
					}
				}
			} catch {
			} finally {
				endStream();
			}
		},
		onError: (errorMsg) => {
			markStreamActivity();
			setStreamError(errorMsg);
			endStream();
			if (!reconnectToastId && isChatRoute() && isPageVisible) {
				toast.error(errorMsg, { duration: 4000 });
			}
			attemptRefreshIfNeeded();
		},
		onPong: handlePong,
		onSessionTitleUpdated: ({ uuid, title }) => {
			updateSession({ uuid, title });
		}
	});
}
function disconnect() {
	if (reconnectTimeout) clearTimeout(reconnectTimeout);
	if (pingInterval) clearInterval(pingInterval);
	if (pongTimeout) clearTimeout(pongTimeout);
	if (watchdogInterval) {
		clearInterval(watchdogInterval);
		watchdogInterval = null;
	}
	if (reconnectToastId) {
		toast.dismiss(reconnectToastId);
			reconnectToastId = undefined;
	}
	reconnectAttempts = 0;
	triedRefreshThisCycle = false;
	isConnecting = false;
	intentionallyClosed = true;
	if (wsInstance) {
		console.log('[WS] Disconnecting intentionally.');
		wsInstance.close();
		wsInstance = null;
		wsStore.setConnection(null);
	}
}
function handleVisibilityChange() {
	if (!browser) return;
	const wasVisible = isPageVisible;
	isPageVisible = document.visibilityState === 'visible';
	if (isPageVisible && !wasVisible) {
		console.log('[WS] Page became visible, resetting reconnect attempts');
		if (reconnectToastId) {
			toast.dismiss(reconnectToastId);
			reconnectToastId = undefined;
		}
		reconnectAttempts = 0;
		triedRefreshThisCycle = false;
		if (!wsStore.connection && auth.accessToken && !intentionallyClosed) {
			if (reconnectTimeout) clearTimeout(reconnectTimeout);
			connect();
		}
		if (!streamStore.isDone && streamStore.currentLogId && !streamStore.isRecovering) {
			console.log('[WS] Resuming active stream after page became visible');
			connectionManager.triggerPollForFinalIfNeeded();
		}
	} else if (!isPageVisible && wasVisible) {
		console.log('[WS] Page became hidden, dismissing reconnect toasts');
		if (reconnectToastId) {
			toast.dismiss(reconnectToastId);
			reconnectToastId = undefined;
		}
		if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS_VISIBLE) {
			reconnectAttempts = Math.min(reconnectAttempts, MAX_RECONNECT_ATTEMPTS_HIDDEN);
		}
	}
}
export const connectionManager = {
	connect,
	disconnect,
	triggerPollForFinalIfNeeded: () => {
		if (!streamStore.isDone && streamStore.currentLogId && !streamStore.isRecovering) {
			const logId = streamStore.currentLogId;
			streamStore.isRecovering = true;
			let attempts = 0;
			const MAX_POLL_ATTEMPTS = 6;
			const poll = async () => {
				attempts++;
				try {
					const resp = await api.get<{ final_response?: string }>(`/logs/${logId}`);
					if (resp?.final_response) {
						streamStore.recoveredFromRest = true;
						appendToStream(resp.final_response);
						endStream();
						streamStore.isRecovering = false;
						return;
					}
				} catch {}
				if (streamStore.isDone) return;
				if (attempts < MAX_POLL_ATTEMPTS) {
					setTimeout(poll, 1500);
				} else {
					console.warn(
						'[WS] Final response not available after resume polling; marking stream as cancelled.'
					);
					streamStore.isRecovering = false;
					stopStreamAsCancelled();
				}
			};
			setTimeout(poll, 400);
		}
	},
	initializeVisibilityHandling: () => {
		if (!browser) return;
		if (visibilityInitialized) return;
		document.addEventListener('visibilitychange', handleVisibilityChange);
		const onlineHandler = () => {
			isOnline = true;
			reconnectAttempts = 0;
			if (!wsStore.connection && auth.accessToken) {
				if (reconnectToastId) {
					toast.dismiss(reconnectToastId);
					reconnectToastId = undefined;
				}
				connect();
			}
		};
		const offlineHandler = () => {
			isOnline = false;
		};
		window.addEventListener('online', onlineHandler);
		window.addEventListener('offline', offlineHandler);
		window.__egoOnlineHandler = onlineHandler;
		window.__egoOfflineHandler = offlineHandler;
		visibilityInitialized = true;
	},
	destroyVisibilityHandling: () => {
		if (!browser) return;
		document.removeEventListener('visibilitychange', handleVisibilityChange);
		if (window.__egoOnlineHandler) window.removeEventListener('online', window.__egoOnlineHandler);
		if (window.__egoOfflineHandler)
			window.removeEventListener('offline', window.__egoOfflineHandler);
		visibilityInitialized = false;
	}
};