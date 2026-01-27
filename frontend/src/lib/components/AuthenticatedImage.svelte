<script lang="ts">
	import { api } from '$lib/api';
	import { SvelteMap } from 'svelte/reactivity';
	let { src, alt, ...rest } = $props<{ src: string; alt: string; [key: string]: unknown }>();
	const imageUrlCache = window.__egoImageUrlCache || new SvelteMap<string, string>();
	const imageInflight = window.__egoImageInflight || new SvelteMap<string, Promise<string>>();
	window.__egoImageUrlCache = imageUrlCache;
	window.__egoImageInflight = imageInflight;
	let objectUrl = $state<string | null>(null);
	let lastFetchedSrc: string | null = null;
	let lastRevocableUrl: string | null = null;
	async function fetchImage(url: string) {
		if (!url) return;
		try {
			if (url.startsWith('blob:') || url.startsWith('data:')) {
				objectUrl = url;
				return;
			}
			const cached = imageUrlCache.get(url);
			if (cached) {
				objectUrl = cached;
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
			lastRevocableUrl = null;
			imageInflight.delete(url);
		} catch (error) {
			console.error('Failed to fetch authenticated image:', error);
			objectUrl = null;
			try {
				imageInflight.delete(url);
			} catch {
				// ignore
			}
		}
	}
	$effect(() => {
		if (src === lastFetchedSrc) return;
		if (lastRevocableUrl) {
			try {
				URL.revokeObjectURL(lastRevocableUrl);
			} catch {
				// ignore
			}
			lastRevocableUrl = null;
		}
		lastFetchedSrc = src;
		fetchImage(src);
		return () => {
			if (lastRevocableUrl) {
				try {
					URL.revokeObjectURL(lastRevocableUrl);
				} catch {
					// ignore
				}
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
