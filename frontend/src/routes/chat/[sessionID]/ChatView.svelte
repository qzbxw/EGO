<script lang="ts">
	import type { PageData } from './$types';
	import type { ChatMessage, ChatSession, FilePayload } from '$lib/types';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte.ts';
	import { uiStore, setShowSettingsModal } from '$lib/stores/ui.svelte.ts';
	import { sessions, addSession, updateSession, getSessionById } from '$lib/stores/sessions.svelte.ts';
	import { chatMessages as egoChatMessages } from '$lib/stores/chatMessages.svelte.ts';
	import { cachedFiles } from '$lib/stores/cachedFiles.svelte.ts';
	import { api } from '$lib/api';
	import { wsStore } from '$lib/stores/websocket.svelte.ts';
	import {
		streamStore,
		startStream,
		stopStreamAsCancelled,
		consumeStreamResult,
		consumeLastUserMessageLogId,
		resetStreamStore,
		tryHydrateFromStorage
	} from '$lib/stores/stream.svelte.ts';
	import { newlyCreatedSessionUUIDStore } from '$lib/stores/stream.svelte.ts';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { inFlightStore } from '$lib/stores/inflight.svelte.ts';
	import { connectionManager } from '$lib/connection-manager';
	import { compressImages, shouldCompress } from '$lib/utils/imageCompression';
	import Modal from '$lib/components/Modal.svelte';
	import UserSettingsModal from '$lib/components/UserSettingsModal.svelte';
	import ChatHeader from '$lib/components/chat/ChatHeader.svelte';
	import MessageList from '$lib/components/chat/MessageList.svelte';
	import ChatInput from '$lib/components/chat/ChatInput.svelte';
	import SessionCreationOverlay from '$lib/components/chat/SessionCreationOverlay.svelte';
	let { data } = $props<{ data: PageData }>();
	type ChatMode = 'default' | 'deeper' | 'research' | 'agent';
	let customInstructionsInput = $state(data.session?.custom_instructions || '');
	let isNewSession = $state(!data.session);
	let chatMode = $state<ChatMode>('default');
	let currentInput = $state('');
	let attachedFiles = $state<File[]>([]);
	let userHasScrolledUp = $state(false);
	let messageListComponent: MessageList | undefined = $state();
	function scrollToBottom(behavior: 'smooth' | 'auto' = 'smooth') {
		messageListComponent?.scrollToBottom(behavior);
		userHasScrolledUp = false;
	}
	function isNearBottom(threshold = 200) {
		return messageListComponent?.isNearBottom(threshold) ?? true;
	}
	function handleScroll() {
		const isScrolledToBottom = isNearBottom(200);
		userHasScrolledUp = !isScrolledToBottom;
	}
	let editingLogId = $state<number | null>(null);
	let editingText = $state('');
	let editingTextarea: HTMLTextAreaElement | null = null;
	let isMobile = $state(false);
	let sending = $state(false);
	let showCreationOverlay = $derived(($page.params?.sessionID === 'new') && sending && !streamStore.isDone);
	let currentSession = $state(
		getSessionById(($page && $page.params && $page.params.sessionID) ? $page.params.sessionID : '') || data.session
	);
	let messages = $state<ChatMessage[]>(data.messages || []);
	let _initializedForRouteUUID = $state<string | null>($page.params?.sessionID ?? null);
	$effect(() => {
		const routeUUID = $page.params?.sessionID;
		try {
			const chatStore = $egoChatMessages;
			if (chatStore && chatStore.sessionUUID === routeUUID && chatStore.messages?.length) {
				messages = chatStore.messages;
			} else {
				messages = data.messages || [];
			}
		} catch {
			messages = data.messages || [];
		}
	});
	$effect(() => {
		const routeUUID = $page.params?.sessionID;
		const sessionsList = $sessions;
		if (routeUUID && routeUUID !== 'new' && sessionsList) {
			const found = sessionsList.find(s => s.uuid === routeUUID);
			if (found) currentSession = found;
		}
	});
	function getActiveSessionUUID(): string {
		const routeUUID = $page.params?.sessionID;
		if (routeUUID && routeUUID !== 'new') return routeUUID;
		if (streamStore.sessionUUID) return streamStore.sessionUUID;
		if (currentSession?.uuid) return currentSession.uuid;
		return 'new';
	}
	let lastMessageLogId = $state<number | null>(null);
	$effect(() => {
		const withLog = [...messages].reverse().find((m) => m.logId);
		const newLogId = withLog ? withLog.logId : null;
		if (newLogId !== lastMessageLogId) {
			lastMessageLogId = newLogId;
			console.log('[ChatView] lastMessageLogId updated to:', lastMessageLogId);
		}
	});
	$effect(() => {
		const routeUUID = $page.params?.sessionID ?? null;
		if (_initializedForRouteUUID === routeUUID) return;
		_initializedForRouteUUID = routeUUID;
		customInstructionsInput = data.session?.custom_instructions || '';
		isNewSession = !data.session;
		if (isNewSession) {
			try { sessionStorage.removeItem('pending-custom-instructions'); } catch {}
		}
		if (routeUUID === 'new') {
			try {
				streamStore.sessionUUID = null;
				streamStore.currentLogId = null;
			} catch {}
		}
		if (editingLogId !== null || editingText !== '') cancelEditing();
		const inFlight = inFlightStore.get();
		if (inFlight && (inFlight.sessionUUID === 'new' || inFlight.sessionUUID === routeUUID)) {
			inFlightStore.clear();
		}
		scrollToBottom('auto');
		if (routeUUID && routeUUID !== 'new') {
			const hydrated = tryHydrateFromStorage(routeUUID);
			if (hydrated) {
				const hasThinking = messages.some((m) => m.author === 'ego' && m.isThinking);
				if (!hasThinking) {
					const egoThinkingMessage: ChatMessage = {
						author: 'ego',
						id: Date.now(),
						text: '',
						isThinking: true,
						logId: streamStore.currentLogId || undefined
					};
					egoChatMessages.addMessage(routeUUID, egoThinkingMessage);
				}
				connectionManager.triggerPollForFinalIfNeeded();
				scrollToBottom('auto');
			}
		}
	});
	$effect(() => {
		const newUUID = $newlyCreatedSessionUUIDStore;
		if (!newUUID) return;
		const currentPath = $page.url.pathname;
		if (!currentPath.endsWith(`/chat/${newUUID}`)) {
			try {
				if (messages && messages.length > 0) {
					egoChatMessages.setMessages(newUUID, messages.slice());
				}
			} catch {}
			newlyCreatedSessionUUIDStore.set(null);
			goto(`/chat/${newUUID}`, { replaceState: true, invalidateAll: false });
		} else {
			newlyCreatedSessionUUIDStore.set(null);
		}
	});
	$effect(() => {
		const mediaQuery = window.matchMedia('(max-width: 768px)');
		const update = (e: MediaQueryList | MediaQueryListEvent) => (isMobile = e.matches);
		mediaQuery.addEventListener('change', update);
		update(mediaQuery);
		return () => mediaQuery.removeEventListener('change', update);
	});
	let isConnecting = $derived(!wsStore.connection);
	$effect(() => {
		if (streamStore.textStream) {
			if (!userHasScrolledUp || isNearBottom(160)) {
				scrollToBottom('auto');
			}
		}
	});
	$effect(() => {
		const lm = consumeLastUserMessageLogId();
		if (lm) {
			console.log('[ChatView] Received logId update:', lm);
			const sid = getActiveSessionUUID();
			try {
				console.log(`[ChatView] Attempting to update messages for session: ${sid}`);
				console.log(`[ChatView] Looking for temp_ids: ${lm.temp_id} and ${lm.temp_id + 1}`);
				console.log(`[ChatView] Current messages:`, messages.map(m => ({ id: m.id, author: m.author, logId: m.logId })));
				egoChatMessages.updateMessage(sid, lm.temp_id, { logId: lm.log_id });
				egoChatMessages.updateMessage(sid, lm.temp_id + 1, { logId: lm.log_id });
				lastMessageLogId = lm.log_id;
				console.log('[ChatView] LogID update completed, lastMessageLogId:', lastMessageLogId);
			} catch (e) {
				console.error('[ChatView] Error updating messages with logId:', e);
			}
		}
	});
	$effect(() => {
		const _done = streamStore.isDone;
		const _text = streamStore.textStream;
		const res = consumeStreamResult();
		if (!res) return;

		let sid = res.sessionUUID || getActiveSessionUUID();
		const storeState = egoChatMessages.get?.();
		if (storeState && storeState.sessionUUID) {
			sid = storeState.sessionUUID;
		}
		if (!sid) return;

		let target = [...messages].reverse().find((m) => m.author === 'ego' && m.isThinking) || null;
		if (!target) {
			const byLog = streamStore.currentLogId
				? [...messages].reverse().find((m) => m.author === 'ego' && m.logId === streamStore.currentLogId)
				: null;
			target = byLog || [...messages].reverse().find((m) => m.author === 'ego') || null;
		}
		if (target) {
			try {
				const finalText = (res.text || target.text || '').trim();
				const shouldFinalize = _done || (res.cancelled || false);
				console.log('[ChatView] Finalizing message:', {
					targetId: target.id,
					finalText: finalText.substring(0, 50) + '...',
					shouldFinalize,
					willSetThinking: !shouldFinalize,
					streamIsDone: _done
				});

				egoChatMessages.updateMessage(sid, target.id, {
					text: finalText,
					isThinking: !shouldFinalize,
					isCancelled: res.cancelled || false
				});

				const messageIndex = messages.findIndex((m) => m.id === target.id);
				if (messageIndex !== -1) {
					messages[messageIndex] = {
						...messages[messageIndex],
						text: finalText,
						isThinking: !shouldFinalize,
						isCancelled: res.cancelled || false
					};
					messages = [...messages];
				}
			} catch {}
		}
	});
	async function ensureWsReady(timeoutMs = 3000) {
		const start = Date.now();
		while (!wsStore.connection || wsStore.connection.readyState !== WebSocket.OPEN) {
			await new Promise((r) => setTimeout(r, 50));
			if (Date.now() - start > timeoutMs) break;
		}
		return !!wsStore.connection && wsStore.connection.readyState === WebSocket.OPEN;
	}
	async function sendMessage() {
		if (sending || (!currentInput.trim() && attachedFiles.length === 0)) return;
		sending = true;
		
		let tempAttachedFiles = attachedFiles;
		try {
			const filesToCompress = tempAttachedFiles.filter(shouldCompress);
			const filesToKeep = tempAttachedFiles.filter(f => !shouldCompress(f));
			
			if (filesToCompress.length > 0) {
				console.log(`[ChatView] Compressing ${filesToCompress.length} images...`);
				const compressed = await compressImages(filesToCompress);
				tempAttachedFiles = [...filesToKeep, ...compressed];
			}
		} catch (error) {
			console.error('[ChatView] Image compression failed:', error);
		}
		
		const tempCurrentInput = currentInput;
		const isNew = isNewSession;
		const tempId = Date.now();
		const optimisticUserMessage: ChatMessage = {
			author: 'user',
			id: tempId,
			text: tempCurrentInput,
			attachments: tempAttachedFiles.map(file => ({
				file_name: file.name,
				mime_type: file.type,
				preview_url: URL.createObjectURL(file)
			}))
		};
		const optimisticEgoMessage: ChatMessage = {
			author: 'ego',
			id: tempId + 1,
			text: '',
			isThinking: true
		};
		let activeSessionUUID = getActiveSessionUUID();
		egoChatMessages.addOptimisticMessages(activeSessionUUID, [optimisticUserMessage, optimisticEgoMessage]);
		try {
			messages = [...messages, optimisticUserMessage, optimisticEgoMessage];
		} catch {}
		scrollToBottom('smooth');
		if (isNew) {
			inFlightStore.set('new', tempCurrentInput, tempAttachedFiles);
			currentInput = '';
			attachedFiles = [];
		} else {
			currentInput = '';
			attachedFiles = [];
		}
		try {
			const filePayloads: FilePayload[] = await Promise.all(
				tempAttachedFiles.map(async (file) => {
					const dataUrl = await new Promise<string>((resolve, reject) => {
						const reader = new FileReader();
						reader.onload = (e) => resolve((e.target?.result as string) || '');
						reader.onerror = (e) => reject(e);
						reader.readAsDataURL(file);
					});
					let inferredMime = '';
					try {
						const m = dataUrl.match(/^data:([^;]+);base64,/i);
						if (m && m[1]) inferredMime = m[1];
					} catch {}
					const mime = (file.type && file.type.length > 0) ? file.type : (inferredMime || 'image/png');
					let fname = (file.name && file.name.length > 0) ? file.name : '';
					if (!fname) {
						const extFromMime = (() => {
							const part = (mime.split('/')[1] || '').toLowerCase();
							if (part) return part;
							return 'png';
						})();
						fname = `pasted-image-${Date.now()}.${extFromMime}`;
					}
					const base64 = dataUrl.startsWith('data:') ? (dataUrl.split(',')[1] || '') : dataUrl;
					return {
						base64_data: base64,
						mime_type: mime,
						file_name: fname
					} satisfies FilePayload;
				})
			);
			let pendingNewUUID: string | null = null;
			if (isNew && (!activeSessionUUID || activeSessionUUID === 'new')) {
				try {
					const newSession = await api.post<ChatSession>('/sessions', {
						custom_instructions: customInstructionsInput || undefined
					});
					if (newSession && newSession.uuid) {
						addSession(newSession);
						activeSessionUUID = newSession.uuid;
						streamStore.sessionUUID = newSession.uuid;
						pendingNewUUID = newSession.uuid;
					}
				} catch (e) {
					console.warn('[sendMessage] Pre-create session failed; falling back to WS session_created', e);
				}
			}
			startStream(activeSessionUUID && activeSessionUUID !== 'new' ? activeSessionUUID : null);
			if (pendingNewUUID) {
				try {
					if (messages && messages.length > 0) {
						egoChatMessages.setMessages(pendingNewUUID, messages.slice());
					}
				} catch {}
				newlyCreatedSessionUUIDStore.set(pendingNewUUID);
				goto(`/chat/${pendingNewUUID}`, { replaceState: true, invalidateAll: false });
			}
			const currentCachedFiles = !isNew ? cachedFiles.get() : null;
			const sessionCachedFiles = (currentCachedFiles?.sessionUUID === activeSessionUUID) ? currentCachedFiles.files : [];
			const payload: any = {
				query: tempCurrentInput,
				mode: chatMode,
				session_uuid: activeSessionUUID && activeSessionUUID !== 'new' ? activeSessionUUID : null,
				files: filePayloads,
				cached_files: sessionCachedFiles,
				custom_instructions: (!activeSessionUUID || activeSessionUUID === 'new') ? customInstructionsInput : undefined,
				is_regeneration: false,
				temp_id: tempId,
				memory_enabled: preferencesStore.memoryEnabled
			};
			if (auth.user?.id) {
				payload.user_id = String(auth.user.id);
			}
			const ready = await ensureWsReady(3000);
			if (!ready) {
				throw new Error('WebSocket is not connected');
			}
			wsStore.connection?.send(payload);
			if (filePayloads.length > 0 && !isNew) {
				const imagesToCache = filePayloads
					.filter(fp => fp.mime_type.startsWith('image/'))
					.map(fp => ({
						file_name: fp.file_name,
						mime_type: fp.mime_type,
						uri: `data:${fp.mime_type};base64,${fp.base64_data}`
					}));
				if (imagesToCache.length > 0) {
					cachedFiles.addFiles(activeSessionUUID, imagesToCache);
				}
			}
		} catch (error: unknown) {
			console.error('Error sending message:', error);
			toast.error($_('errors.send_failed'));
			egoChatMessages.removeMessage(activeSessionUUID, optimisticUserMessage.id);
			egoChatMessages.removeMessage(activeSessionUUID, optimisticEgoMessage.id);
			currentInput = tempCurrentInput;
			attachedFiles = tempAttachedFiles;
		} finally {
			sending = false;
		}
	}
	function regenerate(message: ChatMessage) {
		if (editingLogId !== null) return;
		if (!streamStore.isDone) return;
		const egoWs = wsStore.connection;
		if (!egoWs) {
			toast.error($_('chat.ws_not_connected'));
			return;
		}
		let targetLogId: number | null = null;
		let userMessageToRegen: ChatMessage | null = null;
		if (message.logId) {
			targetLogId = message.logId;
			userMessageToRegen = messages.find((m) => m.logId === targetLogId && m.author === 'user') || null;
		} else if (message.isThinking && streamStore.currentLogId) {
			targetLogId = streamStore.currentLogId;
			const thinkingIndex = messages.findIndex((m) => m.id === message.id);
			if (thinkingIndex > 0) {
				userMessageToRegen = messages[thinkingIndex - 1];
			}
		} else if (!message.isThinking) {
			const lastUserWithLog = [...messages].reverse().find((m) => m.author === 'user' && m.logId);
			if (lastUserWithLog) {
				targetLogId = lastUserWithLog.logId!;
				userMessageToRegen = lastUserWithLog;
			}
		}
		if (!targetLogId || !userMessageToRegen) {
			toast.error($_('chat.regen_target_not_found'));
			return;
		}
		if (message.isThinking) {
			stopGeneration();
		}
		const userMessageIndex = messages.findIndex((m) => m.id === userMessageToRegen!.id);
		if (userMessageIndex === -1) return;
		const messagesToRemove = messages.slice(userMessageIndex + 1);
		for (const msg of messagesToRemove) {
			const sid = getActiveSessionUUID();
			egoChatMessages.removeMessage(sid, msg.id);
		}
		resetStreamStore();
		const egoThinkingMessage: ChatMessage = {
			author: 'ego',
			id: Date.now(),
			text: '',
			isThinking: true,
			logId: targetLogId
		};
		const sessionId = getActiveSessionUUID();
		egoChatMessages.addMessage(sessionId, egoThinkingMessage);
		messages = messages.slice(0, userMessageIndex + 1).concat([egoThinkingMessage]);
		startStream(currentSession!.uuid);
		streamStore.currentLogId = targetLogId;
		const currentCachedFiles = cachedFiles.get();
		const sessionCachedFiles = (currentCachedFiles?.sessionUUID === currentSession!.uuid) ? currentCachedFiles.files : [];
		const requestData: any = {
			query: userMessageToRegen.text,
			mode: chatMode,
			session_uuid: currentSession!.uuid,
			cached_files: sessionCachedFiles,
			is_regeneration: true,
			request_log_id_to_regen: targetLogId,
			memory_enabled: preferencesStore.memoryEnabled
		};
		egoWs.send(requestData);
	}
	async function startEditing(message: ChatMessage) {
		if (!message.logId || !streamStore.isDone) return;
		editingLogId = message.logId ?? null;
		editingText = message.text;
		await tick();
		try { editingTextarea?.focus(); editingTextarea?.setSelectionRange(editingText.length, editingText.length); } catch {}
	}
	function cancelEditing() {
		editingLogId = null;
		editingText = '';
	}
	async function saveAndRegenerate() {
		if (!editingLogId || editingText.trim() === '' || !streamStore.isDone) return;
		const logIdToUpdate = editingLogId;
		const newText = editingText;
		const userMessage = messages.find((m) => m.logId === logIdToUpdate && m.author === 'user');
		if (!userMessage) return;
		const originalText = userMessage.text;
		const userMessageIndex = messages.findIndex((m) => m.id === userMessage.id);
		if (userMessageIndex === -1) return;
		const messagesToRemove = messages.slice(userMessageIndex + 1);
		for (const msg of messagesToRemove) {
			const sid = getActiveSessionUUID();
			egoChatMessages.removeMessage(sid, msg.id);
		}
		egoChatMessages.updateMessage(getActiveSessionUUID(), userMessage.id, { text: newText });
		const egoThinkingMessage: ChatMessage = {
			author: 'ego',
			id: Date.now(),
			text: '',
			isThinking: true,
			logId: logIdToUpdate
		};
		egoChatMessages.addMessage(getActiveSessionUUID(), egoThinkingMessage);
		cancelEditing();
		try {
			await api.patch(`/logs/${logIdToUpdate}`, { query: newText });
			startStream(currentSession!.uuid);
			streamStore.currentLogId = logIdToUpdate;
			const currentCachedFiles = cachedFiles.get();
			const sessionCachedFiles = (currentCachedFiles?.sessionUUID === currentSession!.uuid) ? currentCachedFiles.files : [];
			wsStore.connection?.send({
				query: newText,
				mode: chatMode,
				session_uuid: currentSession!.uuid,
				cached_files: sessionCachedFiles,
				is_regeneration: true,
				request_log_id_to_regen: logIdToUpdate,
				memory_enabled: preferencesStore.memoryEnabled
			});
		} catch (e: unknown) {
			if (e instanceof Error) {
				toast.error($_('errors.update_failed', { values: { message: e.message } }));
			} else {
				toast.error($_('errors.update_failed', { values: { message: 'Unknown error' } }));
			}
			if (userMessage) {
				egoChatMessages.updateMessage(getActiveSessionUUID(), userMessage.id, { text: originalText! });
			}
			const thinkingMessage = messages.find((m) => m.isThinking);
			if (thinkingMessage) {
				egoChatMessages.removeMessage(getActiveSessionUUID(), thinkingMessage.id);
			}
		}
	}
	function stopGeneration() {
		if (wsStore.connection) {
			wsStore.connection.stop();
		}
		stopStreamAsCancelled();
	}
	function handlePaste(event: ClipboardEvent) {
		const dt = event.clipboardData;
		if (!dt) return;
		const items = Array.from(dt.items || []);
		const imageFiles: File[] = [];
		for (const it of items) {
			if (it && typeof it.type === 'string' && it.type.startsWith('image/')) {
				const f = it.getAsFile();
				if (f) imageFiles.push(f);
			}
		}
		if (imageFiles.length > 0) {
			attachedFiles = [...attachedFiles, ...imageFiles];
			return;
		}
		const files = dt.files;
		if (files && files.length > 0) {
			attachedFiles = [...attachedFiles, ...Array.from(files)];
		}
	}
	function handleDrop(event: DragEvent) {
		event.preventDefault();
		if (event.dataTransfer?.files && event.dataTransfer.files.length > 0) {
			attachedFiles = [...attachedFiles, ...Array.from(event.dataTransfer.files)];
		}
	}
	async function saveSettings() {
		try {
			if (currentSession) {
				const updated = await api.patch<ChatSession>(`/sessions/${currentSession.uuid}`, {
					custom_instructions: customInstructionsInput
				});
				updateSession(updated);
			}
			toast.success($_('toasts.instructions_saved'));
			setShowSettingsModal(false);
		} catch (error: unknown) {
			if (error instanceof Error) {
				toast.error(error.message);
			} else {
				toast.error('Failed to save instructions');
			}
		}
	}
</script>
<div class="flex flex-col h-screen relative">
	<SessionCreationOverlay show={showCreationOverlay} />
	<ChatHeader {currentSession} />
	<MessageList
		bind:this={messageListComponent}
		{messages}
		{editingLogId}
		{editingText}
		{editingTextarea}
		{lastMessageLogId}
		{streamStore}
		{isMobile}
		onscroll={handleScroll}
		onstartedit={startEditing}
		oncanceledit={cancelEditing}
		onsaveandregen={saveAndRegenerate}
		onregenerate={regenerate}
	/>
	<ChatInput
		bind:currentInput
		bind:attachedFiles
		bind:chatMode
		{isMobile}
		{isConnecting}
		streamIsDone={streamStore.isDone}
		{editingLogId}
		onsend={sendMessage}
		onstop={stopGeneration}
		onpaste={handlePaste}
		ondrop={handleDrop}
	/>
	<Modal
		title={$_('chat.instructions_title')}
		show={uiStore.showSettingsModal}
		onclose={() => setShowSettingsModal(false)}
	>
		{#if currentSession || isNewSession}
			<p class="mb-4 text-text-secondary">{$_('chat.instructions_prompt')}</p>
			<textarea
				bind:value={customInstructionsInput}
				rows="5"
				class="w-full bg-primary border-2 border-tertiary rounded-lg p-3 focus:ring-accent focus:border-accent transition-all duration-300"
			></textarea>
			<button
				onclick={saveSettings}
				class="mt-4 w-full py-2 bg-accent rounded-lg font-semibold hover:bg-accent-hover transition-colors duration-200"
			>{$_('chat.save')}</button>
		{:else}
			<p class="mb-2 text-text-secondary">{$_('chat.start_dialog_first')}</p>
		{/if}
	</Modal>
</div>