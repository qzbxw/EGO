import { browser } from '$app/environment';
interface InFlightState {
	sessionUUID: string | null;
	text: string;
	files: File[];
}
let state = $state<InFlightState | null>(null);
export const inFlightStore = {
	get subscribe() {
		return {
			subscribe(callback: (value: InFlightState | null) => void) {
				if (browser) {
					$effect(() => {
						callback(state);
					});
				} else {
					callback(state);
				}
				return () => {};
			}
		};
	},
	set: (sessionUUID: string, text: string, files: File[]) => {
		state = { sessionUUID, text, files };
	},
	clear: () => {
		state = null;
	},
	get: (): InFlightState | null => {
		return state;
	}
};
