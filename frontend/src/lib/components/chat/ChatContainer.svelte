<script lang="ts">
	import type { ChatMessage, ChatSession, ChatMode, FilePayload } from '$lib/types';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte.ts';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { cachedFiles } from '$lib/stores/cachedFiles.svelte.ts';
	import { api } from '$lib/api';
	import { connectionManager } from '$lib/connection-manager';
	import {
		startStream,
		stopStreamAsCancelled,
		resetStreamStore,
		streamStore
	} from '$lib/stores/stream.svelte.ts';
	import { untrack } from 'svelte';
	import { fade } from 'svelte/transition';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { inFlightStore } from '$lib/stores/inflight.svelte.ts';
	import { shouldCompress } from '$lib/utils/imageCompression';
	import { imageCompressionWorker } from '$lib/utils/imageCompressionWorker';
	import { debounce } from '$lib/utils/debounce';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import MessageList from './MessageList.svelte';
	import ChatInput from './ChatInput.svelte';
	import ChatMaintenanceOverlay from '../ChatMaintenanceOverlay.svelte';

	let {
		sessionID,
		initialSession,
		initialMessages,
		pendingCustomInstructions = ''
	}: {
		sessionID: string;
		initialSession: ChatSession | null;
		initialMessages: ChatMessage[];
		pendingCustomInstructions?: string;
	} = $props();

	// ============================================================================
	// LOCAL STATE
	// ============================================================================

	let chatMode = $state<ChatMode>('default');
	let currentInput = $state('');
	let attachedFiles = $state<File[]>([]);
	let userHasScrolledUp = $state(false);
	let messageListComponent: MessageList | undefined = $state();
	let editingLogId = $state<number | null>(null);
	let editingText = $state('');
	let editingTextarea: HTMLTextAreaElement | null = null;
	let isMobile = $state(false);
	let sending = $state(false);

	// ============================================================================
	// DERIVED STATE
	// ============================================================================

	let currentSession = $derived(chatStore.getSessionById(sessionID) || initialSession);

	let messages = $derived(
		(() => {
			if (chatStore.messages && chatStore.messages.length > 0) {
				return chatStore.messages;
			}
			// Fallback to initialMessages
			return initialMessages;
		})()
	);

	// ============================================================================
	// HELPER FUNCTIONS
	// ============================================================================

	function scrollToBottom(behavior: 'smooth' | 'auto' = 'smooth') {
		messageListComponent?.scrollToBottom(behavior);
		userHasScrolledUp = false;
	}

	function isNearBottom(threshold = 200) {
		return messageListComponent?.isNearBottom(threshold) ?? true;
	}

	function handleScroll() {
		// Use a smaller threshold for setting userHasScrolledUp
		const isScrolledToBottom = isNearBottom(80);
		userHasScrolledUp = !isScrolledToBottom;
	}

	function getActiveSessionUUID(): string {
		if (sessionID && sessionID !== 'new') return sessionID;
		if (chatStore.streamSessionUUID) return chatStore.streamSessionUUID;
		if (currentSession?.uuid) return currentSession.uuid;
		return 'new';
	}

	// ============================================================================
	// MESSAGE ACTIONS
	// ============================================================================

	async function sendMessage() {
		if (sending || (!currentInput.trim() && attachedFiles.length === 0)) return;

		// Validation for Video/Audio based on provider
		const hasLargeMedia = attachedFiles.some((f) => {
			const type = f.type.toLowerCase();
			return (
				type.startsWith('video/') ||
				type.startsWith('audio/') ||
				type === 'application/pdf' ||
				type.startsWith('text/')
			);
		});

		const currentProvider = auth.user?.llm_provider || 'ego';
		if (hasLargeMedia && currentProvider !== 'ego' && currentProvider !== 'gemini') {
			toast.error(
				$_('chat.media_only_ego_gemini') ||
					'Advanced files (Video, Audio, PDF) are only supported for EGO and Gemini models.'
			);
			return;
		}

		sending = true;

		let tempAttachedFiles = attachedFiles;
		try {
			const filesToCompress = tempAttachedFiles.filter(shouldCompress);
			const filesToKeep = tempAttachedFiles.filter((f) => !shouldCompress(f));

			if (filesToCompress.length > 0) {
				const compressed = await imageCompressionWorker.compressImages(filesToCompress);
				tempAttachedFiles = [...filesToKeep, ...compressed];
			}
		} catch (error) {
			console.error('[ChatContainer] Image compression failed, using original files:', error);
		}

		const tempCurrentInput = currentInput;
		const isNew = !currentSession;
		const tempId = Date.now();

		const optimisticUserMessage: ChatMessage = {
			author: 'user',
			id: tempId,
			text: tempCurrentInput,
			attachments: tempAttachedFiles.map((file) => ({
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
		chatStore.addOptimisticMessages(activeSessionUUID, [
			optimisticUserMessage,
			optimisticEgoMessage
		]);

		scrollToBottom('smooth');

		if (isNew) {
			inFlightStore.set('new', tempCurrentInput, tempAttachedFiles);
		}

		currentInput = '';
		attachedFiles = [];

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
					} catch {
						// ignore
					}

					const mime = file.type && file.type.length > 0 ? file.type : inferredMime || 'image/png';
					let fname = file.name && file.name.length > 0 ? file.name : '';

					if (!fname) {
						fname = `pasted-image-${Date.now()}.png`;
					}

					const base64 = dataUrl.startsWith('data:') ? dataUrl.split(',')[1] || '' : dataUrl;

					return {
						base64_data: base64,
						mime_type: mime,
						file_name: fname
					} satisfies FilePayload;
				})
			);

			// Handle session creation if it's new
			if (isNew && (!activeSessionUUID || activeSessionUUID === 'new')) {
				try {
					const customInstructions =
						pendingCustomInstructions ||
						(currentSession as ChatSession | null)?.custom_instructions;
					const newSession = await api.post<ChatSession>('/sessions', {
						custom_instructions: customInstructions || undefined
					});

					if (newSession && newSession.uuid) {
						chatStore.addSession(newSession);
						const oldUUID = activeSessionUUID;
						activeSessionUUID = newSession.uuid;

						chatStore.setNewlyCreatedSessionUUID(newSession.uuid);

						if (oldUUID && oldUUID !== newSession.uuid) {
							chatStore.migrateMessages(oldUUID, newSession.uuid);
						}

						// Update URL immediately without waiting (non-blocking)
						// This prevents delays in starting the stream
						goto(`/chat/${newSession.uuid}`, { replaceState: true, invalidateAll: false });
					} else {
						throw new Error('Failed to create session: No UUID returned');
					}
				} catch (e) {
					console.error('[ChatContainer] Pre-create session failed', e);
					toast.error(
						$_('errors.session_creation_failed') ||
							'Failed to create a new session. Please try again.'
					);
					sending = false;
					return;
				}
			}

			startStream(activeSessionUUID);

			const currentCachedFiles = !isNew ? cachedFiles.get() : null;
			const sessionCachedFiles =
				currentCachedFiles?.sessionUUID === activeSessionUUID ? currentCachedFiles.files : [];

			const payload = {
				query: tempCurrentInput,
				mode: chatMode,
				session_uuid: activeSessionUUID && activeSessionUUID !== 'new' ? activeSessionUUID : null,
				files: filePayloads,
				cached_files: sessionCachedFiles,
				custom_instructions:
					!activeSessionUUID || activeSessionUUID === 'new'
						? pendingCustomInstructions || currentSession?.custom_instructions
						: undefined,
				is_regeneration: false,
				temp_id: tempId,
				memory_enabled: preferencesStore.memoryEnabled,
				user_id: auth.user?.id ? String(auth.user.id) : undefined
			};

			// Using connectionManager to send via POST + SSE
			connectionManager.sendMessage(payload);

			if (filePayloads.length > 0 && !isNew) {
				const imagesToCache = filePayloads
					.filter((fp) => fp.mime_type.startsWith('image/'))
					.map((fp) => ({
						file_name: fp.file_name,
						mime_type: fp.mime_type,
						uri: `data:${fp.mime_type};base64,${fp.base64_data}`
					}));

				if (imagesToCache.length > 0) {
					cachedFiles.addFiles(activeSessionUUID, imagesToCache);
				}
			}
		} catch (error: unknown) {
			console.error('[ChatContainer] Error sending message:', error);
			toast.error($_('errors.send_failed'));
			chatStore.removeMessage(activeSessionUUID, optimisticUserMessage.id);
			chatStore.removeMessage(activeSessionUUID, optimisticEgoMessage.id);
			currentInput = tempCurrentInput;
			attachedFiles = tempAttachedFiles;
		} finally {
			sending = false;
		}
	}

	function regenerate(message: ChatMessage) {
		if (editingLogId !== null) return;
		if (chatStore.isRegenerating || chatStore.isEditingSaving) return;
		if (!chatStore.streamIsDone) return;

		let targetLogId: number | null = null;
		let userMessageToRegen: ChatMessage | null = null;

		if (message.logId) {
			targetLogId = message.logId;
			userMessageToRegen =
				messages.find((m: ChatMessage) => m.logId === targetLogId && m.author === 'user') || null;
		} else if (message.isThinking && chatStore.stream.currentLogId) {
			targetLogId = chatStore.stream.currentLogId;
			const thinkingIndex = messages.findIndex((m: ChatMessage) => m.id === message.id);
			if (thinkingIndex > 0) {
				userMessageToRegen = messages[thinkingIndex - 1];
			}
		} else if (!message.isThinking) {
			const lastUserWithLog = [...messages]
				.reverse()
				.find((m: ChatMessage) => m.author === 'user' && m.logId);
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

		chatStore.setRegenerating(true);

		const userMessageIndex = messages.findIndex(
			(m: ChatMessage) => m.id === userMessageToRegen!.id
		);
		if (userMessageIndex === -1) {
			chatStore.setRegenerating(false);
			return;
		}

		const sessionId = getActiveSessionUUID();

		// Important: Remove everything AFTER the user message we are regenerating for.
		// We keep the user message.
		// EGO's old response was at userMessageIndex + 1.
		chatStore.regenerateFrom(sessionId, userMessageIndex + 1);

		resetStreamStore();

		const egoThinkingMessage: ChatMessage = {
			author: 'ego',
			id: Date.now(),
			text: '',
			isThinking: true,
			logId: targetLogId
		};

		chatStore.addMessage(sessionId, egoThinkingMessage);

		startStream(currentSession!.uuid);
		chatStore.stream.currentLogId = targetLogId;

		const currentCachedFiles = cachedFiles.get();
		const sessionCachedFiles =
			currentCachedFiles?.sessionUUID === currentSession!.uuid ? currentCachedFiles.files : [];

		const requestData = {
			query: userMessageToRegen.text,
			mode: chatMode,
			session_uuid: currentSession!.uuid,
			cached_files: sessionCachedFiles,
			is_regeneration: true,
			request_log_id_to_regen: targetLogId,
			memory_enabled: preferencesStore.memoryEnabled
		};

		connectionManager.sendMessage(requestData);

		setTimeout(() => {
			chatStore.setRegenerating(false);
		}, 500);
	}

	async function startEditing(message: ChatMessage) {
		if (!message.logId || !chatStore.streamIsDone) return;
		if (chatStore.isRegenerating || chatStore.isEditingSaving) return;

		editingLogId = message.logId ?? null;
		editingText = message.text;
		await tick();
		try {
			editingTextarea?.focus();
			editingTextarea?.setSelectionRange(editingText.length, editingText.length);
		} catch {
			// ignore
		}
	}

	function cancelEditing() {
		editingLogId = null;
		editingText = '';
	}

	async function saveAndRegenerateInternal() {
		if (!editingLogId || editingText.trim() === '' || !chatStore.streamIsDone) return;
		if (chatStore.isRegenerating || chatStore.isEditingSaving) return;

		const logIdToUpdate = editingLogId;
		const newText = editingText;
		const sessionId = getActiveSessionUUID();

		const userMessage = messages.find(
			(m: ChatMessage) => m.logId === logIdToUpdate && m.author === 'user'
		);

		if (!userMessage) return;

		const userMessageIndex = messages.findIndex((m: ChatMessage) => m.id === userMessage.id);
		if (userMessageIndex === -1) return;

		chatStore.setEditingSaving(true);

		const messagesSnapshot = chatStore.createMessagesSnapshot(sessionId);

		if (!messagesSnapshot) {
			chatStore.setEditingSaving(false);
			return;
		}

		resetStreamStore();

		// Remove old response
		chatStore.regenerateFrom(sessionId, userMessageIndex + 1);

		// Update user message text
		chatStore.updateMessage(sessionId, userMessage.id, { text: newText });

		const egoThinkingMessage: ChatMessage = {
			author: 'ego',
			id: Date.now(),
			text: '',
			isThinking: true,
			logId: logIdToUpdate
		};

		chatStore.addMessage(sessionId, egoThinkingMessage);
		cancelEditing();

		try {
			await api.patch<{ success: boolean }>(`/logs/${logIdToUpdate}`, { query: newText });
			startStream(currentSession!.uuid);
			chatStore.stream.currentLogId = logIdToUpdate;

			const currentCachedFiles = cachedFiles.get();
			const sessionCachedFiles =
				currentCachedFiles?.sessionUUID === currentSession!.uuid ? currentCachedFiles.files : [];

			connectionManager.sendMessage({
				query: newText,
				mode: chatMode,
				session_uuid: currentSession!.uuid,
				cached_files: sessionCachedFiles,
				is_regeneration: true,
				request_log_id_to_regen: logIdToUpdate,
				memory_enabled: preferencesStore.memoryEnabled
			});

			setTimeout(() => {
				chatStore.setEditingSaving(false);
			}, 500);
		} catch (e: unknown) {
			chatStore.restoreMessagesFromSnapshot(sessionId, messagesSnapshot);
			if (e instanceof Error) {
				toast.error($_('errors.update_failed', { values: { message: e.message } }));
			} else {
				toast.error($_('errors.update_failed', { values: { message: 'Unknown error' } }));
			}

			// Show retry button
			toast.error($_('chat.edit_failed_retry'), {
				action: {
					label: $_('chat.retry'),
					onClick: () => saveAndRegenerate()
				}
			});

			chatStore.setEditingSaving(false);
		}
	}

	const saveAndRegenerate = debounce(saveAndRegenerateInternal, 300);

	function stopGeneration() {
		connectionManager.stopStream();
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

	// ============================================================================
	// EFFECTS
	// ============================================================================

	// Initialize messages for session
	$effect(() => {
		if (sessionID && sessionID !== 'new') {
			const currentMessages = chatStore.messages;
			const loadedSessionUUID = chatStore.messagesSessionUUID;

			// If we are actively streaming or just created this session in this client,
			// favor the in-memory store state to prevent overwriting optimistic messages
			// with potentially stale server state during navigation/loading.
			// ALSO: Don't overwrite if we're in the middle of a reconnection attempt
			if (
				loadedSessionUUID === sessionID &&
				currentMessages.length > 0 &&
				(chatStore.newlyCreatedSessionUUID === sessionID ||
					!streamStore.isDone ||
					streamStore.isRecovering)
			) {
				return;
			}

			// Otherwise, sync with initialMessages from the server load
			// Only set if we haven't loaded this session yet, or if store is empty but server has data.
			// Checking initialMessages.length > 0 alone causes infinite loops if loadedSessionUUID === sessionID.
			if (
				loadedSessionUUID !== sessionID ||
				(currentMessages.length === 0 && initialMessages.length > 0)
			) {
				chatStore.setMessages(sessionID, initialMessages);
			}
		}
	});

	// Check if we need to reconnect to an active stream
	$effect(() => {
		// Run this check when sessionID changes or when messages load
		if (sessionID && sessionID !== 'new' && messages.length > 0) {
			untrack(() => {
				// Logic: If the last message is from 'user', or from 'ego' but looks like it's thinking/loading
				// or if we just want to be safe, we can try to reconnect.
				// A safer bet: If the last message is User, OR the last message is Ego but incomplete (though we only store final text usually?)
				// Actually, if we left mid-stream, the DB might have the user message but no Ego message, or a partial one?
				// The standard behavior of the backend: it saves the User query immediately.
				// It updates the log with the response only when finished.
				// So if we have a User message as the last one, it's highly likely we are waiting for a response.

				const lastMsg = messages[messages.length - 1];
				if (lastMsg.author === 'user') {
					// We should try to reconnect/listen
					// But don't do it if we are already streaming!
					if (!streamStore.streaming) {
						console.log('[ChatContainer] Last message is user, attempting to reconnect stream...');
						// Create a fake "thinking" message if needed?
						// If we are reloading the page, the server DB has the user message.
						// But the UI needs a placeholder for the answer.

						const tempId = Date.now();
						const optimisticEgoMessage: ChatMessage = {
							author: 'ego',
							id: tempId,
							text: '',
							isThinking: true
						};

						// Check if we already have an optimistic ego message?
						// If the user refreshed, `messages` comes from `initialMessages` (from DB).
						// DB usually has User Query. If job is running, DB does NOT have the response yet.
						// So `messages` ends with User.

						chatStore.addOptimisticMessages(sessionID, [optimisticEgoMessage]);

						connectionManager.reconnectStream(sessionID);
					}
				}
			});
		}
	});

	// Detect mobile
	$effect(() => {
		const mediaQuery = window.matchMedia('(max-width: 768px)');
		const update = (e: MediaQueryList | MediaQueryListEvent) => (isMobile = e.matches);
		mediaQuery.addEventListener('change', update);
		update(mediaQuery);
		return () => mediaQuery.removeEventListener('change', update);
	});

	// Auto-scroll on stream updates
	$effect(() => {
		if (chatStore.textStream) {
			untrack(() => {
				// Only scroll if we were already at the bottom or the stream just started
				if (!userHasScrolledUp) {
					scrollToBottom('auto');
				}
			});
		}
	});

	// Force scroll to bottom when stream finishes to prevent jumping back
	$effect(() => {
		if (streamStore.isDone) {
			untrack(() => {
				if (!userHasScrolledUp && chatStore.textStream) {
					// Small delay to let final markdown render
					setTimeout(() => {
						scrollToBottom('smooth');
					}, 50);
				}
			});
		}
	});

	// Consume last user message log ID
	$effect(() => {
		const lm = streamStore.lastUserMessage;
		if (lm) {
			untrack(() => {
				const sid = getActiveSessionUUID();
				try {
					chatStore.updateMessage(sid, lm.temp_id, { logId: lm.log_id });
					chatStore.updateMessage(sid, lm.temp_id + 1, { logId: lm.log_id });
					chatStore.clearLastUserMessage();
				} catch (e) {
					console.error('[ChatContainer] Error updating messages with logId:', e);
				}
			});
		}
	});

	// Consume stream result: Handled inside chatStore.finish() now to avoid effect cycles.
	// We rely on streamStore reactivity for ongoing updates.

	// Clear in-flight on route change
	$effect(() => {
		const inFlight = inFlightStore.get();
		if (inFlight && (inFlight.sessionUUID === 'new' || inFlight.sessionUUID === sessionID)) {
			inFlightStore.clear();
		}
	});

	// Scroll to bottom when sessionID changes
	$effect(() => {
		if (sessionID) {
			untrack(() => {
				scrollToBottom('auto');
			});
		}
	});
</script>

<div class="relative flex h-screen flex-col">
	<MessageList
		bind:this={messageListComponent}
		{messages}
		{editingLogId}
		{editingText}
		{editingTextarea}
		{streamStore}
		{isMobile}
		onscroll={handleScroll}
		onstartedit={startEditing}
		oncanceledit={cancelEditing}
		onsaveandregen={saveAndRegenerate}
		onregenerate={regenerate}
	/>

	{#if maintenanceStore.isChatMaintenanceActive}
		<div
			class="fixed bottom-0 z-20 w-full bg-gradient-to-t from-primary via-primary/80 to-transparent pb-8 pt-32 transition-all duration-500"
			style="left: var(--sidebar-offset); right: 0; width: auto;"
			in:fade={{ duration: 300 }}
		>
			<div class="mx-auto w-full max-w-4xl px-4">
				<ChatMaintenanceOverlay />
			</div>
		</div>
	{:else}
		<ChatInput
			bind:currentInput
			bind:attachedFiles
			bind:chatMode
			{isMobile}
			streamIsDone={streamStore.isDone}
			{editingLogId}
			onsend={sendMessage}
			onstop={stopGeneration}
			onpaste={handlePaste}
			ondrop={handleDrop}
		/>
	{/if}
</div>
