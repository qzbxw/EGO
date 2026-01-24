import { setIsThinking } from '$lib/stores/ui.svelte.ts';
import { chatStore } from './chat.svelte';

// Re-export all stream-related actions from chatStore
export const startStream = (sessionUUID: string | null) => {
	chatStore.startStream(sessionUUID);
	setIsThinking(true);
};

export const setThoughtHeader = chatStore.setThoughtHeader.bind(chatStore);
export const appendToStream = chatStore.appendToStream.bind(chatStore);

export const endStream = () => {
	chatStore.endStream();
};

export const stopStreamAsCancelled = () => {
	chatStore.stopStreamAsCancelled();
	setIsThinking(false);
};

export const setStreamError = (errorMessage: string) => {
	chatStore.setStreamError(errorMessage);
	setIsThinking(false);
};

export const setLastUserMessageLogId = chatStore.setLastUserMessageLogId.bind(chatStore);
export const consumeLastUserMessageLogId = chatStore.consumeLastUserMessageLogId.bind(chatStore);

export const resetStreamStore = () => {
	chatStore.resetStream();
	setIsThinking(false);
};

export const tryHydrateFromStorage = (sessionUUID: string): boolean => {
	const success = chatStore.tryHydrateFromStorage(sessionUUID);
	if (success && !chatStore.streamIsDone) {
		setIsThinking(true);
	}
	return success;
};

// Stream store object for component compatibility
export const streamStore = {
	get thoughtHeader() {
		return chatStore.stream.thoughtHeader;
	},
	set thoughtHeader(value: string) {
		chatStore.setThoughtHeader(value);
	},

	get thoughts() {
		return chatStore.stream.thoughts;
	},

	get activeTools() {
		return chatStore.stream.activeTools;
	},

	get textStream() {
		return chatStore.stream.textStream;
	},
	set textStream(value: string) {
		// Direct setting not recommended - use appendToStream instead
		chatStore.stream.textStream = value;
	},

	get isDone() {
		return chatStore.stream.isDone;
	},
	set isDone(value: boolean) {
		chatStore.stream.isDone = value;
	},

    // Alias for isDone for backwards compatibility/semantics
    get streaming() {
        return !chatStore.stream.isDone;
    },
    set streaming(value: boolean) {
        chatStore.stream.isDone = !value;
    },

	get error() {
		return chatStore.stream.error;
	},
	set error(value: string) {
		chatStore.stream.error = value;
	},

	get sessionUUID() {
		return chatStore.stream.sessionUUID;
	},
	set sessionUUID(value: string | null) {
		chatStore.stream.sessionUUID = value;
	},

	get newlyCreatedSessionUUID() {
		return chatStore.stream.newlyCreatedSessionUUID;
	},
	set newlyCreatedSessionUUID(value: string | null) {
		chatStore.setNewlyCreatedSessionUUID(value);
	},

	get lastUserMessage() {
		return chatStore.stream.lastUserMessage;
	},
	get currentLogId() {
		return chatStore.stream.currentLogId;
	},
	set currentLogId(value: number | null) {
		chatStore.stream.currentLogId = value;
	},

	get isRecovering() {
		return chatStore.stream.isRecovering;
	},
	set isRecovering(value: boolean) {
		chatStore.stream.isRecovering = value;
	},

	get recoveredFromRest() {
		return chatStore.stream.recoveredFromRest;
	},
	set recoveredFromRest(value: boolean) {
		chatStore.stream.recoveredFromRest = value;
	},

	get lastSeqNumber() {
		return chatStore.stream.lastSeqNumber;
	},
	set lastSeqNumber(value: number) {
		chatStore.stream.lastSeqNumber = value;
	}
};
