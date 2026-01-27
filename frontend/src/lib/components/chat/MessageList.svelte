<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { tick } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { debounce } from '$lib/utils/debounce';
	import MessageItem from './MessageItem.svelte';
	import { fade, fly } from 'svelte/transition';
	import type { ChatMessage, StreamStoreState } from '$lib/types';

	let {
		messages = [],
		editingLogId = null,
		editingText = '',
		editingTextarea = null,
		streamStore,
		isMobile = false,
		onscroll,
		onstartedit,
		oncanceledit,
		onsaveandregen,
		onregenerate
	}: {
		messages: ChatMessage[];
		editingLogId: number | null;
		editingText: string;
		editingTextarea: HTMLTextAreaElement | null;
		streamStore: StreamStoreState;
		isMobile: boolean;
		onscroll: () => void;
		onstartedit: (message: ChatMessage) => void;
		oncanceledit: () => void;
		onsaveandregen: () => void;
		onregenerate: (message: ChatMessage) => void;
	} = $props();

	let chatContainer: HTMLElement | undefined = $state();
	let displayMessages = $derived(
		(() => {
			const out: ChatMessage[] = [];
			const seenIds = new SvelteSet<number>();

			for (const m of messages) {
				// Skip duplicate messages (by id)
				if (seenIds.has(m.id)) {
					console.warn('[MessageList] Skipping duplicate message with id:', m.id);
					continue;
				}
				seenIds.add(m.id);

				// Merge multiple thinking messages into one
				if (m.author === 'ego' && m.isThinking) {
					const last = out[out.length - 1];
					if (last && last.author === 'ego' && last.isThinking) {
						// Replace the last thinking message with this one (keep the most recent)
						out[out.length - 1] = m;
						seenIds.delete(last.id); // Remove old id from seen set
					} else {
						out.push(m);
					}
				} else {
					out.push(m);
				}
			}
			return out;
		})()
	);

	// Debounced scroll handler для оптимизации производительности
	const debouncedOnScroll = debounce(() => {
		if (onscroll && chatContainer) {
			if (onscroll) onscroll();
		}
	}, 50);

	export function scrollToBottom(behavior: 'smooth' | 'auto' = 'smooth') {
		tick().then(() => {
			if (chatContainer) {
				// Use set timeout to ensure DOM is updated and measured correctly
				setTimeout(() => {
					if (chatContainer) {
						chatContainer.scrollTo({
							top: chatContainer.scrollHeight,
							behavior
						});
					}
				}, 10);
			}
		});
	}
	export function isNearBottom(threshold = 100) {
		if (!chatContainer) return true;
		const distanceToBottom =
			chatContainer.scrollHeight - chatContainer.scrollTop - chatContainer.clientHeight;
		return distanceToBottom <= threshold;
	}
</script>

<div
	bind:this={chatContainer}
	onscroll={debouncedOnScroll}
	class="chat-container-scroll custom-scrollbar min-h-0 flex-1 overflow-y-auto px-4 pb-44 pt-24"
>
	<div class="relative mx-auto flex w-full max-w-4xl flex-col gap-2">
		{#if messages.length === 0}
			<div
				class="flex h-full min-h-[50vh] flex-col items-center justify-center text-center"
				in:fade={{ duration: 500 }}
			>
				<div class="group relative mb-6">
					<div
						class="absolute -inset-4 rounded-full bg-accent/20 blur-xl transition-all duration-500 group-hover:bg-accent/30 group-hover:blur-2xl"
					></div>
					<img
						src={getAppLogo(preferencesStore.theme)}
						alt="EGO Logo"
						class="relative h-24 w-24 transition-transform duration-700 ease-in-out group-hover:rotate-12 group-hover:scale-110"
					/>
				</div>
				<h2
					class="mb-3 bg-gradient-to-r from-text-primary to-text-secondary bg-clip-text text-4xl font-bold tracking-tight text-transparent"
					in:fly={{ y: 20, duration: 500, delay: 100 }}
				>
					{$_('chat.welcome_title') || 'EGO'}
				</h2>
				<p
					class="max-w-md text-lg text-text-secondary/80"
					in:fly={{ y: 20, duration: 500, delay: 200 }}
				>
					{$_('chat.landing_subtitle') ||
						'Выберите существующую сессию из списка слева или начните новую.'}
				</p>
			</div>
		{:else}
			{#each displayMessages as msg, i (msg.id)}
				<MessageItem
					message={msg}
					{editingLogId}
					{editingText}
					{editingTextarea}
					{streamStore}
					{isMobile}
					index={displayMessages.length - 1 - i}
					{onstartedit}
					{oncanceledit}
					{onsaveandregen}
					{onregenerate}
				/>
			{/each}
		{/if}
	</div>
</div>

<style>
	.custom-scrollbar::-webkit-scrollbar {
		width: 6px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background: rgba(156, 163, 175, 0.2);
		border-radius: 10px;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: rgba(156, 163, 175, 0.4);
	}
</style>
