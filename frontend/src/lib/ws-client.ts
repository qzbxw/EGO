import { addSession } from '$lib/stores/sessions.svelte.ts';
import { toast } from 'svelte-sonner';
import { _ } from 'svelte-i18n';
import { get as getStore } from 'svelte/store';
import {
	streamStore,
	setThoughtHeader,
	appendToStream,
	endStream,
	setStreamError,
	setLastUserMessageLogId
} from '$lib/stores/stream.svelte.ts';
import { connectWebSocket, type EgoWebSocket } from '$lib/websocket';
import { wsStore } from '$lib/stores/websocket.svelte.ts';
import { auth } from '$lib/stores/auth.svelte.ts';
let wsInstance: EgoWebSocket | null = null;
export function initializeWebSocket() {
	if (wsInstance || !auth.accessToken) {
		return;
	}
	console.log('Initializing WebSocket connection...');
	wsInstance = connectWebSocket(auth.accessToken, {
		onOpen: () => {
			console.log('WebSocket connection established successfully.');
			wsStore.setConnection(wsInstance);
		},
		onClose: () => {
			console.log('WebSocket connection closed.');
			wsStore.setConnection(null);
			wsInstance = null;
		},
		onSessionCreated: (newSession) => {
			if (newSession && newSession.uuid) {
				addSession(newSession);
				streamStore.sessionUUID = newSession.uuid;
			}
		},
		onLogCreated: ({ log_id, temp_id }) => {
			setLastUserMessageLogId(temp_id, log_id);
		},
		onThoughtHeader: setThoughtHeader,
		onChunk: appendToStream,
		onDone: endStream,
        onError: (errorMsg) => {
			setStreamError(errorMsg);
            toast.error(errorMsg || getStore(_ )('errors.generic'));
		},
		onPong: () => {
		}
	});
}
export function disconnectWebSocket() {
	if (wsInstance) {
		console.log('Disconnecting WebSocket.');
		wsInstance.close();
		wsInstance = null;
		wsStore.setConnection(null);
	}
}