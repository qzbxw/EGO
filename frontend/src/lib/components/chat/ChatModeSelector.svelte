<script lang="ts">
	import { MoreHorizontal } from '@lucide/svelte';
	import { slide, fade } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import { getModeLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import type { ChatMode } from '$lib/types';
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
	}
</script>
{#if showInputOptions}
	<div
		role="button"
		tabindex="0"
		onkeydown={(e) => {
			if (e.key === 'Enter' || e.key === ' ') {
				showInputOptions = false;
			}
		}}
		class="fixed inset-0 z-10"
		onclick={() => (showInputOptions = false)}
		transition:fade={{ duration: 200 }}
	></div>
	<div
		class="absolute bottom-full mb-2 w-full bg-secondary p-2 rounded-xl shadow-lg z-20 border border-accent/30"
		transition:slide={{ duration: 150, easing: quintOut }}
	>
		<div class="flex flex-col gap-1">
			<button
				onclick={() => {
					toggleMode('deeper');
					showInputOptions = false;
				}}
				class="flex w-full items-center gap-3 p-3 text-left rounded-lg transition-colors {chatMode === 'deeper' ? 'bg-accent' : 'hover:bg-primary/50'}"
			>
				<img src={getModeLogo('deeper', preferencesStore.theme)} alt="Deeper" class="w-8 h-8" />
				<span class="font-medium">Deeper</span>
			</button>
			<button
				onclick={() => {
					toggleMode('research');
					showInputOptions = false;
				}}
				class="flex w-full items-center gap-3 p-3 text-left rounded-lg transition-colors {chatMode === 'research' ? 'bg-accent' : 'hover:bg-primary/50'}"
			>
				<img src={getModeLogo('research', preferencesStore.theme)} alt="Research" class="w-8 h-8" />
				<span class="font-medium">Research</span>
			</button>
			<button
				onclick={() => {
					toggleMode('agent');
					showInputOptions = false;
				}}
				class="flex w-full items-center gap-3 p-3 text-left rounded-lg transition-colors {chatMode === 'agent' ? 'bg-accent' : 'hover:bg-primary/50'}"
			>
				<img src={getModeLogo('agent', preferencesStore.theme)} alt="Agent" class="w-8 h-8" />
				<span class="font-medium">Agent</span>
			</button>
		</div>
	</div>
{/if}
{#if !isMobile}
	<div class="flex items-center gap-1 ml-2">
		<button
			onclick={() => toggleMode('deeper')}
			class="flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors"
			class:bg-accent={chatMode === 'deeper'}
			class:text-white={chatMode === 'deeper'}
			class:hover:bg-tertiary={chatMode !== 'deeper'}
		>
			<img src={getModeLogo('deeper', preferencesStore.theme)} alt="Deeper" class="w-8 h-8" />
			<span class="font-medium">Deeper</span>
		</button>
		<button
			onclick={() => toggleMode('research')}
			class="flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors"
			class:bg-accent={chatMode === 'research'}
			class:text-white={chatMode === 'research'}
			class:hover:bg-tertiary={chatMode !== 'research'}
		>
			<img src={getModeLogo('research', preferencesStore.theme)} alt="Research" class="w-8 h-8" />
			<span class="font-medium">Research</span>
		</button>
		<button
			onclick={() => toggleMode('agent')}
			class="flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors"
			class:bg-accent={chatMode === 'agent'}
			class:text-white={chatMode === 'agent'}
			class:hover:bg-tertiary={chatMode !== 'agent'}
		>
			<img src={getModeLogo('agent', preferencesStore.theme)} alt="Agent" class="w-8 h-8" />
			<span class="font-medium">Agent</span>
		</button>
	</div>
{:else}
	{#if chatMode === 'deeper'}
		<button
			onclick={() => toggleMode('deeper')}
			class="flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md bg-accent text-white"
		>
			<img src={getModeLogo('deeper', preferencesStore.theme)} alt="Deeper" class="w-8 h-8" />
			<span class="font-medium">Deeper</span>
		</button>
	{:else if chatMode === 'research'}
		<button
			onclick={() => toggleMode('research')}
			class="flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md bg-accent text-white"
		>
			<img src={getModeLogo('research', preferencesStore.theme)} alt="Research" class="w-8 h-8" />
			<span class="font-medium">Research</span>
		</button>
	{:else if chatMode === 'agent'}
		<button
			onclick={() => toggleMode('agent')}
			class="flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md bg-accent text-white"
		>
			<img src={getModeLogo('agent', preferencesStore.theme)} alt="Agent" class="w-8 h-8" />
			<span class="font-medium">Agent</span>
		</button>
	{/if}
	<button
		onclick={() => (showInputOptions = !showInputOptions)}
		class="p-2 rounded-md hover:bg-tertiary cursor-pointer transition-colors"
	>
		<MoreHorizontal class="w-5 h-5 text-text-secondary" />
	</button>
{/if}