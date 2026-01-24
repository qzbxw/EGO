import type { ChatMessage } from '$lib/types';
import { chatStore } from './chat.svelte';
import { browser } from '$app/environment';

// Re-export all actions from chatStore
export const setMessages = chatStore.setMessages.bind(chatStore);
export const addMessage = chatStore.addMessage.bind(chatStore);
export const addOptimisticMessages = chatStore.addOptimisticMessages.bind(chatStore);
export const updateMessage = chatStore.updateMessage.bind(chatStore);
export const removeMessage = chatStore.removeMessage.bind(chatStore);
export const clear = chatStore.clearMessages.bind(chatStore);
export const regenerateFrom = chatStore.regenerateFrom.bind(chatStore);
export const createMessagesSnapshot = chatStore.createMessagesSnapshot.bind(chatStore);
export const restoreMessagesFromSnapshot = chatStore.restoreMessagesFromSnapshot.bind(chatStore);
export const migrateMessages = chatStore.migrateMessages.bind(chatStore);

// Compatibility wrapper for Svelte 4 store subscription pattern
// Note: In Svelte 5, prefer using chatStore directly instead of this wrapper
// This is kept for backwards compatibility only
export const chatMessages = {
	subscribe(callback: (value: { sessionUUID: string | null; messages: ChatMessage[] }) => void) {
		if (!browser) {
			callback({
				sessionUUID: null,
				messages: []
			});
			return () => {};
		}

		// Initial call with current state
		callback({
			sessionUUID: chatStore.stream.sessionUUID,
			messages: chatStore.messages
		});

		// Since chatStore uses Svelte 5 $state, components will automatically track changes
		// This subscribe is mainly for backwards compatibility
		// Return a no-op unsubscribe function
		return () => {
			// No-op: Svelte 5 handles reactivity automatically
		};
	},
	get: () => ({
		sessionUUID: chatStore.stream.sessionUUID,
		messages: chatStore.messages,
		newlyCreatedSessionUUID: chatStore.newlyCreatedSessionUUID
	}),
	setMessages,
	addMessage,
	addOptimisticMessages,
	updateMessage,
	removeMessage,
	clear
};
