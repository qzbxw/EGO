import { api, ApiError } from '$lib/api';
import type { PageLoad } from './$types';
import type { ChatSession, HistoryLog, ChatMessage } from '$lib/types';
import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { setIsThinking } from '$lib/stores/ui.svelte.ts';
import { streamStore, tryHydrateFromStorage, resetStreamStore } from '$lib/stores/stream.svelte.ts';
import { getSessionById } from '$lib/stores/sessions.svelte.ts';
import { chatMessages } from '$lib/stores/chatMessages.svelte.ts';
import { auth, setAccessToken } from '$lib/stores/auth.svelte.ts';
export const load: PageLoad = async ({ params, fetch }: { params: { sessionID: string }, fetch: typeof window.fetch }) => {
	if (!browser) {
		return { session: null, messages: [] };
	}
	setIsThinking(false);
	const { sessionID } = params;
    if (sessionID === 'new' || !sessionID) {
        const existing = chatMessages.get?.();
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
		tryHydrateFromStorage(sessionID);
	}
	const uuid = String(sessionID).toLowerCase();
	const isUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(uuid);
	if (!isUUID) {
		throw error(404, 'Неверный идентификатор чата.');
	}
	try {
        let session = getSessionById(sessionID);
		if (!session) {
			const tryRefreshOnce = async () => {
				if (!auth.refreshToken) return false;
				try {
					const res = await api.post<{ access_token: string }>(`/auth/refresh`, { refresh_token: auth.refreshToken }, fetch);
					if (res?.access_token) {
						setAccessToken(res.access_token);
						return true;
					}
				} catch {}
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
			if (e instanceof ApiError && e.status === 401 && (await (async () => {
				if (!auth.refreshToken) return false;
				try {
					const res = await api.post<{ access_token: string }>(`/auth/refresh`, { refresh_token: auth.refreshToken }, fetch);
					if (res?.access_token) { setAccessToken(res.access_token); return true; }
				} catch {}
				return false;
			})())) {
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
                preview_url: a.mime_type?.startsWith('image/') ? `/attachments/${log.id}/${encodeURIComponent(a.file_name)}` : undefined
            }));
            messages.push({
                author: 'user',
                text: log.user_query,
                id: userMessageId,
                attachments: atts,
                logId: log.id
            });
            if (log.final_response) {
                messages.push({
                    author: 'ego',
                    text: log.final_response,
                    id: egoMessageId,
                    logId: log.id
                });
            } else {
                const isActiveOngoing = streamStore.currentLogId === log.id && !streamStore.isDone;
                const hasExistingText = streamStore.textStream && streamStore.textStream.trim().length > 0;
                if (isActiveOngoing) {
                    messages.push({
                        author: 'ego',
                        text: hasExistingText ? streamStore.textStream : '',
                        id: egoMessageId,
                        isThinking: !hasExistingText, 
                        logId: log.id
                    });
                    if (streamStore.sessionUUID !== sessionID) {
                        streamStore.sessionUUID = sessionID;
                    }
                    setIsThinking(true);
                } else if (hasExistingText) {
                    messages.push({
                        author: 'ego',
                        text: streamStore.textStream,
                        id: egoMessageId,
                        logId: log.id
                    });
                } else {
                }
            }
        }
        messages.sort((a, b) => a.id - b.id);
        try {
            const existing = chatMessages.get?.();
            if (existing && existing.sessionUUID === sessionID && Array.isArray(existing.messages) && existing.messages.length > 0) {
                const optByAuthorLog = new Map<string, ChatMessage>();
                for (const om of existing.messages) {
                    if (om.logId) optByAuthorLog.set(`${om.author}:${om.logId}`, om);
                }
                for (const sm of messages) {
                    if (sm.author === 'user' && sm.logId) {
                        const key = `user:${sm.logId}`;
                        const opt = optByAuthorLog.get(key);
                        if (opt && (!sm.attachments || sm.attachments.length === 0) && opt.attachments && opt.attachments.length > 0) {
                            sm.attachments = opt.attachments;
                        }
                    }
                }
                const serverLogIds = new Set(messages.map(m => m.logId).filter(Boolean) as number[]);
                const merged: ChatMessage[] = [...messages];
                for (const m of existing.messages) {
                    if (!m.logId || !serverLogIds.has(m.logId)) {
                        if (!merged.some(x => x.id === m.id)) {
                            merged.push(m);
                        }
                    }
                }
                merged.sort((a, b) => a.id - b.id);
                messages.length = 0; messages.push(...merged);
            }
        } catch {}
        		if (messages.length === 0 && streamStore.sessionUUID === sessionID && !streamStore.isDone) {
			messages.push({
				author: 'ego',
				text: streamStore.textStream || '',
				id: Date.now(),
				isThinking: !Boolean(streamStore.textStream),
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
			    throw error(404, 'Чат не найден.');
		    }
		    throw error(err.status || 500, err.message || 'Не удалось загрузить чат');
        } else if (err instanceof Error) {
            throw error(500, err.message || 'Не удалось загрузить чат');
        }
        throw error(500, 'Не удалось загрузить чат');
	}
};