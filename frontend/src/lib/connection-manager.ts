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

import { addSession, updateSession } from '$lib/stores/sessions.svelte.ts';
import { toast } from 'svelte-sonner';
import { auth } from './stores/auth.svelte';
import {
	streamStore,
	appendToStream,
	endStream,
	startStream
} from './stores/stream.svelte';
import { chatStore } from '$lib/stores/chat.svelte';
import { backendConfig } from '$lib/config';

// Type definitions for SSE events
interface StreamEvent {
	type: string;
	data: any;
}

interface ChatRequest {
	query: string;
	mode: string;
	session_uuid?: string;
	files?: any[];
	cached_files?: any[]; // Added cached_files to interface
	custom_instructions?: string;
	is_regeneration?: boolean;
	request_log_id_to_regen?: number;
	temp_id?: number;
	memory_enabled?: boolean;
	upload_id?: string;
}

let activeAbortController: AbortController | null = null;

/**
 * Parses a chunk of text from the stream and extracts SSE events.
 * Handles partial data handling across chunks if necessary (simplified here).
 */
function parseSSEChunk(chunk: string): StreamEvent[] {
	const events: StreamEvent[] = [];
	const lines = chunk.split('\n');

	for (const line of lines) {
		if (line.startsWith('data: ')) {
			try {
				const jsonStr = line.slice(6);
				const event = JSON.parse(jsonStr);
				events.push(event);
			} catch (e) {
				console.error('[Stream] Failed to parse SSE event:', e, line);
			}
		}
	}
	return events;
}

/**
 * Sends a chat message and handles the streaming response.
 */
async function sendMessage(req: ChatRequest) {
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
			} catch (e) {}

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

		await readStream(response.body);

	} catch (error: any) {
		if (error.name === 'AbortError') {
			console.log('[Stream] Request aborted by user');
			handleEvent({ type: 'done', data: 'Aborted by user' });
		} else {
			console.error('[Stream] Error:', error);
			handleEvent({ type: 'error', data: { message: error.message || 'Network error' } });
		}
	} finally {
		activeAbortController = null;
		// Ensure stream is marked as ended if not done explicitly
		if (streamStore.streaming) {
			endStream();
		}
	}
}

/**
 * Reconnects to an existing stream for a given session.
 */
async function reconnectStream(sessionUUID: string) {
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
                'Authorization': `Bearer ${token}`
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
                    query: "",
                    mode: "chat"
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
            } catch (e) {}

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

        await readStream(response.body);

    } catch (error: any) {
        if (error.name === 'AbortError') {
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
                    const event = JSON.parse(jsonStr) as StreamEvent;
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
            const event = JSON.parse(jsonStr) as StreamEvent;
            handleEvent(event);
        } catch (e) { /* ignore end of stream garbage */ }
    }
}

/**
 * Handles a single event from the stream.
 */
function handleEvent(event: StreamEvent) {
	const { type, data } = event;

	// Update stream store with current thought/header for UI
	if (type === 'thought_header') {
        // Just purely for UI feedback if needed
    }

	switch (type) {
		case 'session_created':
			if (data && data.uuid) {
				addSession(data);
			}
			break;

		case 'log_created':
			if (data && data.log_id) {
				streamStore.currentLogId = data.log_id;
                // Check if temp_id is provided and update chatStore
                if (data.temp_id) {
                    try {
                        // This updates the optimistic message ID to the real server log ID
                        chatStore.updateMessageLogId(data.temp_id, data.log_id);
                        chatStore.setLastUserMessageLogId(data.temp_id, data.log_id);
                    } catch (e) {
                        console.error('[Stream] Failed to update message log ID:', e);
                    }
                }
			}
			break;

		case 'thought_header':
			if (data && data.header) {
				streamStore.thoughtHeader = data.header;
			}
			break;

		case 'thought':
			const content = data?.thoughts || data?.content;
			const reasoning = data?.tool_reasoning;
			const header = data?.thoughts_header || streamStore.thoughtHeader || 'Thinking...';
			
			console.log(`[Stream] Thought received: ${header} (${content?.length || 0} chars)`);

			if (content || reasoning) {
				chatStore.appendThought({
					id: `thought_${Date.now()}_${Math.random().toString(36).substr(2, 5)}`,
					type: 'thought',
					header,
					content: content || '',
					reasoning: reasoning || '',
					confidence_score: data?.confidence_score,
					self_critique: data?.self_critique,
					plan_status: data?.plan_status
				});
			}
			break;

        case 'tool_call':
            if (data && data.tool_name) {
                const callId = data.call_id || data.tool_name;
				console.log(`[Stream] Tool call: ${data.tool_name} (${callId})`);
				chatStore.appendThought({
					id: callId,
					type: 'tool',
					toolName: data.tool_name,
					header: `Using ${data.tool_name}...`,
					status: 'running'
				});
                chatStore.updateToolStatus(callId, data.tool_name, 'running', undefined, `Using ${data.tool_name}...`);
            }
            break;

        case 'tool_output':
            if (data && data.tool_name) {
                const callId = data.call_id || data.tool_name;
				console.log(`[Stream] Tool output: ${data.tool_name} (${callId})`);
				const existingIndex = streamStore.thoughts.findIndex(t => t.id === callId);
				
				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status: 'completed',
						header: `${data.tool_name} completed`,
						content: data.output
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: data.tool_name,
						header: `${data.tool_name} completed`,
						status: 'completed',
						content: data.output
					});
				}
                chatStore.updateToolStatus(callId, data.tool_name, 'completed', undefined, `${data.tool_name} completed`);
            }
            break;

        case 'tool_progress':
            if (data && data.tool_name) {
                const callId = data.call_id || data.tool_name;
                const status = data.status || 'running';
				const existingIndex = streamStore.thoughts.findIndex(t => t.id === callId);

				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status: status as any,
						header: data.header || `Using ${data.tool_name}...`,
						progress: data.progress
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: data.tool_name,
						header: data.header || `Using ${data.tool_name}...`,
						status: status as any,
						progress: data.progress
					});
				}
                chatStore.updateToolStatus(callId, data.tool_name, status, data.progress, data.header);
            }
            break;

        case 'tool_error':
            if (data && data.tool_name) {
                const callId = data.call_id || data.tool_name;
				const existingIndex = streamStore.thoughts.findIndex(t => t.id === callId);

				if (existingIndex !== -1) {
					chatStore.updateThought(callId, {
						status: 'failed',
						header: `${data.tool_name} failed`,
						content: data.error
					});
				} else {
					chatStore.appendThought({
						id: callId,
						type: 'tool',
						toolName: data.tool_name,
						header: `${data.tool_name} failed`,
						status: 'failed',
						content: data.error
					});
				}
                chatStore.updateToolStatus(callId, data.tool_name, 'failed', undefined, `${data.tool_name} failed: ${data.error || 'Unknown error'}`);
            }
            break;

        case 'usage_update':
            // This is informational events for the UI
            break;

		case 'chunk':
			// Text chunk of the final answer
			if (typeof data === 'string') {
				appendToStream(data);
			} else if (data && data.chunk) {
                appendToStream(data.chunk);
            } else if (data && data.text) {
				appendToStream(data.text);
			}
			break;

        case 'session_title_updated':
            if (data && data.uuid && data.title) {
                chatStore.updateSessionTitle(data.uuid, data.title);
            }
            break;

		case 'done':
            // If data contains "No active stream", we shouldn't trigger endStream in a way that affects UI state negatively
            // But usually done means successful completion.
			endStream();
			break;

		case 'error':
			endStream();
			toast.error(data?.message || 'An error occurred');
			break;

		case 'text_stream_reset':
			chatStore.resetBuffer();
			break;

		case 'plan_updated':
			if (data) {
				chatStore.setPlan(data);
			}
			break;
	}
}

/**
 * Abort the current generation stream.
 */
function stopStream() {
	if (activeAbortController) {
		activeAbortController.abort();
		activeAbortController = null;
		endStream();
	}
}

export const connectionManager = {
	sendMessage,
    reconnectStream,
	stopStream,
    // Legacy stubs
    connect: () => {},
    disconnect: () => {},
    getState: () => ({ state: 'CONNECTED', isOnline: true }),
    initializeVisibilityHandling: () => {},
    destroyVisibilityHandling: () => {}
};
