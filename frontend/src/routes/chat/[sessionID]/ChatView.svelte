<script lang="ts">
	import type { PageData } from './$types';
	import type { ChatSession } from '$lib/types';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { api } from '$lib/api';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { uiStore, setShowSettingsModal } from '$lib/stores/ui.svelte.ts';
	import Modal from '$lib/components/Modal.svelte';
	import ChatHeader from '$lib/components/chat/ChatHeader.svelte';
	import ChatContainer from '$lib/components/chat/ChatContainer.svelte';
	import ErrorBoundary from '$lib/components/ErrorBoundary.svelte';
	import ChatHistorySkeleton from '$lib/components/ui/skeletons/ChatHistorySkeleton.svelte';
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { browser } from '$app/environment';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { canvasStore } from '$lib/stores/canvas.svelte.ts';
	import Canvas from '$lib/components/Canvas.svelte';

	let { data } = $props<{ data: PageData }>();

	// ============================================================================
	// STATE
	// ============================================================================

	let customInstructionsInput = $state('');
	let isNewSession = $state(true);
	let isMounted = $state(false);

	onMount(() => {
		isMounted = true;
	});

	// ============================================================================
	// DERIVED STATE
	// ============================================================================

	let currentSession = $derived(
		(data.session?.uuid ? chatStore.getSessionById(data.session.uuid) : null) || data.session
	);

	let sessionID = $derived(data.session?.uuid || 'new');

	// ============================================================================
	// EFFECTS
	// ============================================================================

	// Update custom instructions when session changes
	$effect(() => {
		const session = data.session;
		if (session) {
			customInstructionsInput = session.custom_instructions || '';
			isNewSession = false;
		} else {
			isNewSession = true;
		}
	});

	// ============================================================================
	// ACTIONS
	// ============================================================================

	async function saveSettings() {
		try {
			if (currentSession) {
				const updated = await api.patch<ChatSession>(`/sessions/${currentSession.uuid}`, {
					custom_instructions: customInstructionsInput
				});
				chatStore.updateSession(updated);
			} else {
				// We are in a new session state, so just "saving" locally effectively means keeping the input value
				// The ChatContainer will read `customInstructionsInput`
			}
			toast.success($_('toasts.instructions_saved'));
			setShowSettingsModal(false);
		} catch (error: unknown) {
			if (error instanceof Error) {
				toast.error(error.message);
			} else {
				toast.error('Failed to save instructions');
			}
		}
	}
</script>

<ErrorBoundary
	showDetails={true}
	onError={(error, errorInfo) => {
		console.error('ChatView error:', error, errorInfo);
		toast.error($_('errors.chat_view_error') || 'Chat view encountered an error');
	}}
>
	<div class="relative flex h-screen flex-col overflow-hidden">
		<ChatHeader {currentSession} />

		{#if isMounted}
			<div in:fade={{ duration: 200 }} class="flex flex-1 overflow-hidden relative">
				<div class="flex-1 min-w-0 h-full flex flex-col transition-all duration-300">
					<ChatContainer
						{sessionID}
						initialSession={data.session}
						initialMessages={data.messages || []}
						pendingCustomInstructions={customInstructionsInput}
					/>
				</div>
				
				{#if canvasStore.isOpen}
					<div 
						class="w-1/2 h-full border-l border-white/10 shadow-2xl z-20 backdrop-blur-xl bg-secondary/30"
						transition:fly={{ x: 100, duration: 500, opacity: 0, easing: cubicOut }}
					>
						<Canvas />
					</div>
				{/if}
			</div>
		{:else}
			<ChatHistorySkeleton />
		{/if}

		<Modal
			title={$_('chat.instructions_title')}
			show={uiStore.showSettingsModal}
			onclose={() => setShowSettingsModal(false)}
		>
			{#if currentSession || isNewSession}
				<p class="mb-4 text-text-secondary">{$_('chat.instructions_prompt')}</p>
				<textarea
					bind:value={customInstructionsInput}
					rows="5"
					class="w-full rounded-lg border-2 border-tertiary bg-primary p-3 transition-all duration-300 focus:border-accent focus:ring-accent"
				></textarea>
				<button
					onclick={saveSettings}
					class="mt-4 w-full rounded-lg bg-accent py-2 font-semibold transition-colors duration-200 hover:bg-accent-hover"
					>{$_('chat.save')}</button
				>
			{:else}
				<p class="mb-2 text-text-secondary">{$_('chat.start_dialog_first')}</p>
			{/if}
		</Modal>
	</div>
</ErrorBoundary>
