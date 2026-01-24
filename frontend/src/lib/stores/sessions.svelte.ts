import type { ChatSession } from '$lib/types';
import { chatStore, setNotificationCallbacks } from './chat.svelte';

// Re-export all actions from chatStore
export const setInitialSessions = chatStore.setInitialSessions.bind(chatStore);
export const addSession = chatStore.addSession.bind(chatStore);
export const updateSession = chatStore.updateSession.bind(chatStore);
export const removeSession = chatStore.removeSession.bind(chatStore);
export const clearUserSessions = chatStore.clearUserSessions.bind(chatStore);
export const bringSessionToTop = chatStore.bringSessionToTop.bind(chatStore);
export const getSessionById = chatStore.getSessionById.bind(chatStore);

// Subscribers for sessions and isLoading
type Subscriber<T> = (value: T) => void;
type Unsubscriber = () => void;

const sessionsSubscribers = new Set<Subscriber<ChatSession[]>>();
const isLoadingSubscribers = new Set<Subscriber<boolean>>();

// Notify functions to be called when values change
function notifySessionsChanged() {
	const currentValue = chatStore.sessions;
	sessionsSubscribers.forEach((callback) => callback(currentValue));
}

function notifyIsLoadingChanged() {
	const currentValue = chatStore.sessionsLoading;
	isLoadingSubscribers.forEach((callback) => callback(currentValue));
}

// Register notification callbacks with chatStore
setNotificationCallbacks(notifySessionsChanged, notifyIsLoadingChanged);

// Svelte-compatible stores
export const sessions = {
	subscribe(callback: Subscriber<ChatSession[]>): Unsubscriber {
		// Immediately call with current value
		callback(chatStore.sessions);

		// Add to subscribers
		sessionsSubscribers.add(callback);

		// Return unsubscribe function
		return () => {
			sessionsSubscribers.delete(callback);
		};
	}
};

export const isLoading = {
	subscribe(callback: Subscriber<boolean>): Unsubscriber {
		// Immediately call with current value
		callback(chatStore.sessionsLoading);

		// Add to subscribers
		isLoadingSubscribers.add(callback);

		// Return unsubscribe function
		return () => {
			isLoadingSubscribers.delete(callback);
		};
	}
};
