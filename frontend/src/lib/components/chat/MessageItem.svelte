<script lang="ts">
	import { Paperclip, StopCircle } from '@lucide/svelte';
	import { fly } from 'svelte/transition';
	import { _ } from 'svelte-i18n';
	import { autosize } from '$lib/actions/autosize.ts';
	import { auth } from '$lib/stores/auth.svelte.ts';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { formatToolHeader, parseToolHeader } from '$lib/utils/toolStatus.ts';
	import MessageBody from '$lib/components/MessageBody.svelte';
	import StreamingMarkdownLite from '$lib/components/StreamingMarkdownLite.svelte';
	import AuthenticatedImage from '$lib/components/AuthenticatedImage.svelte';
	import MessageActions from '$lib/components/MessageActions.svelte';
	import type { ChatMessage } from '$lib/types';
	let {
		message,
		editingLogId = null,
		editingText = '',
		editingTextarea = null,
		lastMessageLogId = null,
		streamStore,
		lastMessageId = null,
		isMobile = false,
		index = 0,
		onstartedit,
		oncanceledit,
		onsaveandregen,
		onregenerate
	}: {
		message: ChatMessage;
		editingLogId: number | null;
		editingText: string;
		editingTextarea: HTMLTextAreaElement | null;
		lastMessageLogId: number | null;
		streamStore: any;
		lastMessageId: number | null;
		isMobile: boolean;
		index: number;
		onstartedit: (message: ChatMessage) => void;
		oncanceledit: () => void;
		onsaveandregen: () => void;
		onregenerate: (message: ChatMessage) => void;
	} = $props();
</script>
<div
	class="flex w-full group"
	class:justify-start={message.author === 'ego'}
	class:justify-end={message.author === 'user'}
>
	<div
		class="flex flex-col w-full max-w-full min-w-0"
		class:items-end={message.author === 'user'}
		class:items-start={message.author === 'ego'}
		class:md:max-w-5xl={message.author === 'user'}
		class:md:max-w-6xl={message.author === 'ego'}
	>
		<div
			class="flex items-center gap-3"
			class:flex-row-reverse={message.author === 'user'}
		>
			<div
				class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 bg-secondary"
			>
				{#if message.author === 'ego'}
					<img src={getAppLogo(preferencesStore.theme)} alt="EGO avatar" class="w-full h-full p-1" />
				{:else}
					<span class="font-bold text-accent text-lg"
						>{auth.user?.username?.charAt(0).toUpperCase() || 'U'}</span
					>
				{/if}
			</div>
			<div class="font-bold text-text-primary text-sm">
				{message.author === 'ego' ? 'EGO' : $_('chat.you')}
			</div>
		</div>
		<div
			class="mt-1 flex w-full flex-col gap-2 min-w-0"
			class:items-end={message.author === 'user'}
			class:items-start={message.author === 'ego'}
			class:pl-0={message.author === 'ego'}
		>
			{#if message.isThinking}
				{#if streamStore.textStream}
					<div class="rounded-xl px-3 py-1.5 text-base break-words shadow-lg shadow-black/20 transition-all min-w-0 inline-block w-fit max-w-[860px] mx-1 md:mx-2 bg-secondary streaming-text loading-bubble-animation has-stream-text" style="align-self:flex-start;">
						<StreamingMarkdownLite text={streamStore.textStream} />
					</div>
				{:else}
					{#if streamStore.thoughtHeader}
						{@const toolInfo = parseToolHeader(streamStore.thoughtHeader)}
						{@const formattedHeader = formatToolHeader(streamStore.thoughtHeader)}
						<div class="rounded-xl px-3.5 py-2 bg-secondary shadow-lg inline-flex items-center gap-2 loading-bubble-animation self-start md:self-auto mx-1 md:mx-2 border border-tertiary/50 hover:border-accent/30 transition-colors duration-300">
							<div class="text-sm font-medium text-transparent bg-clip-text bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] animate-shine">
								{formattedHeader}
							</div>
						</div>
					{:else}
						<div class="rounded-xl px-3.5 py-2 bg-secondary shadow-lg inline-flex items-center gap-2 loading-bubble-animation self-start md:self-auto mx-1 md:mx-2">
							<div class="text-sm font-medium text-transparent bg-clip-text bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] animate-shine">
								{message.text || $_(message.attachments && message.attachments.length > 0 ? 'chat.thinking_with_files' : 'chat.thinking')}
							</div>
						</div>
					{/if}
				{/if}
			{:else if editingLogId === message.logId && message.author === 'user'}
				<div transition:fly={{ y: 5, duration: 200 }} class="flex w-full flex-col">
					<div class="bg-secondary border border-accent/40 rounded-xl p-3 shadow-lg shadow-black/10">
						<textarea
							bind:this={editingTextarea}
							bind:value={editingText}
							onkeydown={(e) => {
								if (!isMobile && e.key === 'Enter' && !e.shiftKey) {
									e.preventDefault();
									onsaveandregen();
								}
								if (e.key === 'Escape') oncanceledit();
							}}
							use:autosize
							class="w-full bg-transparent p-4 resize-none outline-none placeholder:text-text-secondary"
							placeholder="Редактирование сообщения..."
						></textarea>
					</div>
					<div class="mt-2 flex items-center justify-end gap-2">
						<button
							onclick={oncanceledit}
							class="rounded-md bg-secondary px-3 py-1.5 text-sm font-medium transition-colors hover:bg-tertiary"
						>{$_('sidebar.cancel_action')}</button>
						<button
							onclick={onsaveandregen}
							class="rounded-md bg-accent px-3 py-1.5 text-sm font-semibold text-white transition-colors hover:bg-accent-hover"
						>
							Сохранить и сгенерировать
						</button>
					</div>
				</div>
			{:else}
				{#if message.attachments && message.attachments.length > 0}
					<div class="flex flex-wrap gap-2 mb-1" class:justify-end={message.author === 'user'}>
						{#each message.attachments as attachment (attachment.file_name + (attachment.preview_url || ''))}
							{#if attachment.mime_type && attachment.mime_type.startsWith('image/') && attachment.preview_url}
								<div class="rounded-xl overflow-hidden border border-tertiary bg-primary shadow-md">
									{#if attachment.preview_url.startsWith('blob:') || attachment.preview_url.startsWith('data:')}
										<img src={attachment.preview_url} alt={attachment.file_name} class="max-h-40 max-w-[260px] object-cover block" />
									{:else}
										<AuthenticatedImage src={attachment.preview_url} alt={attachment.file_name} class="max-h-40 max-w-[260px] object-cover block" />
									{/if}
								</div>
							{:else}
								<div class="flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs sm:text-sm font-medium bg-tertiary/90 border border-tertiary/60">
									<Paperclip class="h-4 w-4 flex-shrink-0" />
									<span class="truncate max-w-[220px]">{attachment.file_name}</span>
									{#if attachment.mime_type}
										<span class="text-text-secondary/80">{attachment.mime_type}</span>
									{/if}
								</div>
							{/if}
						{/each}
					</div>
				{/if}
				{#if message.text}
					<div
						class="rounded-xl px-3 py-1.5 text-base break-words shadow-lg shadow-black/20 transition-all min-w-0 inline-block mx-1 md:mx-2"
						class:bg-accent={message.author === 'user'}
						class:text-white={message.author === 'user'}
						class:bg-secondary={message.author === 'ego'}
						class:loading-bubble-animation={message.author === 'ego' && (message.isThinking || (message.logId && streamStore.currentLogId && message.logId === streamStore.currentLogId) || (lastMessageId !== null && message.id === lastMessageId && !streamStore.isDone))}
						style={message.author === 'user' ? 'max-width: min(100%, 860px); align-self: flex-end' : 'max-width: min(100%, 860px); align-self: flex-start'}
					>
						<MessageBody text={message.text} invert={message.author === 'user'} />
						{#if message.isCancelled}
							<div class="mt-2 text-xs text-text-secondary/70 italic flex items-center gap-1">
								<StopCircle class="w-3 h-3" />
								Генерация остановлена
							</div>
						{/if}
					</div>
				{/if}
			{/if}
		</div>
		{#if !(editingLogId === message.logId && message.author === 'user')}
			<div class="message-actions mt-1 flex items-center gap-1 transition-opacity group-hover:opacity-100 md:opacity-0" class:justify-end={message.author === 'user'}>
				<MessageActions
					author={message.author}
					text={message.text}
					isLastMessage={index === 0}
					isThinking={message.isThinking}
					isCancelled={message.isCancelled}
					streamIsDone={streamStore.isDone}
					logId={message.logId}
					{lastMessageLogId}
					on:regenerate={() => onregenerate(message)}
					on:edit={() => onstartedit(message)}
				/>
			</div>
		{/if}
	</div>
</div>