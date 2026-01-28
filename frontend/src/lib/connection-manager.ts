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

let activeAbortController: AbortController | null = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 3;
const RECONNECT_DELAY_MS = 2000;

/**
 * Sends a chat message and handles the streaming response.
 */
export async function sendMessage(req: ChatRequest) {
	if (!auth.accessToken) {
		toast.error('You must be logged in to send messages.');
		return;
	}

	// Cancel any existing stream
	if (activeAbortController) {
		activeAbortController.abort();
	}

	activeAbortController = new AbortController();
	const signal = activeAbortController.signal;

	// Only reset stream if we are starting a fresh request, but preserve sessionUUID if provided
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

			// Add maintenance bypass token if present
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

		// Handle 401 and attempt refresh
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
					// Retry the request with the new token
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

		// Reset reconnect attempts on successful connection
		reconnectAttempts = 0;

		await readStream(response.body);
	} catch (error: unknown) {
		if (error instanceof Error && error.name === 'AbortError') {
			console.log('[Stream] Request aborted by user');
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			handleEvent({ type: 'done', data: 'Aborted by user' } as any);
		} else {
			console.error('[Stream] Error:', error);
			const msg = error instanceof Error ? error.message : 'Network error';

			// Attempt to reconnect on network errors if we have a session UUID
			if (req.session_uuid && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
				reconnectAttempts++;
				console.log(
					`[Stream] Network error, attempting reconnect ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}...`
				);

				// Set recovering flag to prevent UI from clearing state
				streamStore.isRecovering = true;

				// Show reconnection toast
				toast.info(`Reconnecting... (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);

				// Wait before reconnecting
				await new Promise((resolve) => setTimeout(resolve, RECONNECT_DELAY_MS));

				// Try to reconnect to the same session
				try {
					await reconnectStream(req.session_uuid);
					streamStore.isRecovering = false;
					toast.success('Reconnected successfully');
					return; // Successfully reconnected, exit
				} catch (reconnectError) {
					console.error('[Stream] Reconnect failed:', reconnectError);
					// If this was the last attempt, fall through to show error
					if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
						streamStore.isRecovering = false;
						toast.error('Failed to reconnect after multiple attempts');
					}
				}
			}

			// If reconnection failed or not applicable, show error
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			handleEvent({ type: 'error', data: { message: msg } } as any);
		}
	} finally {
		activeAbortController = null;
		// Reset recovery flag
		streamStore.isRecovering = false;
		// Ensure stream is marked as ended if not done explicitly
		if (streamStore.streaming) {
			endStream();
		}
	}
}

/**
 * Reconnects to an existing stream for a given session.
 */
export async function reconnectStream(sessionUUID: string) {
	if (!auth.accessToken) return;

	if (activeAbortController) {
		activeAbortController.abort();
	}

	activeAbortController = new AbortController();
	const signal = activeAbortController.signal;

	// Do NOT call startStream() here because it resets the stream buffer.
	// We only want to listen for *new* events.
	streamStore.streaming = true;
	streamStore.sessionUUID = sessionUUID;

	try {
		const baseUrl = backendConfig.apiBaseUrl;

		const performRequest = async (token: string | null) => {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${token}`
			};

			// Add maintenance bypass token if present
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

		// Handle 401 and attempt refresh
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

		if (!response.ok) {
			// Likely 400 or no active stream. Just stop quietly.
			streamStore.streaming = false;
			return;
		}

		if (!response.body) {
			streamStore.streaming = false;
			return;
		}

		// Reset reconnect attempts on successful reconnection
		reconnectAttempts = 0;

		await readStream(response.body);
	} catch (error: unknown) {
		if (error instanceof Error && error.name === 'AbortError') {
			// User navigated away or stopped.
		} else {
			console.error('[Stream] Reconnect error:', error);
		}
	} finally {
		activeAbortController = null;
		streamStore.streaming = false;
	}
}

async function readStream(readableStream: ReadableStream<Uint8Array>) {
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
					handleEvent(event);
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
			handleEvent(event);
		} catch {
			/* ignore end of stream garbage */
		}
	}
}

/**
 * Handles a single event from the stream.
 */
function handleEvent(event: WsEvent) {
	const { type } = event;

	// Update stream store with current thought/header for UI
	if (type === 'thought_header') {
		// Just purely for UI feedback if needed
	}

	switch (type) {
		case 'session_created':
			if (event.data && event.data.uuid) {
				addSession(event.data);
			}
			break;

		case 'log_created':
			if (event.data && event.data.log_id) {
				streamStore.currentLogId = event.data.log_id;
				// Check if temp_id is provided and update chatStore
				if (event.data.temp_id) {
					try {
						// This updates the optimistic message ID to the real server log ID
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

		case 'thought': {
			const content = event.data.thoughts || event.data.content;
			const reasoning = event.data.tool_reasoning;
			const header = event.data.thoughts_header || streamStore.thoughtHeader || 'Thinking...';

			console.log(`[Stream] Thought received: ${header} (${content?.length || 0} chars)`);

			if (content || reasoning) {
				chatStore.appendThought({
					id: `thought_${Date.now()}_${Math.random().toString(36).substr(2, 5)}`,
					type: 'thought',
					header,
					content: content || '',
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
				console.log(`[Stream] Tool call: ${event.data.tool_name} (${callId})`);
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
				console.log(`[Stream] Tool output: ${event.data.tool_name} (${callId})`);
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
				// 'status' is not in WsToolProgressEvent data usually, it implies running.
				// But previous code used data.status. I'll assume 'running'.
				const status = 'running';
				const existingIndex = streamStore.thoughts.findIndex((t) => t.id === callId);

				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status: status,
						header: event.data.header || `Using ${event.data.tool_name}...`,
						progress: event.data.progress
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: event.data.tool_name,
						header: event.data.header || `Using ${event.data.tool_name}...`,
						status: status,
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
			// This is informational events for the UI
			break;

		case 'chunk':
			// Text chunk of the final answer
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

		case 'plan_updated':
			if (event.data) {
				chatStore.setPlan(event.data);
			}
			break;
	}
}

export function stopStream() {
	if (activeAbortController) {
		activeAbortController.abort();
		activeAbortController = null;
	}
	endStream();
}

export function disconnect() {
	stopStream();
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
