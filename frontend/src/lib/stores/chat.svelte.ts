import type { ChatSession, ChatMessage, SessionPlan, ThoughtStep, ToolAction } from '$lib/types';
import { browser } from '$app/environment';

// ============================================================================
// TYPES
// ============================================================================

interface SuperEgoAgent {
	name: string;
	role: 'researcher' | 'solver' | 'critic' | 'optimizer' | 'synthesizer';
	status: 'running' | 'completed' | 'error';
	message?: string;
	error?: string;
}

interface SuperEgoRound {
	number: number;
	title: string;
	agents: SuperEgoAgent[];
	completed: boolean;
}

interface SuperEgoDebate {
	active: boolean;
	rounds: SuperEgoRound[];
	completed: boolean;
	summary?: string;
}

interface MessagesState {
	sessionUUID: string | null;
	messages: ChatMessage[];
}

interface StreamState {
	thoughtHeader: string;
	thoughts: ThoughtStep[];
	activeTools: ToolAction[];
	textStream: string;
	isDone: boolean;
	error: string;
	sessionUUID: string | null;
	newlyCreatedSessionUUID: string | null;
	lastUserMessage: { temp_id: number; log_id: number } | null;
	wasCancelled: boolean;
	currentLogId: number | null;
	isRecovering: boolean;
	recoveredFromRest: boolean;
	lastSeqNumber: number;
	activePlan: SessionPlan | null;
	superEgoDebate: SuperEgoDebate | null;
}

interface UILockState {
	isRegenerating: boolean;
	isEditingSaving: boolean;
}

// ============================================================================
// STATE
// ============================================================================

// Sessions state
let sessions = $state<ChatSession[]>([]);
let sessionsLoading = $state<boolean>(true);

// Messages state
let messagesStore = $state<MessagesState>({ sessionUUID: null, messages: [] });

// Stream state
let stream = $state<StreamState>({
	thoughtHeader: '',
	thoughts: [],
	activeTools: [],
	textStream: '',
	isDone: true,
	error: '',
	sessionUUID: null,
	newlyCreatedSessionUUID: null,
	lastUserMessage: null,
	wasCancelled: false,
	currentLogId: null,
	isRecovering: false,
	recoveredFromRest: false,
	lastSeqNumber: -1,
	activePlan: null,
	superEgoDebate: null
});

// UI Lock state
const uiLock = $state<UILockState>({
	isRegenerating: false,
	isEditingSaving: false
});

// Derived states
const currentSessionMessages = $derived(messagesStore.sessionUUID ? messagesStore.messages : []);

const hasActiveSessions = $derived(sessions.length > 0);

const isUILocked = $derived(uiLock.isRegenerating || uiLock.isEditingSaving || !stream.isDone);

// ============================================================================
// SESSIONS ACTIONS
// ============================================================================

// Notification callbacks
let notifySessionsChangedFn: (() => void) | null = null;
let notifyIsLoadingChangedFn: (() => void) | null = null;

export function setNotificationCallbacks(
	sessionsCallback: () => void,
	isLoadingCallback: () => void
) {
	notifySessionsChangedFn = sessionsCallback;
	notifyIsLoadingChangedFn = isLoadingCallback;
}

function notifyStoreSubscribers() {
	if (notifySessionsChangedFn) notifySessionsChangedFn();
	if (notifyIsLoadingChangedFn) notifyIsLoadingChangedFn();
}

export const chatStore = {
	// Getters for reactive state
	get sessions() {
		return sessions;
	},
	get sessionsLoading() {
		return sessionsLoading;
	},
	get messages() {
		return messagesStore.messages;
	},
	get currentSessionMessages() {
		return currentSessionMessages;
	},
	get messagesSessionUUID() {
		return messagesStore.sessionUUID;
	},
	get hasActiveSessions() {
		return hasActiveSessions;
	},

	// Stream getters
	get stream() {
		return stream;
	},
	get activePlan() {
		return stream.activePlan;
	},
	get thoughtHeader() {
		return stream.thoughtHeader;
	},
	get textStream() {
		return stream.textStream;
	},
	get streamIsDone() {
		return stream.isDone;
	},
	get streamError() {
		return stream.error;
	},
	get streamSessionUUID() {
		return stream.sessionUUID;
	},
	get newlyCreatedSessionUUID() {
		return stream.newlyCreatedSessionUUID;
	},

	// UI Lock getters
	get uiLock() {
		return uiLock;
	},
	get isUILocked() {
		return isUILocked;
	},
	get isRegenerating() {
		return uiLock.isRegenerating;
	},
	get isEditingSaving() {
		return uiLock.isEditingSaving;
	},

	// Sessions actions
	setInitialSessions(initialSessions: ChatSession[]) {
		if (browser) {
			sessions = Array.isArray(initialSessions) ? initialSessions : [];
			sessionsLoading = false;
			notifyStoreSubscribers();
		}
	},

	addSession(newSession: ChatSession) {
		if (browser) {
			if (sessions.some((s) => s.uuid === newSession.uuid)) {
				return;
			}
			sessions = [newSession, ...sessions];
			notifyStoreSubscribers();
		}
	},

	updateSession(updatedSession: Partial<ChatSession>) {
		if (browser) {
			const index = sessions.findIndex((s) => s.uuid === updatedSession.uuid);
			if (index === -1) return;

			const updated = { ...sessions[index], ...updatedSession } as ChatSession;
			sessions = [...sessions.slice(0, index), updated, ...sessions.slice(index + 1)];
			notifyStoreSubscribers();
		}
	},

	updateSessionTitle(uuid: string, title: string) {
		if (browser) {
			const session = sessions.find((s) => s.uuid === uuid);
			if (session) {
				session.title = title;
				notifyStoreSubscribers();
			}
		}
	},

	removeSession(sessionUUID: string) {
		if (browser) {
			sessions = sessions.filter((s) => s.uuid !== sessionUUID);
			notifyStoreSubscribers();
		}
	},

	clearUserSessions() {
		if (browser) {
			sessions = [];
			sessionsLoading = true;
			notifyStoreSubscribers();
		}
	},

	bringSessionToTop(sessionUUID: string) {
		if (browser) {
			const sessionIndex = sessions.findIndex((s) => s.uuid === sessionUUID);
			if (sessionIndex > 0) {
				const sessionToMove = sessions[sessionIndex];
				sessions = [
					sessionToMove,
					...sessions.slice(0, sessionIndex),
					...sessions.slice(sessionIndex + 1)
				];
				notifyStoreSubscribers();
			}
		}
	},

	getSessionById(sessionUUID: string): ChatSession | undefined {
		return browser ? sessions.find((s) => s.uuid === sessionUUID) : undefined;
	},

	// Messages actions
	setMessages(sessionUUID: string, messages: ChatMessage[]) {
		if (browser) {
			messagesStore = { sessionUUID, messages };
			// Update stream sessionUUID to keep it in sync
			// Only update if stream is idle or if it matches
			if (stream.isDone || !stream.sessionUUID || stream.sessionUUID === sessionUUID) {
				stream.sessionUUID = sessionUUID;
				// If switching session, reset plan
				if (stream.sessionUUID !== sessionUUID) {
					stream.activePlan = null;
				}
			}
		}
	},

	addMessage(sessionUUID: string, message: ChatMessage) {
		if (browser) {
			if (messagesStore.sessionUUID !== sessionUUID) {
				console.warn('[ChatStore] Adding message to a different session.');
				// Initialize if empty, but warn
				if (!messagesStore.sessionUUID) {
					messagesStore = { sessionUUID, messages: [message] };
					return;
				}
				return;
			}
			messagesStore = {
				...messagesStore,
				messages: [...messagesStore.messages, message]
			};
		}
	},

	addOptimisticMessages(sessionUUID: string, messages: ChatMessage[]) {
		if (browser) {
			// If switching to a different session, replace all messages
			if (messagesStore.sessionUUID !== sessionUUID && messagesStore.sessionUUID !== null) {
				console.warn(
					'[ChatStore] Adding optimistic messages to a different session, replacing all messages.'
				);
				messagesStore = { sessionUUID, messages };
				// Update stream sessionUUID to match
				stream.sessionUUID = sessionUUID;
				stream.activePlan = null;
			} else {
				// Otherwise, append messages
				messagesStore = {
					sessionUUID,
					messages: [...messagesStore.messages, ...messages]
				};
				// Update stream sessionUUID if not set
				if (!stream.sessionUUID) {
					stream.sessionUUID = sessionUUID;
				}
			}
		}
	},

	updateMessage(sessionUUID: string, messageId: number, updates: Partial<ChatMessage>) {
		if (browser) {
			if (messagesStore.sessionUUID !== sessionUUID) {
				return;
			}

			const index = messagesStore.messages.findIndex((m) => m.id === messageId);
			if (index !== -1) {
				const updated = { ...messagesStore.messages[index], ...updates };
				messagesStore = {
					...messagesStore,
					messages: [
						...messagesStore.messages.slice(0, index),
						updated,
						...messagesStore.messages.slice(index + 1)
					]
				};
			}
		}
	},

	updateMessageLogId(temp_id: number, log_id: number) {
		if (browser) {
			if (!messagesStore.sessionUUID) return;

			// Find by temp_id which we assume is the 'id' field for optimistic messages
			// But optimistic messages have Date.now() as ID.
			// We need to check if the backend returns the same temp_id.
			// Assuming temp_id corresponds to the message.id of the optimistic message
			const index = messagesStore.messages.findIndex((m) => m.id === temp_id);

			if (index !== -1) {
				const updated = { ...messagesStore.messages[index], logId: log_id };

				// Also update the next message (EGO's response) if it exists and looks like it belongs to this pair?
				// Usually the log_id applies to the user message. EGO's response is implicitly linked.
				// But we might want to tag EGO's response with the same logId for regeneration reference.

				messagesStore = {
					...messagesStore,
					messages: [
						...messagesStore.messages.slice(0, index),
						updated,
						...messagesStore.messages.slice(index + 1)
					]
				};

				// Check if the next message is EGO's optimistic thinking message
				if (index + 1 < messagesStore.messages.length) {
					const nextMsg = messagesStore.messages[index + 1];
					if (nextMsg.author === 'ego') {
						const updatedNext = { ...nextMsg, logId: log_id };
						messagesStore = {
							...messagesStore,
							messages: [
								...messagesStore.messages.slice(0, index + 1),
								updatedNext,
								...messagesStore.messages.slice(index + 2)
							]
						};
					}
				}
			}
		}
	},

	removeMessage(sessionUUID: string, messageId: number) {
		if (browser) {
			if (messagesStore.sessionUUID !== sessionUUID) {
				return;
			}
			messagesStore = {
				...messagesStore,
				messages: messagesStore.messages.filter((m) => m.id !== messageId)
			};
		}
	},

	clearMessages() {
		if (browser) {
			messagesStore = { sessionUUID: null, messages: [] };
		}
	},

	/**
	 * Atomically regenerate messages from a specific index
	 * Removes all messages after the specified index (inclusive) and returns them for rollback
	 */
	regenerateFrom(sessionUUID: string, fromIndex: number): ChatMessage[] {
		if (browser) {
			if (messagesStore.sessionUUID !== sessionUUID) {
				return [];
			}

			const messagesSnapshot = [...messagesStore.messages];
			const removedMessages = messagesSnapshot.slice(fromIndex);

			messagesStore = {
				...messagesStore,
				messages: messagesSnapshot.slice(0, fromIndex)
			};

			return removedMessages;
		}
		return [];
	},

	/**
	 * Create a snapshot of current messages for rollback
	 */
	createMessagesSnapshot(sessionUUID: string): ChatMessage[] | null {
		if (browser) {
			if (messagesStore.sessionUUID !== sessionUUID) {
				return null;
			}
			return [...messagesStore.messages];
		}
		return null;
	},

	/**
	 * Restore messages from a snapshot
	 */
	restoreMessagesFromSnapshot(sessionUUID: string, snapshot: ChatMessage[]) {
		if (browser) {
			if (messagesStore.sessionUUID !== sessionUUID) {
				return;
			}
			messagesStore = {
				sessionUUID,
				messages: snapshot
			};
		}
	},

	/**
	 * Migrate messages from one session UUID to another
	 * Used when a new session is created and we need to update the UUID
	 */
	migrateMessages(fromSessionUUID: string, toSessionUUID: string) {
		if (browser) {
			if (messagesStore.sessionUUID === fromSessionUUID) {
				console.log('[ChatStore] Migrating messages from', fromSessionUUID, 'to', toSessionUUID);
				messagesStore = {
					sessionUUID: toSessionUUID,
					messages: messagesStore.messages
				};
				// Update stream sessionUUID as well
				if (stream.sessionUUID === fromSessionUUID) {
					stream.sessionUUID = toSessionUUID;
				}
			}
		}
	},

	// UI Lock actions
	setRegenerating(value: boolean) {
		if (browser) {
			uiLock.isRegenerating = value;
		}
	},

	setEditingSaving(value: boolean) {
		if (browser) {
			uiLock.isEditingSaving = value;
		}
	},

	resetUILocks() {
		if (browser) {
			uiLock.isRegenerating = false;
			uiLock.isEditingSaving = false;
		}
	},

	// Stream actions
	startStream(sessionUUID: string | null) {
		if (browser) {
			const preservePlan = stream.sessionUUID === sessionUUID;
			stream = {
				thoughtHeader: '',
				thoughts: [],
				activeTools: [],
				textStream: '',
				isDone: false,
				error: '',
				sessionUUID,
				newlyCreatedSessionUUID: null,
				lastUserMessage: null,
				wasCancelled: false,
				currentLogId: null,
				isRecovering: false,
				recoveredFromRest: false,
				lastSeqNumber: -1,
				activePlan: preservePlan ? stream.activePlan : null
			};
		}
	},

	resetBuffer() {
		if (browser) {
			stream.textStream = '';
			stream.lastSeqNumber = -1;
		}
	},

	resume(sessionUUID: string, logId: number) {
		if (browser) {
			// If we are already streaming this session, ignore
			if (!stream.isDone && stream.sessionUUID === sessionUUID) return;

			console.log(`[ChatStore] Resuming stream for session ${sessionUUID}, log ${logId}`);
			stream = {
				thoughtHeader: '',
				thoughts: [],
				activeTools: [],
				textStream: '',
				isDone: false,
				error: '',
				sessionUUID,
				newlyCreatedSessionUUID: null,
				lastUserMessage: null,
				wasCancelled: false,
				currentLogId: logId,
				isRecovering: true,
				recoveredFromRest: true,
				lastSeqNumber: -1,
				activePlan: stream.sessionUUID === sessionUUID ? stream.activePlan : null
			};
		}
	},

	setThoughtHeader(header: string) {
		if (browser) {
			stream.thoughtHeader = header;
		}
	},

	setPlan(plan: SessionPlan | null) {
		if (browser) {
			console.log('[ChatStore] Setting plan:', plan);
			stream.activePlan = plan;
		}
	},

	appendThought(thought: ThoughtStep) {
		if (browser) {
			stream.thoughts = [...stream.thoughts, thought];
		}
	},

	updateThought(id: string, updates: Partial<ThoughtStep>) {
		if (browser) {
			const index = stream.thoughts.findIndex((t) => t.id === id);
			if (index !== -1) {
				const updated = { ...stream.thoughts[index], ...updates };
				stream.thoughts = [
					...stream.thoughts.slice(0, index),
					updated,
					...stream.thoughts.slice(index + 1)
				];
			}
		}
	},

	updateToolStatus(
		callId: string,
		toolName: string,
		status: ToolAction['status'],
		progress?: string,
		header?: string
	) {
		if (browser) {
			const index = stream.activeTools.findIndex((t) => t.callId === callId);
			if (index !== -1) {
				const updatedTool = { ...stream.activeTools[index], status, progress, header };
				stream.activeTools = [
					...stream.activeTools.slice(0, index),
					updatedTool,
					...stream.activeTools.slice(index + 1)
				];
			} else {
				stream.activeTools = [
					...stream.activeTools,
					{ callId, toolName, status, progress, header }
				];
			}
		}
	},

	appendToStream(chunk: string, seq?: number) {
		if (browser) {
			// If sequence number is provided, check for duplicates
			if (seq !== undefined) {
				if (seq <= stream.lastSeqNumber) {
					// skip duplicate
					return;
				}
				stream.lastSeqNumber = seq;
			}

			stream.textStream += chunk;
		}
	},

	finish() {
		if (browser) {
			stream.isDone = true;
			stream.isRecovering = false;

			this.applyFinalStreamResult(stream.textStream, false);

			// Final consumption to update UI
			setTimeout(() => {
				stream.currentLogId = null;
			}, 1500);
		}
	},

	endStream() {
		this.finish();
	},

	stopStreamAsCancelled() {
		if (browser) {
			stream.wasCancelled = true;
			stream.isDone = true;

			this.applyFinalStreamResult(stream.textStream, true);

			setTimeout(() => {
				stream.currentLogId = null;
			}, 1500);
		}
	},

	applyFinalStreamResult(text: string, cancelled: boolean) {
		if (!stream.sessionUUID || messagesStore.sessionUUID !== stream.sessionUUID) return;

		const messages = messagesStore.messages;
		let targetIndex = -1;

		if (stream.currentLogId) {
			targetIndex = messages.findIndex(
				(m) => m.logId === stream.currentLogId && m.author === 'ego'
			);
		}

		if (targetIndex === -1) {
			// Find last thinking ego message
			for (let i = messages.length - 1; i >= 0; i--) {
				if (messages[i].author === 'ego' && messages[i].isThinking) {
					targetIndex = i;
					break;
				}
			}
		}

		if (targetIndex !== -1) {
			const target = messages[targetIndex];
			const finalText = (text || target.text || '').trim();

			this.updateMessage(stream.sessionUUID, target.id, {
				text: finalText,
				thoughts: [...stream.thoughts],
				isThinking: false,
				isCancelled: cancelled
			});
		}

		stream.wasCancelled = false;
	},

	setStreamError(errorMessage: string) {
		if (browser) {
			stream.error = errorMessage;
			stream.isDone = true;
			stream.textStream = '';
		}
	},

	setLastUserMessageLogId(temp_id: number, log_id: number) {
		if (browser) {
			stream.lastUserMessage = { temp_id, log_id };
			stream.currentLogId = log_id;
		}
	},

	consumeLastUserMessageLogId() {
		if (browser) {
			const val = stream.lastUserMessage;
			stream.lastUserMessage = null;
			return val;
		}
		return null;
	},

	clearLastUserMessage() {
		if (browser) {
			stream.lastUserMessage = null;
		}
	},

	setNewlyCreatedSessionUUID(uuid: string | null) {
		if (browser) {
			stream.newlyCreatedSessionUUID = uuid;
		}
	},

	resetStream() {
		if (browser) {
			stream = {
				thoughtHeader: '',
				thoughts: [],
				activeTools: [],
				textStream: '',
				isDone: true,
				error: '',
				sessionUUID: null,
				newlyCreatedSessionUUID: null,
				lastUserMessage: null,
				wasCancelled: false,
				currentLogId: null,
				isRecovering: false,
				recoveredFromRest: false,
				lastSeqNumber: -1,
				activePlan: null,
				superEgoDebate: null
			};
		}
	},

	// ============================================================================
	// SUPEREGO ACTIONS
	// ============================================================================

	startSuperEgoRound(round: number, title: string) {
		if (browser) {
			if (!stream.superEgoDebate) {
				stream.superEgoDebate = {
					active: true,
					rounds: [],
					completed: false
				};
			}
			stream.superEgoDebate.rounds.push({
				number: round,
				title,
				agents: [],
				completed: false
			});
		}
	},

	startSuperEgoAgent(
		round: number,
		agentName: string,
		agentRole: 'researcher' | 'solver' | 'critic' | 'optimizer' | 'synthesizer'
	) {
		if (browser && stream.superEgoDebate) {
			const roundData = stream.superEgoDebate.rounds.find((r) => r.number === round);
			if (roundData) {
				roundData.agents.push({
					name: agentName,
					role: agentRole,
					status: 'running'
				});
			}
		}
	},

	updateSuperEgoAgentMessage(round: number, agentName: string, message: string) {
		if (browser && stream.superEgoDebate) {
			const roundData = stream.superEgoDebate.rounds.find((r) => r.number === round);
			if (roundData) {
				const agent = roundData.agents.find((a) => a.name === agentName);
				if (agent) {
					agent.message = message;
				}
			}
		}
	},

	completeSuperEgoAgent(round: number, agentName: string) {
		if (browser && stream.superEgoDebate) {
			const roundData = stream.superEgoDebate.rounds.find((r) => r.number === round);
			if (roundData) {
				const agent = roundData.agents.find((a) => a.name === agentName);
				if (agent) {
					agent.status = 'completed';
				}
			}
		}
	},

	errorSuperEgoAgent(round: number, agentName: string, error: string) {
		if (browser && stream.superEgoDebate) {
			const roundData = stream.superEgoDebate.rounds.find((r) => r.number === round);
			if (roundData) {
				const agent = roundData.agents.find((a) => a.name === agentName);
				if (agent) {
					agent.status = 'error';
					agent.error = error;
				}
			}
		}
	},

	completeSuperEgoRound(round: number) {
		if (browser && stream.superEgoDebate) {
			const roundData = stream.superEgoDebate.rounds.find((r) => r.number === round);
			if (roundData) {
				roundData.completed = true;
			}
		}
	},

	completeSuperEgoDebate(summary?: string) {
		if (browser && stream.superEgoDebate) {
			stream.superEgoDebate.completed = true;
			stream.superEgoDebate.active = false;
			if (summary) {
				stream.superEgoDebate.summary = summary;
			}
		}
	},

	tryHydrateFromStorage(): boolean {
		// Disabled hydration from localStorage for now as we rely on backend state
		return false;
	}
};
