import type { ChatMessage } from '$lib/types';
import { browser } from '$app/environment';
interface ChatMessagesStore {
  sessionUUID: string | null;
  messages: ChatMessage[];
}
let store = $state<ChatMessagesStore>({ sessionUUID: null, messages: [] });
export const chatMessages = {
  get subscribe() {
    return {
      subscribe(callback: (value: ChatMessagesStore) => void) {
        if (browser) {
          $effect(() => {
            callback(store);
          });
        } else {
          callback(store);
        }
        return () => {};
      }
    };
  },
  get: () => store,
  setMessages: (sessionUUID: string, messages: ChatMessage[]) => {
    store = { sessionUUID, messages };
  },
  addMessage: (sessionUUID: string, message: ChatMessage) => {
    if (store.sessionUUID !== sessionUUID) {
      console.warn('ChatMessagesStore: adding message to a different session.');
      store = { sessionUUID, messages: [message] };
      return;
    }
    store = { ...store, messages: [...store.messages, message] };
  },
  addOptimisticMessages: (sessionUUID: string, messages: ChatMessage[]) => {
    if (store.sessionUUID !== sessionUUID && store.sessionUUID !== null) {
      console.warn('ChatMessagesStore: adding optimistic messages to a different session.');
      store = { sessionUUID, messages };
    } else {
      store = { sessionUUID, messages: [...store.messages, ...messages] };
    }
  },
  updateMessage: (sessionUUID: string, messageId: number, updates: Partial<ChatMessage>) => {
    if (store.sessionUUID !== sessionUUID) {
      return;
    }
    const index = store.messages.findIndex((m) => m.id === messageId);
    if (index !== -1) {
      const newMessages = [...store.messages];
      newMessages[index] = { ...newMessages[index], ...updates };
      store = { ...store, messages: newMessages };
    }
  },
  removeMessage: (sessionUUID: string, messageId: number) => {
    if (store.sessionUUID !== sessionUUID) {
      return;
    }
    store = { ...store, messages: store.messages.filter((m: ChatMessage) => m.id !== messageId) };
  },
  clear: () => {
    store = { sessionUUID: null, messages: [] };
  }
};