import type {
	ChatSession,
	WsEvent,
	WebSocketSendPayload,
	FilePayload,
	CachedFile
} from '$lib/types';
import { getStoredBypassToken } from '$lib/api/maintenance';
export interface WebSocketHandlers {
	onOpen: () => void;
	onClose: () => void;
	onSessionCreated: (session: ChatSession) => void;
	onLogCreated: (log: { log_id: number; temp_id: number }) => void;
	onThoughtHeader: (header: string) => void;
	onChunk: (text: string) => void;
	onDone: () => void;
	onError: (errorMsg: string) => void;
	onPong: () => void;
	onSessionTitleUpdated?: (update: { uuid: string; title: string }) => void;
}
export interface EgoWebSocket {
	readyState: number;
	send: (payload: WebSocketSendPayload) => void;
	ping: () => void;
	stop: () => void;
	close: () => void;
}
export function connectWebSocket(token: string, handlers: WebSocketHandlers): EgoWebSocket {
	if (!token) {
		handlers.onError('Вы не авторизованы. Обновите страницу.');
		return { readyState: WebSocket.CLOSED, send: () => {}, stop: () => {}, close: () => {}, ping: () => {} };
	}
	const configuredBackend = import.meta.env.VITE_PUBLIC_EGO_BACKEND_URL as string | undefined;
	function isAbsoluteHttp(url: string): boolean {
		return /^https?:\/\//.test(url);
	}
	const qs = new URLSearchParams({ token });
	try {
		const bypass = getStoredBypassToken();
		if (bypass) qs.set('bypass_token', bypass);
	} catch {}
	let wsUrl = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/ws?${qs.toString()}`;
	try {
		const cfg = (configuredBackend || '').trim();
		if (cfg && isAbsoluteHttp(cfg)) {
			const base = cfg.replace(/\/+$/, '');
			wsUrl = base.replace(/^http/, 'ws') + `/api/ws?${qs.toString()}`;
		} else {
			if (typeof window !== 'undefined' && location.port === '5173') {
				const host = location.hostname || 'localhost';
				wsUrl = `${
					location.protocol === 'https:' ? 'wss' : 'ws'
				}://${host}:8080/api/ws?${qs.toString()}`;
			}
		}
	} catch {}
	let ws: WebSocket | null = null;
	try {
		ws = new WebSocket(wsUrl);
		ws.onopen = () => {
			handlers.onOpen();
		};
		ws.onclose = (_event) => {
			handlers.onClose();
		};
		ws.onerror = (error) => {
			console.error('WebSocket error observed:', error);
			handlers.onError('Произошла ошибка соединения WebSocket.');
		};
		setupMessageHandler(ws, handlers);
	} catch (error) {
		console.error('Failed to create WebSocket:', error);
		handlers.onError('Не удалось установить WebSocket соединение.');
	}
	return {
		get readyState() {
			return ws ? ws.readyState : WebSocket.CLOSED;
		},
		send: (payload: WebSocketSendPayload) => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				const wrapped = { type: 'stream_request', ...payload } as const;
				ws.send(JSON.stringify(wrapped));
			} else {
				console.warn('WebSocket is not connected. Message not sent.');
			}
		},
		ping: () => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: 'ping' }));
			}
		},
		stop: () => {
			if (ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: 'stop' }));
			}
		},
		close: () => {
			if (ws) {
				ws.close(1000, 'Manual close');
			}
		}
	};
}
function setupMessageHandler(websocket: WebSocket, handlers: WebSocketHandlers) {
	websocket.onmessage = (event) => {
		if (typeof event.data !== 'string') return;
		try {
			const wsEvent = JSON.parse(event.data) as WsEvent;
			switch (wsEvent.type) {
				case 'error':
					handlers.onError(wsEvent.data.message || 'Произошла неизвестная ошибка на сервере');
					break;
				case 'pong':
					handlers.onPong();
					break;
				case 'session_created':
					handlers.onSessionCreated(wsEvent.data);
					break;
				case 'log_created':
					handlers.onLogCreated(wsEvent.data);
					break;
				case 'thought_header':
					console.log('[WebSocket] Received thought_header event:', wsEvent.data);
					handlers.onThoughtHeader(wsEvent.data.header);
					break;
				case 'tool_progress':
					console.log('[WebSocket] Received tool_progress event:', wsEvent.data);
					handlers.onThoughtHeader(wsEvent.data.header);
					break;
				case 'chunk':
					handlers.onChunk(wsEvent.data.text);
					break;
				case 'session_title_updated':
					if (handlers.onSessionTitleUpdated) {
						handlers.onSessionTitleUpdated(wsEvent.data);
					}
					break;
				case 'usage_update':
					break;
				case 'done':
					handlers.onDone();
					break;
				default:
					console.debug('[WS] Unhandled event type:', (wsEvent as any)?.type);
					break;
			}
		} catch (error) {
			console.error('Failed to parse WebSocket message:', event.data, error);
			handlers.onError('Ошибка обработки ответа от сервера.');
		}
	};
}