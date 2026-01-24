import { browser } from '$app/environment';

let memoryEnabled = $state<boolean>(true);
export const memoryStore = {
	get memoryEnabled() {
		return memoryEnabled;
	},
	setMemoryEnabled(enabled: boolean) {
		if (browser) {
			memoryEnabled = enabled;
		}
	}
};
