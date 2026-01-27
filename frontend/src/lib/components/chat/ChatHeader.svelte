<script lang="ts">
	import { Check, X, Pencil } from '@lucide/svelte';
	import { _ } from 'svelte-i18n';
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { updateSession } from '$lib/stores/sessions.svelte.ts';
	import { toast } from 'svelte-sonner';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import type { ChatSession } from '$lib/types';
	let { currentSession }: { currentSession: ChatSession | null } = $props();
	let isRenaming = $state(false);
	let sessionTitleInput = $state('');
	let titleInput: HTMLInputElement | undefined = $state();

	$effect(() => {
		if (isRenaming && titleInput) {
			titleInput.focus();
		}
	});

	function startRenaming() {
		if (!currentSession) return;
		isRenaming = true;
		sessionTitleInput = currentSession.title;
	}
	function cancelRenaming() {
		isRenaming = false;
	}
	async function saveTitle() {
		if (!isRenaming || !currentSession || !sessionTitleInput.trim()) {
			cancelRenaming();
			return;
		}
		try {
			const updated = await api.patch<ChatSession>(`/sessions/${currentSession.uuid}`, {
				title: sessionTitleInput.trim()
			});
			updateSession(updated);
			toast.success($_('toasts.title_updated'));
		} catch (e: unknown) {
			if (e instanceof Error) {
				toast.error($_('errors.title_update_failed', { values: { message: e.message } }));
			} else {
				toast.error($_('errors.title_update_failed', { values: { message: 'Unknown error' } }));
			}
		} finally {
			cancelRenaming();
		}
	}
</script>

<header
	class="chat-header fixed top-0 z-30 flex h-16 w-full items-center justify-between border-b border-black/5 bg-secondary/80 px-6 backdrop-blur-xl transition-all duration-300 dark:border-white/5 dark:bg-secondary/60"
	style="left: var(--sidebar-offset); right: 0; width: auto;"
>
	<div class="flex flex-1 items-center gap-4 overflow-hidden">
		<!-- Menu placeholder to prevent overlap with fixed burger button -->
		<div class="w-14 shrink-0"></div>

		<div class="group flex min-w-0 items-center gap-2">
			{#if isRenaming}
				<div class="flex items-center gap-2">
					<input
						type="text"
						bind:this={titleInput}
						bind:value={sessionTitleInput}
						onkeydown={(e) => {
							if (e.key === 'Enter') saveTitle();
							if (e.key === 'Escape') cancelRenaming();
						}}
						onblur={saveTitle}
						class="min-w-[200px] border-b-2 border-accent bg-transparent px-1 py-0.5 text-lg font-bold text-text-primary focus:outline-none"
					/>
					<div class="flex items-center gap-1">
						<button
							onclick={saveTitle}
							class="rounded-lg bg-green-500/10 p-1.5 text-green-500 transition-colors hover:bg-green-500/20"
						>
							<Check class="h-4 w-4" />
						</button>
						<button
							onclick={cancelRenaming}
							class="rounded-lg bg-red-500/10 p-1.5 text-red-500 transition-colors hover:bg-red-500/20"
						>
							<X class="h-4 w-4" />
						</button>
					</div>
				</div>
			{:else}
				<h1
					class="truncate text-lg font-bold tracking-tight text-text-primary/90 transition-colors hover:text-text-primary"
				>
					{currentSession?.title || $_('chat.untitled_session') || 'New Session'}
				</h1>
				{#if $page.params?.sessionID && currentSession?.uuid === $page.params.sessionID}
					<button
						class="opacity-0 transition-all duration-200 focus:opacity-100 group-hover:opacity-100"
						title={$_('chat.rename_session')}
						onclick={startRenaming}
					>
						<Pencil class="h-4 w-4 text-text-secondary hover:text-accent" />
					</button>
				{/if}
			{/if}
		</div>
	</div>

	<div class="flex items-center gap-4">
		<LanguageSwitcher />
	</div>
</header>
