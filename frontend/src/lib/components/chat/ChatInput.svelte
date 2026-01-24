<script lang="ts">
	import { Send, StopCircle, X, Paperclip, ArrowUp } from '@lucide/svelte';
	import { _ } from 'svelte-i18n';
	import { autosize } from '$lib/actions/autosize.ts';
	import type { ChatMode } from '$lib/types';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import FileAttachments from './FileAttachments.svelte';
	import ChatModeSelector from './ChatModeSelector.svelte';
	import { fly, scale } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';

	let {
		currentInput = $bindable(''),
		attachedFiles = $bindable([]),
		chatMode = $bindable('default' as ChatMode),
		isMobile = false,
		isConnecting = false,
		streamIsDone = true,
		editingLogId = null,
		onsend,
		onstop,
		onpaste,
		ondrop
	}: {
		currentInput: string;
		attachedFiles: File[];
		chatMode: ChatMode;
		isMobile: boolean;
		isConnecting: boolean;
		streamIsDone: boolean;
		editingLogId: number | null;
		onsend: () => void;
		onstop: () => void;
		onpaste: (event: ClipboardEvent) => void;
		ondrop: (event: DragEvent) => void;
	} = $props();

	let inputArea: HTMLTextAreaElement | undefined = $state();
	let showInputOptions = $state(false);
	let fileAttachmentsRef: FileAttachments | undefined = $state();
	let isFocused = $state(false);
	let isTyping = $state(false);
	let typingTimeout: NodeJS.Timeout;

	const isInputDisabled = $derived(
		!streamIsDone || editingLogId !== null || chatStore.isUILocked
	);

	function handleKeydown(e: KeyboardEvent) {
		if (!isMobile && e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			onsend();
		}
	}

	function handleInput() {
		isTyping = true;
		clearTimeout(typingTimeout);
		typingTimeout = setTimeout(() => {
			isTyping = false;
		}, 500); // Reset typing state after 500ms of inactivity
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		fileAttachmentsRef?.addFiles(event.dataTransfer?.files || []);
		ondrop(event);
	}
</script>

<div
	class="pointer-events-none fixed bottom-0 z-20 w-full bg-gradient-to-t from-primary via-primary/80 to-transparent pt-32 pb-6 transition-all duration-500"
	style="left: var(--sidebar-offset); right: 0; width: auto;"
>
	<div class="pointer-events-auto mx-auto w-full max-w-4xl px-4">
		<div class="flex flex-col gap-3">
			
			<!-- File Previews -->
			{#if attachedFiles.length > 0}
				<div class="flex flex-wrap items-center gap-2 px-4" transition:fly={{ y: 20, duration: 200 }}>
					{#each attachedFiles as file, i (file.name + i)}
						<div
							class="group relative flex items-center gap-2.5 overflow-hidden rounded-xl border border-black/5 dark:border-white/10 bg-secondary/80 pl-1.5 pr-2.5 py-1.5 text-xs font-medium text-text-primary shadow-lg backdrop-blur-md transition-all hover:border-red-500/30 hover:bg-red-500/10"
							transition:scale={{ duration: 200, start: 0.9 }}
						>
							{#if file.type?.startsWith('image/')}
								<img
									src={URL.createObjectURL(file)}
									alt="preview"
									class="h-8 w-8 rounded-lg object-cover shadow-sm"
								/>
							{:else}
								<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-tertiary">
									<Paperclip class="h-4 w-4 text-text-secondary" />
								</div>
							{/if}
							<span class="max-w-[150px] truncate">{file.name}</span>
							<button
								onclick={() => fileAttachmentsRef?.removeFile(i)}
								class="absolute inset-0 flex items-center justify-center bg-red-500/90 text-white opacity-0 transition-opacity group-hover:opacity-100"
								aria-label="Remove file"
							>
								<X class="h-4 w-4" />
							</button>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Input Container -->
			<div
				class="relative flex w-full items-end gap-2 rounded-[26px] border border-black/5 dark:border-white/10 bg-secondary/60 p-1.5 backdrop-blur-xl transition-all duration-200 ease-out will-change-transform
				{isTyping || isFocused ? 'shadow-[0_8px_40px_rgba(var(--color-accent-rgb),0.15)]' : 'shadow-[0_8px_32px_rgba(0,0,0,0.12)]'}
				{isFocused ? 'ring-2 ring-accent ring-opacity-50' : ''}
				{isTyping ? 'scale-[1.005] border-accent/30' : ''}"
				role="region"
				aria-label="Message input"
				ondragover={(e) => e.preventDefault()}
				ondrop={handleDrop}
			>
				<!-- Left Actions -->
				<div class="flex shrink-0 items-center gap-1 h-10">
					<div class="relative">
						<FileAttachments bind:this={fileAttachmentsRef} bind:attachedFiles />
					</div>
					<ChatModeSelector bind:chatMode {isMobile} bind:showInputOptions />
				</div>

				<!-- Textarea -->
				<textarea
					bind:this={inputArea}
					use:autosize
					bind:value={currentInput}
					onkeydown={handleKeydown}
					oninput={handleInput}
					onfocus={() => isFocused = true}
					onblur={() => isFocused = false}
					{onpaste}
					placeholder={$_('chat.placeholder')}
					disabled={isInputDisabled}
					class="custom-scrollbar max-h-[200px] min-h-[40px] w-full resize-none bg-transparent py-2.5 px-1 text-[15px] leading-relaxed text-text-primary placeholder:text-text-secondary/50 focus:outline-none disabled:opacity-50"
					rows="1"
				></textarea>

				<!-- Send Button -->
				<div class="shrink-0 h-10 w-10">
					{#if !streamIsDone}
						<button
							onclick={onstop}
							class="group flex h-10 w-10 items-center justify-center rounded-full bg-text-primary text-secondary shadow-lg transition-all hover:scale-105 active:scale-95"
							title={$_('chat.stop_generation')}
							aria-label={$_('chat.stop_generation')}
							transition:scale={{ duration: 200, easing: cubicOut }}
						>
							<div class="h-3 w-3 rounded-[2px] bg-secondary group-hover:bg-red-500 transition-colors"></div>
						</button>
					{:else}
						<button
							onclick={onsend}
							disabled={isInputDisabled || (!currentInput.trim() && attachedFiles.length === 0)}
							class="flex h-10 w-10 items-center justify-center rounded-full bg-accent text-white shadow-lg shadow-accent/25 transition-all duration-300 hover:bg-accent-hover hover:scale-105 hover:shadow-accent/40 active:scale-95 disabled:cursor-not-allowed disabled:bg-tertiary disabled:text-text-secondary disabled:shadow-none disabled:hover:scale-100"
							class:scale-110={isTyping && currentInput.trim().length > 0}
							title={$_('chat.send_message')}
							aria-label={$_('chat.send_message')}
							transition:scale={{ duration: 200, easing: cubicOut }}
						>
							<ArrowUp class="h-5 w-5 stroke-[2.5]" />
						</button>
					{/if}
				</div>
			</div>

			<!-- Footer -->
			<div class="text-center">
				<p class="text-[10px] text-text-secondary/40 transition-opacity hover:opacity-100">
					{$_('chat.disclaimer_small')}
				</p>
			</div>
		</div>
	</div>
</div>

<style>
	.custom-scrollbar::-webkit-scrollbar {
		width: 4px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background: rgba(156, 163, 175, 0.2);
		border-radius: 10px;
	}
</style>
