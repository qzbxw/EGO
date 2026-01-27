<script lang="ts">
	import { Copy, Pencil, RefreshCw } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { chatStore } from '$lib/stores/chat.svelte';

	const dispatch = createEventDispatcher();
	let {
		author,
		text,
		isLastMessage,
		isThinking,
		isCancelled,
		streamIsDone,
		logId
	}: {
		author: string;
		text: string;
		isLastMessage: boolean;
		isThinking: boolean;
		isCancelled: boolean;
		streamIsDone: boolean;
		logId?: number;
	} = $props();

	const canRegenerate = $derived(
		isLastMessage && !isThinking && !isCancelled && !!logId && streamIsDone && !chatStore.isUILocked
	);

	const canEdit = $derived(author === 'user' && !!logId && streamIsDone && !chatStore.isUILocked);

	function copyToClipboard(str: string) {
		navigator.clipboard
			.writeText(str || '')
			.then(() =>
				toast.success(
					$_(author === 'user' ? 'toasts.copy_message_success' : 'toasts.copy_response_success')
				)
			)
			.catch(() =>
				toast.error(
					$_(author === 'user' ? 'toasts.copy_message_error' : 'toasts.copy_response_error')
				)
			);
	}
</script>

<div class="flex items-center gap-1 text-text-secondary/70 md:gap-2">
	{#if canRegenerate}
		<button
			onclick={() => dispatch('regenerate')}
			disabled={chatStore.isUILocked}
			class="p-2 transition-all hover:text-text-primary active:scale-90 disabled:cursor-not-allowed disabled:opacity-50 md:p-1"
			title={$_('chat.regenerate')}
		>
			<RefreshCw size={14} class={chatStore.isRegenerating ? 'animate-spin' : ''} />
		</button>
	{/if}
	{#if canEdit}
		<button
			type="button"
			onclick={(e) => {
				e.stopPropagation();
				dispatch('edit');
			}}
			disabled={chatStore.isUILocked}
			class="p-2 transition-all hover:text-text-primary active:scale-90 disabled:cursor-not-allowed disabled:opacity-50 md:p-1"
			title={$_('chat.edit_and_regenerate')}
		>
			<Pencil size={14} />
		</button>
	{/if}
	<button
		onclick={() => copyToClipboard(text as string)}
		class="p-2 transition-all hover:text-text-primary active:scale-90 md:p-1"
		title={$_('chat.copy')}
	>
		<Copy size={14} />
	</button>
</div>
