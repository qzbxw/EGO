<script lang="ts">
	import { marked } from 'marked';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	try {
		marked.setOptions({ gfm: true, breaks: true });
	} catch (e) {
		console.debug('Failed to set marked options:', e);
	}
	let {
		text = '',
		invert = false,
		isDone = false
	} = $props<{ text?: string; invert?: boolean; isDone?: boolean }>();

	let container: HTMLDivElement | undefined = $state();
	async function renderContent() {
		try {
			if (!container) return;
			const src = text || '';
			if (!src) {
				container.innerHTML = ''; // eslint-disable-line svelte/no-dom-manipulating
				return;
			}
			const html = await marked(src, { async: true });
			if (!container) return;

			// We use a temporary element to compare or just apply classes to new children
			container.innerHTML = html; // eslint-disable-line svelte/no-dom-manipulating
		} catch (err) {
			console.error('[StreamingMarkdownLite] render error', err);
		}
	}
	let __renderTimer: number | null = null;
	let __lastRenderAt = 0;
	function scheduleRender() {
		const now = performance.now();
		const minInterval = 16; // 60fps for much smoother streaming
		const elapsed = now - __lastRenderAt;
		if (elapsed >= minInterval) {
			__lastRenderAt = now;
			if (__renderTimer) {
				clearTimeout(__renderTimer);
				__renderTimer = null;
			}
			void renderContent().catch((e) =>
				console.error('[StreamingMarkdownLite] renderContent rejected', e)
			);
		} else {
			const wait = Math.max(8, minInterval - elapsed);
			if (__renderTimer) clearTimeout(__renderTimer);
			__renderTimer = window.setTimeout(() => {
				__lastRenderAt = performance.now();
				void renderContent().catch((e) =>
					console.error('[StreamingMarkdownLite] renderContent rejected', e)
				);
				__renderTimer = null;
			}, wait);
		}
	}
	$effect(() => {
		if (!container) return;
		void text;
		scheduleRender();
	});
	$effect(() => () => {
		if (__renderTimer) {
			clearTimeout(__renderTimer);
			__renderTimer = null;
		}
	});
</script>

<div
	bind:this={container}
	class="streaming-content prose w-full min-w-0 max-w-full break-words selection:bg-accent/30"
	class:prose-invert={invert || preferencesStore.theme === 'dark'}
	class:is-streaming={!isDone && text.length > 0}
	aria-live="polite"
></div>

<style>
	:global(.streaming-content) {
		user-select: text !important;
		-webkit-user-select: text !important;
		font-variant-numeric: tabular-nums;
	}
	/* Removed jumpy text-reveal animation for cleaner output */

	:global(.streaming-content.is-streaming > *:last-child::after) {
		content: '';
		display: inline-block;
		width: 6px;
		height: 1.1em;
		background-color: var(--color-accent);
		margin-left: 2px;
		vertical-align: -0.2em; /* Align better with baseline */
		border-radius: 1px;
		opacity: 0.8;
		animation: cursor-pulse 1s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes cursor-pulse {
		0%,
		100% {
			opacity: 0.8;
			transform: scaleY(1);
		}
		50% {
			opacity: 0.3;
			transform: scaleY(0.9);
		}
	}

	:global(.prose pre) {
		overflow-x: auto;
		-webkit-overflow-scrolling: touch;
		word-break: break-word;
	}
	:global(.prose code) {
		word-break: break-word;
	}
</style>
