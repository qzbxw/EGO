import type { ChatSession } from '$lib/types';
import { writable } from 'svelte/store';
const sessionList = writable<ChatSession[]>([]);
const loading = writable<boolean>(true);
export const sessions = sessionList;
export const isLoading = loading;
export function setInitialSessions(initialSessions: ChatSession[]) {
    sessionList.set(Array.isArray(initialSessions) ? initialSessions : []);
    loading.set(false);
}
export function addSession(newSession: ChatSession) {
    sessionList.update((list: ChatSession[]) => {
        if (list.some((s) => s.uuid === newSession.uuid)) {
            return list;
        }
        return [newSession, ...list];
    });
}
export function updateSession(updatedSession: Partial<ChatSession>) {
    sessionList.update((list: ChatSession[]) => {
        const index = list.findIndex((s) => s.uuid === updatedSession.uuid);
        if (index === -1) return list;
        const copy = list.slice();
        copy[index] = { ...copy[index], ...updatedSession } as ChatSession;
        return copy;
    });
}
export function removeSession(sessionUUID: string) {
    sessionList.update((list: ChatSession[]) => list.filter((s) => s.uuid !== sessionUUID));
}
export function clearUserSessions() {
    sessionList.set([]);
    loading.set(true);
}
export function bringSessionToTop(sessionUUID: string) {
    sessionList.update((list: ChatSession[]) => {
        const sessionIndex = list.findIndex((s) => s.uuid === sessionUUID);
        if (sessionIndex > 0) {
            const copy = list.slice();
            const [sessionToMove] = copy.splice(sessionIndex, 1);
            copy.unshift(sessionToMove);
            return copy;
        }
        return list;
    });
}
let currentSessionList: ChatSession[] = [];
sessionList.subscribe((list: ChatSession[]) => currentSessionList = list);
export function getSessionById(sessionUUID: string): ChatSession | undefined {
    return currentSessionList.find((s) => s.uuid === sessionUUID);
}