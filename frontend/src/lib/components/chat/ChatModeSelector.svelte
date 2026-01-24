<script lang="ts">
	import { slide, fade, scale } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { getModeLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import type { ChatMode } from '$lib/types';
	import { ChevronUp } from '@lucide/svelte';

	let {
		chatMode = $bindable('default' as ChatMode),
		isMobile = false,
		showInputOptions = $bindable(false)
	}: {
		chatMode: ChatMode;
		isMobile: boolean;
		showInputOptions: boolean;
	} = $props();

	function toggleMode(mode: ChatMode) {
		chatMode = chatMode === mode ? 'default' : mode;
		showInputOptions = false;
	}

	const modes: { id: ChatMode; label: string }[] = [
		{ id: 'default', label: 'Adaptive' },
		{ id: 'deeper', label: 'Deeper' },
		{ id: 'research', label: 'Research' },
		{ id: 'agent', label: 'Agent' }
	];
</script>

<div class="relative">
	<!-- Backdrop for clicking outside -->
	{#if showInputOptions}
		<div
			role="button"
			tabindex="0"
			class="fixed inset-0 z-10 cursor-default"
			onclick={() => (showInputOptions = false)}
			onkeydown={(e) => {
				if (e.key === 'Enter' || e.key === ' ') showInputOptions = false;
			}}
		></div>
	{/if}

	<!-- Dropdown Menu -->
	{#if showInputOptions}
		<div
			class="absolute bottom-full left-0 z-20 mb-3 min-w-[160px] overflow-hidden rounded-2xl border border-white/10 bg-secondary/80 p-1.5 shadow-[0_10px_40px_rgba(0,0,0,0.25)] backdrop-blur-2xl"
			transition:scale={{ duration: 200, start: 0.95, easing: cubicOut, transformOrigin: 'bottom left' }}
		>
			<div class="flex flex-col gap-0.5">
				{#each modes as mode}
					<button
						onclick={() => toggleMode(mode.id)}
						class="group flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left transition-all duration-200
                        {chatMode === mode.id
							? 'bg-accent text-white shadow-md shadow-accent/20'
							: 'text-text-secondary hover:bg-white/10 hover:text-text-primary'}"
					>
						<div class="relative h-5 w-5 shrink-0">
							<img
								src={getModeLogo(mode.id, preferencesStore.theme)}
								alt={mode.label}
								class="h-full w-full object-contain transition-transform duration-200 group-hover:scale-110"
							/>
						</div>
						<span class="text-sm font-medium">{mode.label}</span>
						{#if chatMode === mode.id}
							<div class="ml-auto h-1.5 w-1.5 rounded-full bg-white shadow-[0_0_8px_white]"></div>
						{/if}
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Trigger Button -->
	<button
		onclick={() => (showInputOptions = !showInputOptions)}
		class="group flex items-center gap-1.5 rounded-xl border border-transparent bg-tertiary/50 pl-2 pr-1.5 py-1.5 transition-all duration-200 hover:bg-tertiary hover:border-white/5 active:scale-95"
		class:bg-accent={showInputOptions}
		class:text-white={showInputOptions}
		title="Select Model Mode"
	>
		<img
			src={getModeLogo(chatMode, preferencesStore.theme)}
			alt={chatMode}
			class="h-5 w-5 object-contain transition-transform duration-300 group-hover:rotate-12"
		/>
		<ChevronUp
			class="h-3 w-3 text-text-secondary opacity-50 transition-transform duration-300 group-hover:text-text-primary {showInputOptions
				? 'rotate-180 text-white opacity-100'
				: ''}"
		/>
	</button>
</div>
