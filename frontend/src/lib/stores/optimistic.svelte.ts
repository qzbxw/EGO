import type { ChatMessage } from '$lib/types';
let optimisticData = $state<{ messages: ChatMessage[] } | null>(null);
export const optimisticStore = {
	get data() {
		return optimisticData;
	},
	capture(messages: ChatMessage[]) {
		optimisticData = { messages };
	},
	clear() {
		optimisticData = null;
	}
};
