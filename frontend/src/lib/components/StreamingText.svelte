<script lang="ts">
	let { text, cps = 60 } = $props<{ text: string; cps?: number }>();
	let container: HTMLDivElement | undefined = $state();
	let committed = $state('');
	let pending = $state('');
	let rafId = $state<number | null>(null);
	let lastTs = $state(0);
	let charBudget = $state(0);
	let maxPerFrame = $state(3);
	$effect(() => {
		maxPerFrame = Math.max(2, Math.round(cps / 20));
	});
	function start() {
		if (rafId === null) rafId = requestAnimationFrame(tick);
	}
	function stop() {
		if (rafId !== null) cancelAnimationFrame(rafId);
		rafId = null;
		renderCursor(true);
	}
	function renderCursor(_on: boolean) {}
	function appendChars(count: number) {
		if (!container || count <= 0 || pending.length === 0) return;
		const take = Math.min(count, pending.length);
		const chunk = pending.slice(0, take);
		pending = pending.slice(take);
		committed += chunk;
		container.textContent = committed;
	}
	function tick(ts: number) {
		if (lastTs === 0) lastTs = ts;
		const dt = (ts - lastTs) / 1000;
		charBudget += dt * cps;
		let burst = Math.floor(charBudget);
		if (burst > 0) {
			burst = Math.min(burst, maxPerFrame);
			appendChars(burst);
			charBudget -= burst;
		}
		lastTs = ts;
		if (pending.length > 0) {
			rafId = requestAnimationFrame(tick);
		} else {
			rafId = null;
		}
	}
	function enqueueDelta(nextText: string) {
		if (nextText.startsWith(committed)) {
			pending += nextText.slice(committed.length);
		} else {
			committed = '';
			pending = nextText;
			charBudget = 0;
			lastTs = 0;
			if (container) container.textContent = '';
		}
		start();
	}
	$effect(() => {
		if (!container) return;
		enqueueDelta(text || '');
	});
	$effect(() => () => {
		if (rafId) cancelAnimationFrame(rafId);
	});
</script>

<div class="streaming-type" bind:this={container} aria-live="polite"></div>

<style>
	.streaming-type {
		white-space: pre-wrap;
		word-break: break-word;
		contain: content;
		will-change: contents;
		transition: opacity 120ms ease;
	}
	.streaming-type::after {
		content: '';
		display: inline-block;
		width: 0.55ch;
		height: 1em;
		margin-left: 1px;
		background: currentColor;
		opacity: 0.35;
		animation: blink 1.1s steps(1, end) infinite;
		vertical-align: -0.15em;
		border-radius: 2px;
	}
	@keyframes blink {
		50% {
			opacity: 0.05;
		}
	}
</style>
