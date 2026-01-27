<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { formatToolHeader } from '$lib/utils/toolStatus.ts';
	import StreamingMarkdownLite from '$lib/components/StreamingMarkdownLite.svelte';

	// Reactive access to stream state
	let stream = $derived(chatStore.stream);
	let hasTextStream = $derived(!!stream.textStream);
	let hasThoughtHeader = $derived(!!stream.thoughtHeader);
</script>

{#if hasTextStream}
	<div
		class="streaming-text loading-bubble-animation has-stream-text mx-1 inline-block w-fit min-w-0 max-w-[860px] break-words rounded-xl bg-secondary px-3 py-1.5 text-base shadow-lg shadow-black/20 transition-all md:mx-2"
		style="align-self:flex-start;"
	>
		<StreamingMarkdownLite text={stream.textStream} />
	</div>
{:else if stream.activeTools && stream.activeTools.length > 0}
	<div class="mx-1 flex flex-col gap-1.5 self-start md:mx-2">
		{#each stream.activeTools as tool (tool.callId)}
			{@const formattedHeader = formatToolHeader(tool.header || tool.toolName)}
			<div
				class="loading-bubble-animation inline-flex items-center gap-2.5 rounded-2xl border border-tertiary/40 bg-secondary/30 px-3 py-2 shadow-sm backdrop-blur-md"
			>
				<div
					class="thinking-elements animate-shine bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] bg-clip-text text-sm font-semibold text-transparent"
				>
					{formattedHeader}
					{#if tool.status === 'running'}<span class="thinking-dots"
							><span class="thinking-dot animate-pulse-gentle"></span><span
								class="thinking-dot animate-pulse-gentle [animation-delay:0.2s]"
							></span><span class="thinking-dot animate-pulse-gentle [animation-delay:0.4s]"
							></span></span
						>{/if}
				</div>
			</div>
		{/each}
	</div>
{:else if hasThoughtHeader}
	{@const formattedHeader = formatToolHeader(stream.thoughtHeader)}
	<div
		class="loading-bubble-animation mx-1 inline-flex items-center gap-2 self-start rounded-xl border border-tertiary/50 bg-secondary px-3.5 py-2 shadow-lg transition-colors duration-300 hover:border-accent/30 md:mx-2 md:self-auto"
	>
		<div
			class="thinking-elements animate-shine bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] bg-clip-text text-sm font-medium text-transparent"
		>
			{formattedHeader}
		</div>
	</div>
{:else}
	{@const hasHeader = !!stream.thoughtHeader}
	{@const formattedHeader = hasHeader
		? formatToolHeader(stream.thoughtHeader)
		: $_('chat.thinking')}
	<div
		class="loading-bubble-animation mx-1 inline-flex items-center gap-2 self-start rounded-xl bg-secondary px-3.5 py-2 shadow-lg md:mx-2 md:self-auto"
	>
		<div
			class="thinking-elements animate-shine bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] bg-clip-text text-sm font-medium text-transparent"
		>
			{formattedHeader}
			<div class="thinking-dots">
				<span class="thinking-dot animate-pulse-gentle"></span>
				<span class="thinking-dot animate-pulse-gentle [animation-delay:0.2s]"></span>
				<span class="thinking-dot animate-pulse-gentle [animation-delay:0.4s]"></span>
			</div>
		</div>
	</div>
{/if}

<style>
	@keyframes -global-pulse-gentle {
		0%,
		100% {
			transform: scale(1);
			opacity: 0.5;
		}
		50% {
			transform: scale(1.2);
			opacity: 1;
		}
	}
</style>
