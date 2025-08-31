<script lang="ts">
	import { Check, X, Pencil, CircleHelp } from '@lucide/svelte';
	import { _ } from 'svelte-i18n';
	import { page } from '$app/stores';
	import { tick } from 'svelte';
	import { api } from '$lib/api';
	import { updateSession } from '$lib/stores/sessions.svelte.ts';
	import { toast } from 'svelte-sonner';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import type { ChatSession } from '$lib/types';
	let { currentSession }: { currentSession: ChatSession | null } = $props();
	let isRenaming = $state(false);
	let sessionTitleInput = $state('');
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
	class="fixed top-0 p-3 pl-16 border-b border-tertiary bg-primary/80 backdrop-blur-md z-30 chat-header"
	style="left: var(--sidebar-offset); right: 0;"
>
	<div class="flex items-center justify-between gap-3">
		<div class="flex items-center gap-2 group min-w-0">
			{#if isRenaming}
				<input
					type="text"
					bind:value={sessionTitleInput}
					onkeydown={(e) => {
						if (e.key === 'Enter') saveTitle();
						if (e.key === 'Escape') cancelRenaming();
					}}
					onblur={saveTitle}
					class="text-lg font-bold truncate bg-transparent border-b-2 border-accent w-full focus:outline-none"
				/>
				<button
					onclick={saveTitle}
					class="p-2 rounded-md hover:bg-tertiary text-green-400 transition-colors"
				>
					<Check class="w-5 h-5" />
				</button>
				<button
					onclick={cancelRenaming}
					class="p-2 rounded-md hover:bg-tertiary text-red-400 transition-colors"
				>
					<X class="w-5 h-5" />
				</button>
			{:else}
				<h1 class="text-lg font-bold truncate max-w-[60vw] md:max-w-[50vw]">
					{currentSession?.title || $_('chat.untitled_session') || 'Новая сессия'}
				</h1>
				{#if $page.params?.sessionID && currentSession?.uuid === $page.params.sessionID}
					<button
						class="p-2 rounded-md hover:bg-tertiary transition-colors"
						title={$_('chat.rename_session')}
						onclick={startRenaming}
					>
						<Pencil class="w-5 h-5" />
					</button>
				{/if}
			{/if}
		</div>
		<div class="flex items-center gap-2">
			<LanguageSwitcher />
			<a
				href="/knowego"
				class="p-2 rounded-md hover:bg-tertiary transition-colors"
				title={$_('chat.help')}
			>
				<CircleHelp class="w-5 h-5" />
			</a>
		</div>
	</div>
</header>