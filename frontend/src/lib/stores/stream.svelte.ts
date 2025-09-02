import { setIsThinking } from '$lib/stores/ui.svelte.ts';
import { writable } from 'svelte/store';
import { browser } from '$app/environment';
export const textStreamStore = writable<string>('');
let thoughtHeader = $state<string>('');
let textStream = $state<string>('');
let isDone = $state<boolean>(true);
let error = $state<string>('');
let streamingSessionUUID = $state<string | null>(null);
let newlyCreatedSessionUUID = $state<string | null>(null); 
export const newlyCreatedSessionUUIDStore = writable<string | null>(null);
let lastUserMessage = $state<{ temp_id: number; log_id: number } | null>(null);
let consumedText = $state<string | null>(null);
let wasCancelled = $state<boolean>(false);
let currentLogId = $state<number | null>(null);
let isRecovering = $state<boolean>(false);
let recoveredFromRest = $state<boolean>(false);
type PersistedStreamState = {
  sessionUUID: string;
  textStream: string;
  isDone: boolean;
  thoughtHeader: string;
  currentLogId: number | null;
  ts: number;
};
function storageKey(sessionUUID: string) {
  return `ego_stream_${sessionUUID}`;
}
function persistState() {
  if (!browser) return;
  if (!streamingSessionUUID) return;
  const payload: PersistedStreamState = {
    sessionUUID: streamingSessionUUID,
    textStream,
    isDone,
    thoughtHeader,
    currentLogId,
    ts: Date.now()
  };
  try {
    localStorage.setItem(storageKey(streamingSessionUUID), JSON.stringify(payload));
  } catch {}
}
function clearPersistedState(sessionUUID?: string | null) {
  if (!browser) return;
  const key = sessionUUID || streamingSessionUUID;
  if (!key) return;
  try { localStorage.removeItem(storageKey(key)); } catch {}
}
function readPersistedState(sessionUUID: string): PersistedStreamState | null {
  if (!browser) return null;
  try {
    const raw = localStorage.getItem(storageKey(sessionUUID));
    if (!raw) return null;
    const data = JSON.parse(raw) as PersistedStreamState;
    if (data && data.sessionUUID === sessionUUID) return data;
  } catch {}
  return null;
}
export const streamStore = {
	get thoughtHeader() { return thoughtHeader; },
	set thoughtHeader(value) { thoughtHeader = value; },
	get textStream() { return textStream; },
	set textStream(value) { textStream = value; },
	get isDone() { return isDone; },
	set isDone(value) { isDone = value; },
	get error() { return error; },
	set error(value) { error = value; },
	get sessionUUID() { return streamingSessionUUID; },
	set sessionUUID(value) { streamingSessionUUID = value; },
	get newlyCreatedSessionUUID() { return newlyCreatedSessionUUID; },
	set newlyCreatedSessionUUID(value) { newlyCreatedSessionUUID = value; newlyCreatedSessionUUIDStore.set(value); },
    get lastUserMessage() { return lastUserMessage; },
    get currentLogId() { return currentLogId; },
    set currentLogId(value) { currentLogId = value; },
    get isRecovering() { return isRecovering; },
    set isRecovering(value) { isRecovering = value; },
    get recoveredFromRest() { return recoveredFromRest; },
    set recoveredFromRest(value) { recoveredFromRest = value; }
};
export function startStream(sessionUUID: string | null) {
	thoughtHeader = '';
	textStream = '';
	textStreamStore.set('');
	isDone = false;
	error = '';
	consumedText = null;
	wasCancelled = false;
	streamingSessionUUID = sessionUUID;
	newlyCreatedSessionUUID = null; 
    currentLogId = null;
    isRecovering = false;
    recoveredFromRest = false;
    newlyCreatedSessionUUIDStore.set(null);
	setIsThinking(true);
    try { console.debug('[stream] startStream', { sessionUUID }); } catch {}
    if (sessionUUID) persistState();
}
export function setThoughtHeader(header: string) {
	thoughtHeader = header;
    try { console.debug('[stream] setThoughtHeader', { header }); } catch {}
    persistState();
}
export function appendToStream(chunk: string) {
  if (recoveredFromRest && !isRecovering) return;
  textStream += chunk;
  textStreamStore.set(textStream);
  try { console.debug('[stream] appendToStream', { len: chunk?.length ?? 0, total: textStream.length }); } catch {}
  persistState();
}
export function endStream() {
  isDone = true;
  consumedText = textStream;
  textStreamStore.set(textStream);
  try { console.debug('[stream] endStream', { hasText: !!textStream, header: thoughtHeader, currentLogId }); } catch {}
  
  // Delay clearing currentLogId to prevent bubble messages from disappearing too early
  setTimeout(() => {
    currentLogId = null;
    clearPersistedState();
  }, 1500);
}
export function stopStreamAsCancelled() {
	wasCancelled = true;
	endStream();
	setIsThinking(false);
}
export function consumeStreamResult(): { text: string; sessionUUID: string | null; cancelled: boolean } | null {
    if (consumedText !== null) {
        const result = { text: consumedText, sessionUUID: streamingSessionUUID, cancelled: wasCancelled };
        consumedText = null;
        wasCancelled = false;
        return result;
    }
    return null;
}
export function setStreamError(errorMessage: string) {
	error = errorMessage;
	isDone = true;
	streamingSessionUUID = null;
	textStreamStore.set('');
	setIsThinking(false);
	clearPersistedState();
    try { console.debug('[stream] setStreamError', { errorMessage }); } catch {}
}
export function setLastUserMessageLogId(temp_id: number, log_id: number) {
	console.log('[StreamStore] Setting lastUserMessage:', { temp_id, log_id });
	lastUserMessage = { temp_id, log_id };
    currentLogId = log_id;
    persistState();
}
export function consumeLastUserMessageLogId() {
	const val = lastUserMessage;
	lastUserMessage = null;
	return val;
}
export function resetStreamStore() {
    const prevUUID = streamingSessionUUID;
    thoughtHeader = '';
    textStream = '';
    textStreamStore.set('');
    isDone = true;
    error = '';
    lastUserMessage = null;
    consumedText = null;
    wasCancelled = false;
    newlyCreatedSessionUUID = null;
    newlyCreatedSessionUUIDStore.set(null);
    currentLogId = null;
    isRecovering = false;
    recoveredFromRest = false;
    setIsThinking(false);
    clearPersistedState(prevUUID);
    streamingSessionUUID = null;
}
export function tryHydrateFromStorage(sessionUUID: string): boolean {
  const data = readPersistedState(sessionUUID);
  if (!data) return false;
  streamingSessionUUID = data.sessionUUID;
  textStream = data.textStream || '';
  textStreamStore.set(textStream);
  isDone = Boolean(data.isDone);
  thoughtHeader = data.thoughtHeader || '';
  currentLogId = data.currentLogId ?? null;
  if (!isDone) setIsThinking(true);
  return true;
}