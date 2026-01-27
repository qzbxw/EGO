<script lang="ts">
	import { canvasStore } from '$lib/stores/canvas.svelte.ts';
	import { X, Play, Code as CodeIcon, RefreshCw } from 'lucide-svelte';
	import { fade } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import CodeBlock from './CodeBlock.svelte';

	let iframeKey = $state(0);

	function close() {
		canvasStore.close();
	}

	function reload() {
		iframeKey += 1;
	}

	let isHtml = $derived(canvasStore.language === 'html' || canvasStore.language === 'svg');

	// Create a safe blob URL for the iframe if needed, or just use srcdoc
	let srcDoc = $derived(canvasStore.content);
</script>

<div
	class="flex h-full w-full flex-col overflow-hidden bg-gradient-to-br from-secondary/95 to-primary/95 shadow-2xl backdrop-blur-3xl"
>
	<!-- Header -->
	<div
		class="flex items-center justify-between border-b border-white/5 bg-white/5 px-4 py-3 backdrop-blur-xl transition-colors duration-300 hover:bg-white/10"
	>
		<div class="flex items-center gap-3 overflow-hidden">
			<div
				class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-white/10 to-white/5 shadow-inner ring-1 ring-white/10"
			>
				{#if isHtml}
					<Play class="h-4 w-4 text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.5)]" />
				{:else}
					<CodeIcon class="h-4 w-4 text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.5)]" />
				{/if}
			</div>
			<div class="flex flex-col">
				<span class="max-w-[200px] truncate text-sm font-bold tracking-tight text-text-primary"
					>{canvasStore.title}</span
				>
				<span class="text-[10px] font-medium uppercase tracking-wider text-text-secondary"
					>{canvasStore.language}</span
				>
			</div>
		</div>

		<div class="flex items-center gap-2">
			{#if isHtml}
				<button
					class="group relative flex items-center justify-center rounded-lg p-2 text-text-secondary transition-all hover:bg-white/10 hover:text-text-primary active:scale-95"
					onclick={reload}
					title="Reload Preview"
				>
					<RefreshCw class="h-4 w-4 transition-transform duration-500 group-hover:rotate-180" />
				</button>
			{/if}

			<div class="mx-1 h-4 w-[1px] bg-white/10"></div>

			<button
				class="group flex items-center justify-center rounded-lg p-2 text-text-secondary transition-all hover:bg-red-500/10 hover:text-red-400 active:scale-95"
				onclick={close}
				title="Close Canvas"
			>
				<X class="h-4 w-4 transition-transform duration-300 group-hover:scale-110" />
			</button>
		</div>
	</div>

	<!-- Content -->
	<div class="relative flex-1 overflow-hidden bg-black/20">
		{#key canvasStore.content}
			<div class="h-full w-full" in:fade={{ duration: 300, easing: cubicOut }}>
				{#if isHtml}
					{#key iframeKey}
						<iframe
							srcdoc={srcDoc}
							title="Preview"
							class="h-full w-full border-0 bg-white shadow-inner"
							sandbox="allow-scripts allow-modals allow-forms allow-popups allow-same-origin allow-top-navigation-by-user-activation allow-downloads-without-user-activation"
						></iframe>
					{/key}
				{:else}
					<div class="custom-scrollbar h-full overflow-auto p-4">
						<CodeBlock code={canvasStore.content} language={canvasStore.language} inCanvas={true} />
					</div>
				{/if}
			</div>
		{/key}
	</div>
</div>

<style>
	.custom-scrollbar::-webkit-scrollbar {
		width: 8px;
		height: 8px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background: rgba(255, 255, 255, 0.1);
		border-radius: 4px;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: rgba(255, 255, 255, 0.2);
	}
</style>
