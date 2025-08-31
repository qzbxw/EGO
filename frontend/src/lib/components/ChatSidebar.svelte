<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { MessageSquare, Plus, Trash2, Settings, LogOut, Pencil, Check, X, User, BarChart3 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import { auth, logout } from '$lib/stores/auth.svelte.ts';
    import {
        sessions,
        isLoading,
        removeSession,
        clearUserSessions,
        updateSession
    } from '$lib/stores/sessions.svelte.ts';
	import { uiStore, setShowSettingsModal, setShowUserSettingsModal, setShowUserStatsModal } from '$lib/stores/ui.svelte.ts';
	import { api, apiPy } from '$lib/api';
    import { getAppLogo } from '$lib/config';
    import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { resetStreamStore } from '$lib/stores/stream.svelte.ts';
	import { connectionManager } from '$lib/connection-manager';
	import { browser } from '$app/environment';
	import type { ChatSession } from '$lib/types';
	let renamingId = $state<string | null>(null);
	let renameInputText = $state('');
	let renameInput: HTMLInputElement | undefined = $state();
	$effect(() => {
		if (renamingId !== null && renameInput) {
			setTimeout(() => {
				renameInput?.focus();
			}, 0);
		}
	});
	function handleLogout() {
		connectionManager.destroyVisibilityHandling();
		connectionManager.disconnect();
		clearUserSessions();
		resetStreamStore();
		if (browser) {
			sessionStorage.clear();
		}
		logout();
		goto('/login', { replaceState: true });
	}
	function confirmDelete(sessionUUID: string, event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		toast.error($_('sidebar.delete_confirm'), {
			action: {
				label: $_('sidebar.delete_action'),
				onClick: async () => {
					try {
						const isCurrentSession = $page.params.sessionID === sessionUUID;
						await api.delete(`/sessions/${sessionUUID}`);
						apiPy.deleteSessionVectors(sessionUUID).catch(() => {});
						toast.success($_('sidebar.session_deleted'));
						removeSession(sessionUUID);
						if (isCurrentSession) {
							await goto('/chat/new', { replaceState: true, invalidateAll: false });
						}
                    } catch (e: unknown) {
                        if (e instanceof Error) {
                            toast.error(e.message);
                        } else {
                            toast.error($_('errors.generic'));
                        }
					}
				}
			},
			cancel: {
				label: $_('sidebar.cancel_action'),
				onClick: () => {}
			}
		});
	}
	function startRenaming(session: ChatSession, e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		renamingId = session.uuid;
		renameInputText = session.title;
	}
	function cancelRenaming() {
		renamingId = null;
		renameInputText = '';
	}
	async function saveRename() {
		if (renamingId === null || !renameInputText.trim()) {
			cancelRenaming();
			return;
		}
		const sessionUUID = renamingId;
		const newTitle = renameInputText.trim();
        const originalSession = $sessions.find((s) => s.uuid === sessionUUID);
		cancelRenaming();
		if (originalSession && originalSession.title === newTitle) {
			return;
		}
		try {
			updateSession({ uuid: sessionUUID, title: newTitle } as ChatSession);
			const updatedSession = await api.patch<ChatSession>(`/sessions/${sessionUUID}`, {
				title: newTitle
			});
			updateSession(updatedSession);
		} catch (e: unknown) {
            if (e instanceof Error) {
			    toast.error($_('errors.update_failed', { values: { message: e.message } }));
            } else {
                toast.error($_('errors.update_failed', { values: { message: 'Unknown error' } }));
            }
			if (originalSession) {
				updateSession(originalSession);
			}
		}
	}
	function newChat(e?: Event) {
		e?.preventDefault();
		e?.stopPropagation();
		const isNewChat = $page.url.pathname.endsWith('/chat/new');
		resetStreamStore();
		if (isNewChat) return;
		goto('/chat/new', { replaceState: true, invalidateAll: false });
	}
</script>
<aside
    class="w-72 bg-secondary/80 backdrop-blur-lg flex flex-col p-4 border-r border-tertiary h-full will-change-transform sticky top-0"
>
	<div class="flex items-center justify-between mb-4 flex-shrink-0">
		<div class="w-10 h-10"></div>
        <a href="/chat" class="flex items-center gap-2 text-2xl font-bold text-text-primary hover:opacity-90 transition-opacity">
            <img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-8 h-8" />
            EGO
        </a>
		<button
			onclick={newChat}
            class="p-2 rounded-lg hover:bg-tertiary/60 active:scale-95 transition-all w-10 h-10 flex items-center justify-center"
			aria-label={$_('chat.new_chat')}
		>
			<Plus class="w-6 h-6" />
		</button>
	</div>
	<div class="flex-1 overflow-y-auto -mr-2 pr-2 min-h-0">
		<nav class="space-y-1">
            {#if $isLoading}
				{#each Array(5) as _}
					<div class="h-10 p-3 bg-tertiary/50 rounded-lg animate-pulse"></div>
				{/each}
            {:else if !$sessions || $sessions.length === 0}
				<p class="text-text-secondary text-sm px-2 py-4 text-center">
					{$_('sidebar.no_sessions')}
				</p>
			{:else}
                {#each ($sessions || []) as session (session.uuid)}
					{@const deleteAriaLabel = $_('sidebar.delete_session_aria', { values: { title: session.title } })}
                    <a
                        href={`/chat/${session.uuid}`}
                        onclick={(e) => { e.preventDefault(); goto(`/chat/${session.uuid}`); }}
                        class="flex items-center justify-between p-3 rounded-lg text-sm group transition-all duration-200 ease-out ring-1 ring-transparent focus-visible:ring-accent/60 focus-visible:outline-none {!renamingId
							? 'hover:bg-tertiary/50'
							: ''}"
						class:bg-accent={!renamingId && $page.params.sessionID === session.uuid}
						class:text-white={!renamingId && $page.params.sessionID === session.uuid}
						class:bg-tertiary={renamingId === session.uuid}
					>
						{#if renamingId === session.uuid}
							<div
								class="w-full flex items-center gap-2"
								transition:slide={{ duration: 150, easing: quintOut }}
							>
								<input
									type="text"
									bind:this={renameInput}
									bind:value={renameInputText}
									onkeydown={(e) => {
										if (e.key === 'Enter') {
											e.preventDefault();
											saveRename();
										}
										if (e.key === 'Escape') cancelRenaming();
									}}
									onblur={saveRename}
									class="flex-grow bg-transparent focus:outline-none text-sm font-medium"
								/>
								<button
									onclick={(e) => {
										e.preventDefault();
										e.stopPropagation();
										saveRename();
									}}
									class="p-1 rounded-md text-green-400 hover:bg-green-500/20"
									aria-label="Сохранить"
								>
									<Check class="w-4 h-4" />
								</button>
								<button
									onclick={(e) => {
										e.preventDefault();
										e.stopPropagation();
										cancelRenaming();
									}}
									class="p-1 rounded-md text-red-400 hover:bg-red-500/20"
									aria-label="Отменить"
								>
									<X class="w-4 h-4" />
								</button>
							</div>
						{:else}
							<div class="flex items-center space-x-3 truncate">
								<MessageSquare class="w-4 h-4 flex-shrink-0" />
								<span class="truncate font-medium">{session.title}</span>
							</div>
							<div class="flex items-center flex-shrink-0">
								<button
									onclick={(e) => startRenaming(session, e)}
									class="p-1 rounded-md text-text-secondary hover:bg-white/20 hover:text-white transition-[background-color,transform] duration-150 md:opacity-0 group-hover:opacity-100 focus:opacity-100 active:scale-95 focus-visible:ring-2 focus-visible:ring-white/50"
									aria-label="Переименовать сессию"
								>
									<Pencil class="w-4 h-4" />
								</button>
								<button
									onclick={(e) => confirmDelete(session.uuid, e)}
									class="p-1 rounded-md text-text-secondary hover:bg-red-500/50 hover:text-white transition-[background-color,transform] duration-150 md:opacity-0 group-hover:opacity-100 focus:opacity-100 active:scale-95 focus-visible:ring-2 focus-visible:ring-red-300/60"
									aria-label={deleteAriaLabel}
								>
									<Trash2 class="w-4 h-4" />
								</button>
							</div>
						{/if}
					</a>
				{/each}
			{/if}
		</nav>
	</div>
	<div class="mt-auto pt-4 border-t border-tertiary flex-shrink-0">
		{#if auth.user}
			<div class="space-y-1 mb-2">
                <button
					onclick={() => setShowSettingsModal(true)}
                    class="flex w-full items-center gap-3 p-3 rounded-lg text-sm text-text-secondary hover:bg-tertiary/60 active:scale-[0.99] transition-all duration-200 ring-1 ring-transparent focus-visible:ring-accent/60 focus-visible:outline-none"
				>
					<Settings class="w-4 h-4" />
					<span>{$_('chat.instructions_title')}</span>
				</button>
                <button
					onclick={() => setShowUserSettingsModal(true)}
                    class="flex w-full items-center gap-3 p-3 rounded-lg text-sm text-text-secondary hover:bg-tertiary/60 active:scale-[0.99] transition-all duration-200 ring-1 ring-transparent focus-visible:ring-accent/60 focus-visible:outline-none"
				>
					<User class="w-4 h-4" />
					<span>{$_('sidebar.user_settings')}</span>
				</button>
                <button
					onclick={() => setShowUserStatsModal(true)}
                    class="flex w-full items-center gap-3 p-3 rounded-lg text-sm text-text-secondary hover:bg-tertiary/60 active:scale-[0.99] transition-all duration-200 ring-1 ring-transparent focus-visible:ring-accent/60 focus-visible:outline-none"
				>
					<BarChart3 class="w-4 h-4" />
					<span>{$_('sidebar.user_stats')}</span>
				</button>
			</div>
			<div class="flex items-center justify-between p-2">
				<span class="text-sm text-text-secondary truncate font-medium" title={auth.user?.username}>
					{auth.user?.username}
				</span>
				<button
					onclick={handleLogout}
					class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-sm font-medium text-red-400 hover:bg-red-500/20 hover:text-red-300 transform hover:scale-105 transition-all duration-200"
				>
					<LogOut class="w-4 h-4" />
					<span>{$_('sidebar.logout')}</span>
				</button>
			</div>
		{/if}
	</div>
</aside>