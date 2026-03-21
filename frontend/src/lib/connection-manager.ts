/**
 * Chat Stream Manager (formerly WebSocket Connection Manager)
 *
 * This refactored version replaces WebSocket logic with HTTP POST + SSE Streaming.
 * It manages the chat stream lifecycle, including:
 * - Sending messages via POST /api/chat/stream
 * - Reading the SSE response stream
 * - Handling events (thought, chunk, done, etc.)
 * - Aborting streams (Stop button)
 * - Updating UI stores (stream.svelte.ts, chat.svelte.ts)
 */

import { addSession } from '$lib/stores/sessions.svelte.ts';
import { toast } from 'svelte-sonner';
import { auth } from './stores/auth.svelte';
import { streamStore, appendToStream, endStream, startStream } from './stores/stream.svelte';
import { chatStore } from '$lib/stores/chat.svelte';
import { backendConfig } from '$lib/config';
import type { WsEvent, FilePayload, CachedFile } from '$lib/types';

interface ChatRequest {
	query: string;
	mode: string;
	session_uuid?: string | null;
	files?: (FilePayload | CachedFile)[];
	cached_files?: CachedFile[];
	custom_instructions?: string | null;
	is_regeneration?: boolean;
	request_log_id_to_regen?: number;
	temp_id?: number;
	memory_enabled?: boolean;
	upload_id?: string;
}

type AbortMode = 'user_stop' | 'replace' | 'reconnect' | 'disconnect' | null;

interface ActiveAbortMeta {
	controller: AbortController;
	mode: AbortMode;
	requestId: number;
}

let activeAbortMeta: ActiveAbortMeta | null = null;
let requestSequence = 0;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 3;
const RECONNECT_DELAY_MS = 2000;

function isActiveRequest(requestId: number) {
	return activeAbortMeta?.requestId === requestId;
}

function appendCommentary(text: string) {
	const normalized = text?.trim();
	if (!normalized) return;

	const lastThought = streamStore.thoughts[streamStore.thoughts.length - 1];
	if (
		lastThought &&
		lastThought.type === 'commentary' &&
		(lastThought.content || lastThought.header) === normalized
	) {
		return;
	}

	chatStore.appendThought({
		id: `commentary_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
		type: 'commentary',
		header: normalized,
		content: normalized
	});
}

/**
 * Sends a chat message and handles the streaming response.
 */
export async function sendMessage(req: ChatRequest) {
	if (!auth.accessToken) {
		toast.error('You must be logged in to send messages.');
		return;
	}

	if (activeAbortMeta) {
		activeAbortMeta.mode = 'replace';
		activeAbortMeta.controller.abort();
	}

	const requestId = ++requestSequence;
	const abortMeta: ActiveAbortMeta = {
		controller: new AbortController(),
		mode: null,
		requestId
	};
	activeAbortMeta = abortMeta;
	const signal = abortMeta.controller.signal;

	startStream(req.session_uuid || null);
	streamStore.streaming = true;

	try {
		const baseUrl = backendConfig.apiBaseUrl;

		const performRequest = async (token: string | null) => {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json'
			};
			if (token) {
				headers['Authorization'] = `Bearer ${token}`;
			}

			try {
				const { getStoredBypassToken } = await import('./api/maintenance');
				const bypassToken = getStoredBypassToken();
				if (bypassToken) {
					headers['X-Bypass-Token'] = bypassToken;
				}
			} catch (e) {
				console.error('[Stream] Error getting bypass token:', e);
			}

			return fetch(`${baseUrl}/chat/stream`, {
				method: 'POST',
				headers,
				body: JSON.stringify(req),
				signal
			});
		};

		let response = await performRequest(auth.accessToken);

		if (response.status === 401 && auth.refreshToken) {
			console.log('[Stream] 401 received, attempting token refresh...');

			const refreshHeaders: Record<string, string> = { 'Content-Type': 'application/json' };
			try {
				const { getStoredBypassToken } = await import('./api/maintenance');
				const bypassToken = getStoredBypassToken();
				if (bypassToken) {
					refreshHeaders['X-Bypass-Token'] = bypassToken;
				}
			} catch {
				// ignore
			}

			const refreshResp = await fetch(`${baseUrl}/auth/refresh`, {
				method: 'POST',
				headers: refreshHeaders,
				body: JSON.stringify({ refresh_token: auth.refreshToken })
			});

			if (refreshResp.ok) {
				const data = await refreshResp.json();
				const newToken = data?.access_token;
				if (newToken) {
					const { setAccessToken } = await import('./stores/auth.svelte');
					setAccessToken(newToken);
					response = await performRequest(newToken);
				}
			} else {
				console.warn('[Stream] Token refresh failed during stream start');
				const { logout } = await import('./stores/auth.svelte');
				logout();
				return;
			}
		}

		if (!response.ok) {
			const errorText = await response.text();
			throw new Error(`Server error ${response.status}: ${errorText}`);
		}

		if (!response.body) {
			throw new Error('Response body is null');
		}

		reconnectAttempts = 0;
		await readStream(response.body, requestId);
	} catch (error: unknown) {
		if (error instanceof Error && error.name === 'AbortError') {
			console.log('[Stream] Request aborted:', abortMeta.mode || 'unknown');
			return;
		}

		if (!isActiveRequest(requestId)) {
			return;
		}

		console.error('[Stream] Error:', error);
		const msg = error instanceof Error ? error.message : 'Network error';

		if (req.session_uuid && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
			reconnectAttempts++;
			console.log(
				`[Stream] Network error, attempting reconnect ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}...`
			);

			streamStore.isRecovering = true;
			toast.info(`Reconnecting... (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);
			await new Promise((resolve) => setTimeout(resolve, RECONNECT_DELAY_MS));

			try {
				await reconnectStream(req.session_uuid);
				streamStore.isRecovering = false;
				toast.success('Reconnected successfully');
				return;
			} catch (reconnectError) {
				console.error('[Stream] Reconnect failed:', reconnectError);
				if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
					streamStore.isRecovering = false;
					toast.error('Failed to reconnect after multiple attempts');
				}
			}
		}

		handleEvent({ type: 'error', data: { message: msg } } as WsEvent, requestId);
	} finally {
		if (!isActiveRequest(requestId)) {
			return;
		}

		const abortMode = abortMeta.mode;
		activeAbortMeta = null;
		streamStore.isRecovering = false;

		if (streamStore.streaming && !abortMode) {
			endStream();
		}
	}
}

/**
 * Reconnects to an existing stream for a given session.
 */
export async function reconnectStream(sessionUUID: string) {
	if (!auth.accessToken) return;

	if (activeAbortMeta) {
		activeAbortMeta.mode = 'reconnect';
		activeAbortMeta.controller.abort();
	}

	const requestId = ++requestSequence;
	const abortMeta: ActiveAbortMeta = {
		controller: new AbortController(),
		mode: null,
		requestId
	};
	activeAbortMeta = abortMeta;
	const signal = abortMeta.controller.signal;

	streamStore.streaming = true;
	streamStore.sessionUUID = sessionUUID;

	try {
		const baseUrl = backendConfig.apiBaseUrl;

		const performRequest = async (token: string | null) => {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${token}`
			};

			try {
				const { getStoredBypassToken } = await import('./api/maintenance');
				const bypassToken = getStoredBypassToken();
				if (bypassToken) {
					headers['X-Bypass-Token'] = bypassToken;
				}
			} catch (e) {
				console.error('[Stream] Error getting bypass token during reconnect:', e);
			}

			return fetch(`${baseUrl}/chat/stream`, {
				method: 'POST',
				headers,
				body: JSON.stringify({
					session_uuid: sessionUUID,
					query: '',
					mode: 'chat'
				}),
				signal
			});
		};

		let response = await performRequest(auth.accessToken);

		if (response.status === 401 && auth.refreshToken) {
			console.log('[Stream] 401 received during reconnect, attempting token refresh...');

			const refreshHeaders: Record<string, string> = { 'Content-Type': 'application/json' };
			try {
				const { getStoredBypassToken } = await import('./api/maintenance');
				const bypassToken = getStoredBypassToken();
				if (bypassToken) {
					refreshHeaders['X-Bypass-Token'] = bypassToken;
				}
			} catch {
				// ignore
			}

			const refreshResp = await fetch(`${baseUrl}/auth/refresh`, {
				method: 'POST',
				headers: refreshHeaders,
				body: JSON.stringify({ refresh_token: auth.refreshToken })
			});

			if (refreshResp.ok) {
				const data = await refreshResp.json();
				const newToken = data?.access_token;
				if (newToken) {
					const { setAccessToken } = await import('./stores/auth.svelte');
					setAccessToken(newToken);
					response = await performRequest(newToken);
				}
			} else {
				const { logout } = await import('./stores/auth.svelte');
				logout();
				return;
			}
		}

		if (!response.ok || !response.body) {
			streamStore.streaming = false;
			return;
		}

		reconnectAttempts = 0;
		await readStream(response.body, requestId);
	} catch (error: unknown) {
		if (error instanceof Error && error.name === 'AbortError') {
			return;
		}

		if (!isActiveRequest(requestId)) {
			return;
		}

		console.error('[Stream] Reconnect error:', error);
	} finally {
		if (!isActiveRequest(requestId)) {
			return;
		}

		if (!abortMeta.mode) {
			streamStore.streaming = false;
		}
		activeAbortMeta = null;
	}
}

async function readStream(readableStream: ReadableStream<Uint8Array>, requestId: number) {
	const reader = readableStream.getReader();
	const decoder = new TextDecoder();
	let buffer = '';

	while (true) {
		const { done, value } = await reader.read();

		if (done) {
			break;
		}

		const chunk = decoder.decode(value, { stream: true });
		buffer += chunk;

		const parts = buffer.split('\n\n');
		buffer = parts.pop() || '';

		for (const part of parts) {
			if (part.trim().startsWith('data: ')) {
				try {
					const jsonStr = part.trim().slice(6);
					const event = JSON.parse(jsonStr) as WsEvent;
					handleEvent(event, requestId);
				} catch (e) {
					console.error('[Stream] JSON parse error:', e, part);
				}
			}
		}
	}

	if (buffer.trim().startsWith('data: ')) {
		try {
			const jsonStr = buffer.trim().slice(6);
			const event = JSON.parse(jsonStr) as WsEvent;
			handleEvent(event, requestId);
		} catch {
			/* ignore end of stream garbage */
		}
	}
}

/**
 * Handles a single event from the stream.
 */
function handleEvent(event: WsEvent, requestId?: number) {
	if (requestId !== undefined && !isActiveRequest(requestId)) {
		return;
	}

	const { type } = event;

	switch (type) {
		case 'session_created':
			if (event.data && event.data.uuid) {
				addSession(event.data);
			}
			break;

		case 'log_created':
			if (event.data && event.data.log_id) {
				streamStore.currentLogId = event.data.log_id;
				if (event.data.temp_id) {
					try {
						chatStore.updateMessageLogId(event.data.temp_id, event.data.log_id);
						chatStore.setLastUserMessageLogId(event.data.temp_id, event.data.log_id);
					} catch (e) {
						console.error('[Stream] Failed to update message log ID:', e);
					}
				}
			}
			break;

		case 'thought_header':
			if (event.data && event.data.header) {
				streamStore.thoughtHeader = event.data.header;
			}
			break;

		case 'commentary':
			if (event.data?.text) {
				appendCommentary(event.data.text);
			}
			break;

		case 'thought': {
			const content = event.data.thoughts || event.data.content;
			const reasoning = event.data.tool_reasoning;
			const header = event.data.thoughts_header || streamStore.thoughtHeader || 'Thinking...';

			if (content || reasoning) {
				chatStore.appendThought({
					id: `thought_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
					type: 'thought',
					header,
					content: content || '',
					commentary: event.data.commentary,
					reasoning: reasoning || '',
					confidence_score: event.data.confidence_score,
					self_critique: event.data.self_critique,
					plan_status: event.data.plan_status
				});
			}
			break;
		}

		case 'tool_call': {
			if (event.data && event.data.tool_name) {
				const callId = event.data.call_id || event.data.tool_name;
				chatStore.appendThought({
					id: callId,
					type: 'tool',
					toolName: event.data.tool_name,
					header: `Using ${event.data.tool_name}...`,
					status: 'running'
				});
				chatStore.updateToolStatus(
					callId,
					event.data.tool_name,
					'running',
					undefined,
					`Using ${event.data.tool_name}...`
				);
			}
			break;
		}

		case 'tool_output': {
			if (event.data && event.data.tool_name) {
				const callId = event.data.call_id || event.data.tool_name;
				const existingIndex = streamStore.thoughts.findIndex((t) => t.id === callId);

				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status: 'completed',
						header: `${event.data.tool_name} completed`,
						content: event.data.output
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: event.data.tool_name,
						header: `${event.data.tool_name} completed`,
						status: 'completed',
						content: event.data.output
					});
				}

				chatStore.updateToolStatus(
					callId,
					event.data.tool_name,
					'completed',
					undefined,
					`${event.data.tool_name} completed`
				);
			}
			break;
		}

		case 'tool_progress': {
			if (event.data && event.data.tool_name) {
				const callId = event.data.call_id || event.data.tool_name;
				const status = 'running';
				const existingIndex = streamStore.thoughts.findIndex((t) => t.id === callId);

				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status,
						header: event.data.header || `Using ${event.data.tool_name}...`,
						progress: event.data.progress
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: event.data.tool_name,
						header: event.data.header || `Using ${event.data.tool_name}...`,
						status,
						progress: event.data.progress
					});
				}

				chatStore.updateToolStatus(
					callId,
					event.data.tool_name,
					status,
					event.data.progress,
					event.data.header
				);
			}
			break;
		}

		case 'tool_error': {
			if (event.data && event.data.tool_name) {
				const callId = event.data.call_id || event.data.tool_name;
				const existingIndex = streamStore.thoughts.findIndex((t) => t.id === callId);

				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status: 'failed',
						header: `${event.data.tool_name} failed`,
						content: event.data.error
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: event.data.tool_name,
						header: `${event.data.tool_name} failed`,
						status: 'failed',
						content: event.data.error
					});
				}

				chatStore.updateToolStatus(
					callId,
					event.data.tool_name,
					'failed',
					undefined,
					`${event.data.tool_name} failed: ${event.data.error || 'Unknown error'}`
				);
			}
			break;
		}

		case 'usage_update':
			break;

		case 'chunk':
			if (event.data && event.data.text) {
				appendToStream(event.data.text);
			}
			break;

		case 'session_title_updated':
			if (event.data && event.data.uuid && event.data.title) {
				chatStore.updateSessionTitle(event.data.uuid, event.data.title);
			}
			break;

		case 'done':
			endStream();
			break;

		case 'error':
			endStream();
			toast.error(event.data?.message || 'An error occurred');
			break;

		case 'text_stream_reset':
			chatStore.resetBuffer();
			break;

		case 'superego_round_start':
			if (event.data) {
				chatStore.startSuperEgoRound(event.data.round, event.data.title);
			}
			break;

		case 'superego_agent_start':
			if (event.data) {
				chatStore.startSuperEgoAgent(
					event.data.round,
					event.data.agent_name,
					event.data.agent_role
				);
			}
			break;

		case 'superego_agent_message':
			if (event.data) {
				chatStore.updateSuperEgoAgentMessage(
					event.data.round,
					event.data.agent_name,
					event.data.message
				);
			}
			break;

		case 'superego_agent_done':
			if (event.data) {
				chatStore.completeSuperEgoAgent(event.data.round, event.data.agent_name);
			}
			break;

		case 'superego_agent_error':
			if (event.data) {
				chatStore.errorSuperEgoAgent(event.data.round, event.data.agent_name, event.data.error);
			}
			break;

		case 'superego_round_done':
			if (event.data) {
				chatStore.completeSuperEgoRound(event.data.round);
			}
			break;

		case 'superego_debate_complete':
			if (event.data) {
				chatStore.completeSuperEgoDebate(event.data.summary);
			}
			break;

		case 'plan_updated':
			if (event.data) {
				chatStore.setPlan(event.data);
			}
			break;
	}
}

export function stopStream(reason: Exclude<AbortMode, null> = 'user_stop') {
	if (activeAbortMeta) {
		activeAbortMeta.mode = reason;
		activeAbortMeta.controller.abort();
	}
}

export function disconnect() {
	stopStream('disconnect');
	endStream();
}

export function initializeVisibilityHandling() {
	// No-op for HTTP streaming
}

export function destroyVisibilityHandling() {
	// No-op for HTTP streaming
}

export const connectionManager = {
	sendMessage,
	reconnectStream,
	stopStream,
	disconnect,
	initializeVisibilityHandling,
	destroyVisibilityHandling
};
