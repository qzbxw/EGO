<script lang="ts">
	import { api } from '$lib/api';
	let { src, alt, ...rest } = $props<{ src: string; alt: string; [key: string]: any }>();
	const imageUrlCache: Map<string, string> = window.__egoImageUrlCache || new Map();
	const imageInflight: Map<string, Promise<string>> = window.__egoImageInflight || new Map();
	window.__egoImageUrlCache = imageUrlCache;
	window.__egoImageInflight = imageInflight;
	let objectUrl = $state<string | null>(null);
	let createdByUs = $state(false);
	let lastFetchedSrc: string | null = null;
	let lastRevocableUrl: string | null = null;
	async function fetchImage(url: string) {
		if (!url) return;
		try {
			if (url.startsWith('blob:') || url.startsWith('data:')) {
				objectUrl = url;
				createdByUs = false;
				return;
			}
			const cached = imageUrlCache.get(url);
			if (cached) {
				objectUrl = cached;
				createdByUs = false;
				return;
			}
			let promise = imageInflight.get(url);
			if (!promise) {
				promise = (async () => {
					const blob = await api.getBlob(url);
					const newObjUrl = URL.createObjectURL(blob);
					imageUrlCache.set(url, newObjUrl);
					return newObjUrl;
				})();
				imageInflight.set(url, promise);
			}
			const newObjUrl = await promise;
			objectUrl = newObjUrl;
			createdByUs = false;
			lastRevocableUrl = null;
			imageInflight.delete(url);
		} catch (error) {
			console.error('Failed to fetch authenticated image:', error);
			objectUrl = null;
			createdByUs = false;
			try {
				imageInflight.delete(url);
			} catch {}
		}
	}
	$effect(() => {
		if (src === lastFetchedSrc) return;
		if (lastRevocableUrl) {
			try {
				URL.revokeObjectURL(lastRevocableUrl);
			} catch {}
			lastRevocableUrl = null;
		}
		lastFetchedSrc = src;
		fetchImage(src);
		return () => {
			if (lastRevocableUrl) {
				try {
					URL.revokeObjectURL(lastRevocableUrl);
				} catch {}
				lastRevocableUrl = null;
			}
		};
	});
</script>

{#if objectUrl}
	<img src={objectUrl} {alt} {...rest} />
{:else}
	<div class="h-full w-full animate-pulse rounded-xl bg-tertiary"></div>
{/if}
