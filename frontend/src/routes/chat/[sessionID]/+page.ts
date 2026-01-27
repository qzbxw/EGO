import { api, ApiError } from '$lib/api';
import type { PageLoad } from './$types';
import type {
	ChatSession,
	HistoryLog,
	ChatMessage,
	ThoughtStep,
	BackendThoughtStep
} from '$lib/types';
import { error, redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { setIsThinking } from '$lib/stores/ui.svelte.ts';
import { streamStore, tryHydrateFromStorage, resetStreamStore } from '$lib/stores/stream.svelte.ts';
import { getSessionById } from '$lib/stores/sessions.svelte.ts';
import { chatMessages } from '$lib/stores/chatMessages.svelte.ts';
import { auth, initAuthStore, setAccessToken } from '$lib/stores/auth.svelte.ts';

export const load: PageLoad = async ({
	params,
	fetch
}: {
	params: { sessionID: string };
	fetch: typeof window.fetch;
}) => {
	if (!browser) {
		return { session: null, messages: [] };
	}

	// Ensure auth store is hydrated from localStorage before using tokens
	initAuthStore();
	setIsThinking(false);
	const { sessionID } = params;

	// Optimize for newly created sessions on the client:
	// If we just created this session and have its data in memory, skip network fetch.
	// This avoids race conditions, duplicates, and "jerkiness".
	const existing = chatMessages.get?.();
	if (
		existing &&
		existing.newlyCreatedSessionUUID === sessionID &&
		existing.sessionUUID === sessionID &&
		existing.messages?.length > 0
	) {
		console.log('[PageLoad] Using in-memory state for newly created session:', sessionID);

		let session = getSessionById(sessionID) || null;

		// If we don't have the session object in the store yet, try to fetch it
		if (!session) {
			try {
				session = await api.get<ChatSession>(`/sessions/${sessionID}`, fetch);
			} catch (e) {
				console.warn('[PageLoad] Failed to fetch session details for new session:', e);
				// Fallback to minimal session object if fetch fails but we have messages
				// This prevents a full crash/404 if the API is slightly delayed
				session = {
					uuid: sessionID,
					title: 'New Chat',
					created_at: new Date().toISOString(),
					updated_at: new Date().toISOString(),
					user_id: 0, // Placeholder
					mode: 'default',
					custom_instructions: null
				} as ChatSession;
			}
		}

		return {
			session,
			messages: existing.messages
		};
	}

	if (sessionID === 'new' || !sessionID) {
		if (existing && existing.sessionUUID === 'new' && existing.messages?.length) {
			return {
				session: null,
				messages: existing.messages
			};
		}
		resetStreamStore();
		chatMessages.clear();
		return {
			session: null,
			messages: []
		};
	}
	if (streamStore.sessionUUID && streamStore.sessionUUID !== sessionID) {
		streamStore.sessionUUID = null;
		streamStore.currentLogId = null;
		streamStore.textStream = '';
		streamStore.thoughtHeader = '';
		streamStore.isDone = true;
	}
	if (sessionID) {
		tryHydrateFromStorage();
	}
	const uuid = String(sessionID).toLowerCase();
	const isUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(
		uuid
	);
	if (!isUUID) {
		throw error(404, 'Неверный идентификатор чата.');
	}
	try {
		let session = getSessionById(sessionID);
		if (!session) {
			const tryRefreshOnce = async () => {
				if (!auth.refreshToken) return false;
				try {
					const res = await api.post<{ access_token: string }>(
						`/auth/refresh`,
						{ refresh_token: auth.refreshToken },
						fetch
					);
					if (res?.access_token) {
						setAccessToken(res.access_token);
						return true;
					}
				} catch {
					console.warn('[PageLoad] Token refresh failed');
				}

				return false;
			};
			try {
				session = await api.get<ChatSession>(`/sessions/${sessionID}`, fetch);
			} catch (e: unknown) {
				if (e instanceof ApiError && e.status === 401 && (await tryRefreshOnce())) {
					session = await api.get<ChatSession>(`/sessions/${sessionID}`, fetch);
				} else {
					throw e;
				}
			}
		}
		let historyLogs: HistoryLog[];
		try {
			historyLogs = await api.get<HistoryLog[]>(`/sessions/${sessionID}/history`, fetch);
		} catch (e: unknown) {
			if (
				e instanceof ApiError &&
				e.status === 401 &&
				(await (async () => {
					if (!auth.refreshToken) return false;
					try {
						const res = await api.post<{ access_token: string }>(
							`/auth/refresh`,
							{ refresh_token: auth.refreshToken },
							fetch
						);
						if (res?.access_token) {
							setAccessToken(res.access_token);
							return true;
						}
					} catch {
						// ignore
					}
					return false;
				})())
			) {
				historyLogs = await api.get<HistoryLog[]>(`/sessions/${sessionID}/history`, fetch);
			} else {
				throw e;
			}
		}
		const messages: ChatMessage[] = [];
		for (const log of historyLogs) {
			const baseId = Number(log.id) * 2;
			const userMessageId = baseId - 1;
			const egoMessageId = baseId;
			const atts = (log.attachments || []).map((a) => ({
				...a,
				preview_url: a.mime_type?.startsWith('image/')
					? `/attachments/${log.id}/${encodeURIComponent(a.file_name)}`
					: undefined
			}));
			messages.push({
				author: 'user',
				text: log.user_query,
				id: userMessageId,
				attachments: atts,
				logId: log.id
			});
			if (log.final_response) {
				// Normalize thoughts if they exist
				const normalizedThoughts = log.thoughts?.map((t: BackendThoughtStep) => ({
					...t,
					header:
						t.header || t.thoughts_header || (t.type === 'thought' ? 'Thinking' : 'Using tool'),
					reasoning: t.reasoning || t.tool_reasoning,
					content: t.content || t.thoughts,
					toolName: t.toolName || t.tool_name
				})) as ThoughtStep[];

				messages.push({
					author: 'ego',
					text: log.final_response,
					id: egoMessageId,
					logId: log.id,
					thoughts: normalizedThoughts
				});
			} else {
				const isActiveOngoing = streamStore.currentLogId === log.id && !streamStore.isDone;
				const hasExistingText = streamStore.textStream && streamStore.textStream.trim().length > 0;
				if (isActiveOngoing) {
					messages.push({
						author: 'ego',
						text: hasExistingText ? streamStore.textStream : '',
						id: egoMessageId,
						isThinking: true,
						logId: log.id
					});
					if (streamStore.sessionUUID !== sessionID) {
						streamStore.sessionUUID = sessionID;
					}
					setIsThinking(true);
				} else if (hasExistingText) {
					// Stream has existing text but may still be ongoing — ensure thinking bubble stays visible
					messages.push({
						author: 'ego',
						text: '',
						id: egoMessageId,
						isThinking: true,
						logId: log.id
					});
					setIsThinking(true);
				}
			}
		}
		messages.sort((a, b) => a.id - b.id);
		try {
			const existing = chatMessages.get?.();
			if (
				existing &&
				existing.sessionUUID === sessionID &&
				Array.isArray(existing.messages) &&
				existing.messages.length > 0
			) {
				const optByAuthorLog = new Map<string, ChatMessage>();
				for (const om of existing.messages) {
					if (om.logId) optByAuthorLog.set(`${om.author}:${om.logId}`, om);
				}
				for (const sm of messages) {
					if (sm.author === 'user' && sm.logId) {
						const key = `user:${sm.logId}`;
						const opt = optByAuthorLog.get(key);
						if (
							opt &&
							(!sm.attachments || sm.attachments.length === 0) &&
							opt.attachments &&
							opt.attachments.length > 0
						) {
							sm.attachments = opt.attachments;
						}
					}
				}
				const serverLogIds = new Set(messages.map((m) => m.logId).filter(Boolean) as number[]);
				const merged: ChatMessage[] = [...messages];
				for (const m of existing.messages) {
					// Add if not in server logs AND not duplicate text/ID
					if (!m.logId || !serverLogIds.has(m.logId)) {
						// Deduplicate by ID and content for user messages to avoid identical doubles
						const isDuplicate = merged.some(
							(x) =>
								x.id === m.id || (x.author === m.author && x.text === m.text && x.author === 'user')
						);
						if (!isDuplicate) {
							merged.push(m);
						}
					}
				}
				merged.sort((a, b) => a.id - b.id);
				messages.length = 0;
				messages.push(...merged);
			}
		} catch {
			// ignore
		}
		if (messages.length === 0 && streamStore.sessionUUID === sessionID && !streamStore.isDone) {
			messages.push({
				author: 'ego',
				text: streamStore.textStream || '',
				id: Date.now(),
				isThinking: !streamStore.textStream,
				logId: streamStore.currentLogId ?? undefined
			});
			setIsThinking(true);
		}
		chatMessages.setMessages(sessionID, messages);
		return {
			session,
			messages
		};
	} catch (err: unknown) {
		if (err instanceof ApiError) {
			if (err.status === 404) {
				throw error(
					404,
					'Чат не найден или доступ запрещён. Возможно, вы вошли с другого аккаунта в другой вкладке.'
				);
			}
			if (err.status === 401) {
				// Если сервер вернул 401, считаем, что сессия устарела и просим перелогиниться
				throw redirect(307, '/login');
			}
			throw error(err.status || 500, err.message || 'Не удалось загрузить чат');
		} else if (err instanceof Error) {
			throw error(500, err.message || 'Не удалось загрузить чат');
		}
		throw error(500, 'Не удалось загрузить чат');
	}
};
