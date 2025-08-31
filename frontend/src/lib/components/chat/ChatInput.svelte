<script lang="ts">
	import { Send, StopCircle, X } from '@lucide/svelte';
	import { _ } from 'svelte-i18n';
	import { autosize } from '$lib/actions/autosize.ts';
	import type { ChatMode } from '$lib/types';
	import FileAttachments from './FileAttachments.svelte';
	import ChatModeSelector from './ChatModeSelector.svelte';
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
	function handleKeydown(e: KeyboardEvent) {
		if (!isMobile && e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			onsend();
		}
	}
	function handleDrop(event: DragEvent) {
		event.preventDefault();
		fileAttachmentsRef?.addFiles(event.dataTransfer?.files || []);
		ondrop(event);
	}
</script>
<div
	class="fixed bottom-0 pt-24 bg-gradient-to-t from-primary to-transparent pointer-events-none z-20"
	style="left: var(--sidebar-offset); right: 0;"
>
	<div class="w-full px-4 pb-2 pointer-events-auto">
		<div class="max-w-4xl mx-auto flex flex-col gap-2 relative">
			{#if attachedFiles.length > 0}
				<div class="px-1 pb-2 flex flex-wrap gap-2 items-start content-start">
					{#each attachedFiles as file, i (file.name + i)}
						<div class="bg-tertiary/80 text-text-primary text-xs sm:text-sm font-medium px-2.5 py-1.5 rounded-lg flex items-center gap-2.5">
							{#if file.type?.startsWith('image/')}
								<img src={URL.createObjectURL(file)} alt="preview" class="w-8 h-8 rounded object-cover" />
							{/if}
							<span class="truncate max-w-[50vw] sm:max-w-[240px]">{file.name}</span>
							<button onclick={() => fileAttachmentsRef?.removeFile(i)} class="text-text-secondary hover:text-text-primary transition-colors">
								<X class="w-5 h-5" />
							</button>
						</div>
					{/each}
				</div>
			{/if}
			<div
				class="input-container bg-secondary border border-tertiary rounded-2xl shadow-lg flex flex-col max-h-[50vh] overflow-visible md:overflow-visible w-full max-w-4xl mx-auto"
				ondragover={e => e.preventDefault()}
				ondrop={handleDrop}
			>
				<textarea
					bind:this={inputArea}
					use:autosize
					bind:value={currentInput}
					onkeydown={handleKeydown}
					onpaste={onpaste}
					placeholder={$_('chat.placeholder')}
					disabled={isConnecting || !streamIsDone || editingLogId !== null}
					class="w-full bg-transparent p-4 resize-none outline-none placeholder-text-secondary max-h-52 overflow-y-auto touch-pan-y overscroll-contain"
					style="-webkit-overflow-scrolling: touch;"
					rows="1"
				></textarea>
				<div class="flex items-center justify-between w-full border-t border-tertiary/50 px-2 py-1">
					<div class="flex items-center gap-1">
						<FileAttachments bind:this={fileAttachmentsRef} bind:attachedFiles />
						<ChatModeSelector bind:chatMode {isMobile} bind:showInputOptions />
					</div>
					<div class="flex items-center">
						{#if !streamIsDone}
							<button
								onclick={onstop}
								class="rounded-full bg-red-500 hover:bg-red-600 text-white transition-colors duration-200 w-10 h-10 flex items-center justify-center"
								title={$_('chat.stop_generation')}
							>
								<StopCircle class="w-5 h-5" />
							</button>
						{:else}
							<button
								onclick={onsend}
								disabled={isConnecting ||
									(!currentInput.trim() && attachedFiles.length === 0) ||
									editingLogId !== null}
								class="rounded-full bg-accent hover:bg-accent-hover disabled:bg-tertiary disabled:cursor-not-allowed text-white transition-colors duration-200 w-10 h-10 flex items-center justify-center"
								title={$_('chat.send_message')}
							>
								<Send class="w-5 h-5" />
							</button>
						{/if}
					</div>
				</div>
			</div>
			<div class="w-full text-center text-[11px] leading-4 text-text-secondary opacity-80 select-none pb-2">
				{$_('chat.disclaimer_small')}
			</div>
		</div>
	</div>
</div>