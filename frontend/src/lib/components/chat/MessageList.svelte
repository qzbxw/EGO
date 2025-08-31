<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { page } from '$app/stores';
	import { tick } from 'svelte';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import MessageItem from './MessageItem.svelte';
	import type { ChatMessage } from '$lib/types';
	let {
		messages = [],
		editingLogId = null,
		editingText = '',
		editingTextarea = null,
		lastMessageLogId = null,
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
		lastMessageLogId: number | null;
		streamStore: any;
		isMobile: boolean;
		onscroll: () => void;
		onstartedit: (message: ChatMessage) => void;
		oncanceledit: () => void;
		onsaveandregen: () => void;
		onregenerate: (message: ChatMessage) => void;
	} = $props();
	let chatContainer: HTMLElement | undefined = $state();
	let displayMessages = $derived((() => {
		const out: ChatMessage[] = [];
		for (const m of messages) {
			if (m.author === 'ego' && m.isThinking) {
				const last = out[out.length - 1];
				if (last && last.author === 'ego' && last.isThinking) {
					out[out.length - 1] = m;
				} else {
					out.push(m);
				}
			} else {
				out.push(m);
			}
		}
		return out;
	})());
	let lastMessageId = $derived(displayMessages.at(-1)?.id ?? null);
	export function scrollToBottom(behavior: 'smooth' | 'auto' = 'smooth') {
		tick().then(() => {
			if (chatContainer) {
				chatContainer.scrollTo({
					top: chatContainer.scrollHeight,
					behavior
				});
			}
		});
	}
	export function isNearBottom(threshold = 200) {
		if (!chatContainer) return true;
		return chatContainer.scrollHeight - chatContainer.scrollTop <= chatContainer.clientHeight + threshold;
	}
</script>
<div
	bind:this={chatContainer}
	onscroll={onscroll}
	class="flex-1 min-h-0 overflow-y-auto px-4 pt-20 pb-44"
>
	<div class="w-full max-w-4xl mx-auto flex flex-col gap-2 relative">
		{#if messages.length === 0}
			{#if $page.params?.sessionID === 'new' && !streamStore.isDone}
				<div class="flex flex-col items-center justify-center text-center animate-fade-in-up not-prose pt-16 px-4 text-text-secondary">
					<img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-16 h-16 text-accent mb-2" />
					<div class="rounded-xl px-3.5 py-2 bg-secondary shadow-lg inline-flex items-center gap-2 loading-bubble-animation mx-1 md:mx-2">
						<div class="text-sm font-medium text-transparent bg-clip-text bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] animate-shine">
							{$_('chat.creating_session') || 'Создаём сессию для вас…'}
						</div>
					</div>
				</div>
			{:else}
				<div class="flex flex-col items-center justify-center text-center pt-16 px-4 text-text-secondary not-prose">
					<img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-16 h-16 text-accent mb-2" />
					<h2 class="text-xl font-bold text-text-primary mt-1">{$_('chat.welcome_title') || 'EGO'}</h2>
					<p class="text-sm mt-2 max-w-prose opacity-80">{$_('chat.landing_subtitle') || 'Выберите существующую сессию из списка слева или начните новую.'}</p>
				</div>
			{/if}
		{:else}
			{#each displayMessages as msg, i (msg.id)}
				<MessageItem
					message={msg}
					{editingLogId}
					{editingText}
					{editingTextarea}
					{lastMessageLogId}
					{streamStore}
					lastMessageId={lastMessageId}
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