<script lang="ts">
	import {
		Copy,
		Pencil,
		RefreshCw
	} from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	const dispatch = createEventDispatcher();
	let { author, text, isLastMessage, isThinking, isCancelled, streamIsDone, logId, lastMessageLogId } = $props();
	$effect(() => {
		if (author === 'ego') {
			console.log('[MessageActions] EGO message conditions:', {
				author,
				isLastMessage,
				isThinking,
				isCancelled,
				logId,
				streamIsDone,
				shouldShowRegen: author === 'ego' && isLastMessage && !isThinking && !isCancelled && logId
			});
		}
	});
	function copyToClipboard(str: string) {
		navigator.clipboard
			.writeText(str)
			.then(() => toast.success($_(author === 'user' ? 'toasts.copy_message_success' : 'toasts.copy_response_success')))
			.catch(() => toast.error($_(author === 'user' ? 'toasts.copy_message_error' : 'toasts.copy_response_error')));
	}
</script>
<div class="flex items-center gap-2 text-gray-400">
	{#if author === 'ego' && isLastMessage && !isThinking && !isCancelled && logId}
		<button
			onclick={() => dispatch('regenerate')}
			class="p-1 hover:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
			disabled={!streamIsDone}
			title={$_('chat.regenerate')}
		>
			<RefreshCw size={16} />
		</button>
	{/if}
	{#if author === 'user' && isLastMessage && logId}
		<button
			onclick={() => dispatch('regenerate')}
			class="p-1 hover:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
			disabled={!streamIsDone}
			title={$_('chat.regenerate')}
		>
			<RefreshCw size={16} />
		</button>
	{/if}
	{#if author === 'user' && logId}
        <button
            type="button"
            onclick={(e) => { e.stopPropagation(); dispatch('edit'); }}
            class="p-1 hover:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={!streamIsDone}
            title={$_('chat.edit_and_regenerate')}
        >
            <Pencil size={16} />
        </button>
    {/if}
	<button onclick={() => copyToClipboard(text)} class="p-1 hover:text-gray-200" title={$_('chat.copy')}>
		<Copy size={16} />
	</button>
</div>