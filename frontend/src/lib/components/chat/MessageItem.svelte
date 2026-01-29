<script lang="ts">
	import { Paperclip, StopCircle } from '@lucide/svelte';
	import { fly, fade } from 'svelte/transition';
	import { cubicOut, quintOut } from 'svelte/easing';
	import { _ } from 'svelte-i18n';
	import { autosize } from '$lib/actions/autosize.ts';
	import { auth } from '$lib/stores/auth.svelte.ts';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import MessageBody from '$lib/components/MessageBody.svelte';
	import StreamingMarkdownLite from '$lib/components/StreamingMarkdownLite.svelte';
	import ThoughtChain from './ThoughtChain.svelte';
	import SuperEgoDebate from './SuperEgoDebate.svelte';
	import LazyImage from '$lib/components/LazyImage.svelte';
	import MessageActions from '$lib/components/MessageActions.svelte';
	import type { ChatMessage, StreamStoreState } from '$lib/types';

	let {
		message,
		editingLogId = null,
		editingText = '',
		editingTextarea = null,
		streamStore,
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
		streamStore: StreamStoreState;
		isMobile: boolean;
		index: number;
		onstartedit: (message: ChatMessage) => void;
		oncanceledit: () => void;
		onsaveandregen: () => void;
		onregenerate: (message: ChatMessage) => void;
	} = $props();
</script>

<div
	class="group mb-6 flex w-full"
	class:justify-start={message.author === 'ego'}
	class:justify-end={message.author === 'user'}
>
	<div
		class="flex w-full min-w-0 max-w-full flex-col"
		class:items-end={message.author === 'user'}
		class:items-start={message.author === 'ego'}
		class:md:max-w-5xl={message.author === 'user'}
		class:md:max-w-6xl={message.author === 'ego'}
		in:fly={message.author === 'user'
			? { y: 15, duration: 300, delay: Math.min(index * 30, 150), easing: quintOut }
			: { y: 15, duration: 300, delay: Math.min(index * 30, 150), easing: quintOut }}
	>
		<!-- Avatar + Name Header -->
		<div
			class="mb-2 flex items-center gap-3 px-1"
			class:flex-row-reverse={message.author === 'user'}
		>
			<div
				class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border bg-secondary/80 shadow-sm backdrop-blur-md transition-colors {message.author ===
				'user'
					? 'border-accent/30'
					: 'border-white/10'}"
			>
				{#if message.author === 'ego'}
					<img
						src={getAppLogo(preferencesStore.theme)}
						alt="EGO avatar"
						class="h-5 w-5 object-contain"
					/>
				{:else}
					<span class="text-xs font-black text-accent">
						{auth.user?.username?.charAt(0).toUpperCase() || 'U'}
					</span>
				{/if}
			</div>
			<span class="text-[10px] font-black uppercase tracking-[0.2em] text-text-secondary/60">
				{message.author === 'ego' ? 'EGO' : $_('chat.you')}
			</span>
		</div>

		<!-- Message Content Container -->
		<div
			class="relative flex w-full min-w-0 flex-col gap-2"
			class:items-end={message.author === 'user'}
			class:items-start={message.author === 'ego'}
		>
			<!-- Thinking / Streaming / Tools -->
			{#if message.isThinking}
				<ThoughtChain
					thoughts={streamStore.thoughts}
					currentHeader={streamStore.thoughtHeader}
					isThinking={true}
				/>

				{#if streamStore.superEgoDebate}
					<SuperEgoDebate debate={streamStore.superEgoDebate} />
				{/if}

				{#if streamStore.textStream}
					<!-- Streaming Text Bubble -->
					<div
						class="glass-bubble relative overflow-hidden rounded-2xl rounded-tl-none border border-white/5 bg-secondary/60 p-5 shadow-xl backdrop-blur-2xl"
					>
						<StreamingMarkdownLite text={streamStore.textStream} isDone={streamStore.isDone} />
					</div>
				{/if}

				<!-- Editing Mode -->
			{:else if editingLogId === message.logId && message.author === 'user'}
				<div in:fly={{ y: 10, duration: 200 }} class="w-full max-w-2xl">
					<div
						class="rounded-2xl border border-accent/30 bg-secondary/80 p-1 shadow-2xl backdrop-blur-xl"
					>
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
							class="max-h-[300px] w-full resize-none rounded-xl bg-transparent p-4 text-text-primary outline-none placeholder:text-text-secondary/50"
							placeholder="Edit your message..."
						></textarea>
					</div>
					<div class="mt-3 flex justify-end gap-2 px-1">
						<button
							onclick={oncanceledit}
							class="rounded-xl px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-white/5 hover:text-text-primary"
						>
							Cancel
						</button>
						<button
							onclick={onsaveandregen}
							class="rounded-xl bg-accent px-4 py-2 text-sm font-bold text-white shadow-lg shadow-accent/20 transition-all hover:bg-accent-hover hover:shadow-accent/40 active:scale-95"
						>
							Save & Regenerate
						</button>
					</div>
				</div>

				<!-- Standard Message -->
			{:else}
				<!-- Attachments -->
				{#if message.attachments && message.attachments.length > 0}
					<div class="mb-2 flex flex-wrap gap-2" class:justify-end={message.author === 'user'}>
						{#each message.attachments as attachment (attachment.file_name + attachment.mime_type)}
							<div
								class="overflow-hidden rounded-xl border border-white/10 bg-secondary/50 shadow-md transition-transform hover:scale-[1.02]"
							>
								{#if attachment.mime_type?.startsWith('image/') && attachment.preview_url}
									<LazyImage
										src={attachment.preview_url}
										alt={attachment.file_name}
										class="block max-h-48 max-w-[280px] object-cover"
									/>
								{:else}
									<div class="flex items-center gap-3 px-4 py-3">
										<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-white/10">
											<Paperclip class="h-4 w-4 text-text-primary" />
										</div>
										<div class="flex flex-col">
											<span class="max-w-[180px] truncate text-sm font-medium text-text-primary"
												>{attachment.file_name}</span
											>
											{#if attachment.mime_type}
												<span class="text-[10px] text-text-secondary">{attachment.mime_type}</span>
											{/if}
										</div>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
				<!-- Text Bubble -->
				{#if message.text || (message.thoughts && message.thoughts.length > 0)}
					<div class="flex w-full flex-col gap-3" class:items-end={message.author === 'user'}>
						{#if message.thoughts && message.thoughts.length > 0}
							<ThoughtChain thoughts={message.thoughts} />
						{/if}

						{#if message.text}
							<div
								in:fade={{ duration: 300, easing: cubicOut }}
								class="message-bubble relative z-10 w-fit max-w-[85vw] break-words px-6 py-4 text-[15px] leading-7 shadow-sm transition-all md:max-w-3xl
								{message.author === 'user'
									? 'rounded-2xl rounded-tr-none border border-accent/20 bg-accent/[0.03] text-text-primary backdrop-blur-2xl'
									: 'rounded-2xl rounded-tl-none border border-white/5 bg-secondary/60 text-text-primary backdrop-blur-2xl'}"
							>
								<MessageBody text={message.text} invert={false} />

								{#if message.isCancelled}
									<div
										class="mt-3 flex items-center gap-1.5 rounded-lg bg-red-500/10 px-3 py-1.5 text-xs font-medium text-red-400"
									>
										<StopCircle class="h-3 w-3" />
										<span>Generation stopped</span>
									</div>
								{/if}
							</div>
						{/if}
					</div>
				{/if}
			{/if}
		</div>

		<!-- Message Actions -->
		{#if !(editingLogId === message.logId && message.author === 'user')}
			<div
				class="mt-2 flex items-center gap-2 opacity-0 transition-opacity duration-200 group-hover:opacity-100"
				class:justify-end={message.author === 'user'}
				class:justify-start={message.author === 'ego'}
			>
				<MessageActions
					author={message.author}
					text={message.text}
					isLastMessage={index === 0}
					isThinking={!!message.isThinking}
					isCancelled={!!message.isCancelled}
					streamIsDone={streamStore.isDone}
					logId={message.logId}
					on:regenerate={() => onregenerate(message)}
					on:edit={() => onstartedit(message)}
				/>
			</div>
		{/if}
	</div>
</div>

<style>
	.glass-bubble {
		box-shadow:
			0 4px 6px -1px rgba(0, 0, 0, 0.1),
			0 2px 4px -1px rgba(0, 0, 0, 0.06),
			inset 0 1px 0 rgba(255, 255, 255, 0.05);
	}
	.message-bubble {
		user-select: text !important;
		-webkit-user-select: text !important;
		overflow: visible !important;
	}
</style>
