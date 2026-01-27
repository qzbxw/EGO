import { SvelteSet } from 'svelte/reactivity';
import type { CachedFile } from '$lib/types';
import { browser } from '$app/environment';
interface CachedFilesStore {
	sessionUUID: string | null;
	files: CachedFile[];
}
let store = $state<CachedFilesStore>({ sessionUUID: null, files: [] });
export const cachedFiles = {
	get subscribe() {
		return {
			subscribe(callback: (value: CachedFilesStore) => void) {
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
	setFiles: (sessionUUID: string, files: CachedFile[]) => {
		store = { sessionUUID, files };
	},
	addFiles: (sessionUUID: string, newFiles: CachedFile[]) => {
		if (store.sessionUUID !== sessionUUID) {
			store = { sessionUUID, files: newFiles };
			return;
		}
		const existing = new SvelteSet(store.files.map((f) => `${f.file_name}-${f.mime_type}`));
		const filtered = newFiles.filter((f) => !existing.has(`${f.file_name}-${f.mime_type}`));
		store = { ...store, files: [...store.files, ...filtered] };
	},
	clear: () => {
		store = { sessionUUID: null, files: [] };
	}
};
