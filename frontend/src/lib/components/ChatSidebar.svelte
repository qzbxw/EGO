<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		MessageSquare,
		Plus,
		Trash2,
		Settings,
		LogOut,
		Pencil,
		Check,
		X,
		User,
		BarChart3,
		MoreHorizontal,
		Sparkles
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { slide, fade, fly, scale } from 'svelte/transition';
	import { quintOut, elasticOut, cubicOut } from 'svelte/easing';
	import { auth, logout } from '$lib/stores/auth.svelte.ts';
	import {
		sessions,
		isLoading,
		removeSession,
		clearUserSessions,
		updateSession
	} from '$lib/stores/sessions.svelte.ts';
	import {
		uiStore,
		setShowSettingsModal,
		setShowUserSettingsModal,
		setShowUserStatsModal
	} from '$lib/stores/ui.svelte.ts';
	import { api, apiPy } from '$lib/api';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { resetStreamStore, stopStreamAsCancelled } from '$lib/stores/stream.svelte.ts';
	import { connectionManager } from '$lib/connection-manager';
	import { browser } from '$app/environment';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import SessionListSkeleton from '$lib/components/ui/skeletons/SessionListSkeleton.svelte';
	import type { ChatSession } from '$lib/types';
	import { onMount } from 'svelte';

	// Lazy loading state
	let isSessionsMounted = $state(false);

	onMount(() => {
		// Small delay to prioritize main content render
		setTimeout(() => {
			isSessionsMounted = true;
		}, 100);
	});

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
							// Ensure we clear the stream state so we don't try to reuse the deleted session ID
							resetStreamStore();
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
		if (maintenanceStore.isChatMaintenanceActive) return;
		e?.preventDefault();
		e?.stopPropagation();

		// Immediately stop any active generation
		stopStreamAsCancelled();
		resetStreamStore();

		const isNewChat = $page.url.pathname.endsWith('/chat/new');
		if (isNewChat) return;

		// Optimistic update - clear UI logic if needed, but router will handle the page change
		goto('/chat/new', { replaceState: true, invalidateAll: false });
	}
</script>

<aside
	class="relative flex h-full w-full flex-col overflow-hidden border-r border-black/10 bg-secondary backdrop-blur-2xl transition-all duration-300 dark:border-white/5 dark:bg-secondary/80"
>
	<!-- Header -->
	<div class="relative z-10 flex flex-shrink-0 flex-col gap-4 p-5">
		<div class="flex items-center gap-3 pl-14">
			<div
				role="button"
				tabindex="0"
				class="relative group cursor-pointer outline-none"
				onclick={() => goto('/chat')}
				onkeydown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						goto('/chat');
					}
				}}
			>
				<div class="absolute -inset-2 rounded-full bg-accent/20 blur-lg opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
				<img
					src={getAppLogo(preferencesStore.theme)}
					alt="EGO Logo"
					class="relative h-8 w-8 transition-transform duration-500 group-hover:rotate-12 group-hover:scale-110"
					class:logo-malfunction={maintenanceStore.isChatMaintenanceActive}
				/>
			</div>
			<span class="text-xl font-bold tracking-tight text-text-primary">EGO</span>
		</div>

		<button
			onclick={newChat}
			disabled={maintenanceStore.isChatMaintenanceActive}
			class="group relative w-full overflow-hidden rounded-xl bg-gradient-to-r from-accent to-accent-hover p-[1px] shadow-lg shadow-accent/20 transition-all duration-300 hover:shadow-accent/40 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed disabled:grayscale disabled:hover:shadow-none"
		>
			<div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000"></div>
			<div class="relative flex items-center justify-center gap-2 rounded-[11px] bg-secondary/90 backdrop-blur-sm py-3 text-sm font-bold text-text-primary transition-colors group-hover:bg-transparent group-hover:text-white">
				<Plus class="h-4 w-4 transition-transform duration-300 group-hover:rotate-90" strokeWidth={3} />
				<span>{$_('chat.new_chat')}</span>
			</div>
		</button>
	</div>

	<!-- Scrollable List -->
	<div class="custom-scrollbar flex-1 overflow-y-auto px-3 pb-2 hover:pr-3 transition-[padding]">
		<div class="space-y-1">
			{#if !isSessionsMounted || $isLoading}
				<div class="px-2 pt-2">
					<SessionListSkeleton />
				</div>
			{:else if !$sessions || $sessions.length === 0}
				<div class="flex flex-col items-center justify-center px-4 py-12 text-center" in:fade={{ duration: 400 }}>
					<div class="mb-3 rounded-full bg-tertiary/50 p-3">
						<MessageSquare class="h-6 w-6 text-text-secondary/50" />
					</div>
					<p class="text-sm font-medium text-text-secondary">
						{$_('sidebar.no_sessions')}
					</p>
					<p class="text-xs text-text-secondary/60 mt-1">{$_('sidebar.start_prompt')}</p>
				</div>
			{:else}
				<div class="space-y-1" in:fade={{ duration: 300 }}>
					{#each $sessions || [] as session (session.uuid)}
						{@const isActive = !renamingId && $page.params.sessionID === session.uuid}
						{@const isRenaming = renamingId === session.uuid}
						{@const deleteAriaLabel = $_('sidebar.delete_session_aria', {
							values: { title: session.title }
						})}
						
						<div class="relative group/item">
							<!-- Active Indicator -->
							{#if isActive}
								<div
									layoutId="active-indicator"
									class="absolute left-0 top-1/2 h-8 w-[3px] -translate-y-1/2 rounded-r-full bg-accent shadow-[0_0_10px_rgba(var(--color-accent-rgb),0.5)]"
									in:scale={{ duration: 200, start: 0.5 }}
								></div>
							{/if}

							<a
								href={`/chat/${session.uuid}`}
								onclick={(e) => {
									e.preventDefault();
									goto(`/chat/${session.uuid}`);
								}}
								class="group relative flex items-center justify-between rounded-xl px-3 py-3 text-sm transition-all duration-300 ease-out outline-none
								{isActive 
									? 'bg-accent/10 text-accent font-medium' 
									: 'text-text-secondary hover:bg-black/5 dark:hover:bg-white/5 hover:text-text-primary hover:pl-4'}"
							>
								{#if isRenaming}
									<div
										class="flex w-full items-center gap-2"
										transition:slide={{ duration: 200, axis: 'x' }}
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
											class="w-full min-w-0 bg-transparent text-sm font-medium text-text-primary placeholder-text-secondary/50 focus:outline-none"
											placeholder="New title..."
										/>
										<div class="flex items-center gap-1">
											<button
												onclick={(e) => {
													e.preventDefault();
													e.stopPropagation();
													saveRename();
												}}
												class="rounded-lg bg-green-500/10 p-1.5 text-green-500 hover:bg-green-500/20"
											>
												<Check class="h-3.5 w-3.5" strokeWidth={3} />
											</button>
											<button
												onclick={(e) => {
													e.preventDefault();
													e.stopPropagation();
													cancelRenaming();
												}}
												class="rounded-lg bg-red-500/10 p-1.5 text-red-500 hover:bg-red-500/20"
											>
												<X class="h-3.5 w-3.5" strokeWidth={3} />
											</button>
										</div>
									</div>
								{:else}
									<div class="flex min-w-0 items-center gap-3 overflow-hidden">
										<MessageSquare class="h-4 w-4 flex-shrink-0 transition-transform duration-300 group-hover:scale-110 {isActive ? 'text-accent' : 'text-text-secondary/70'}" />
										<span class="truncate transition-all duration-300 group-hover:translate-x-1">{session.title}</span>
									</div>
									
									<!-- Action Buttons (Hover only) -->
									<div class="absolute right-2 top-1/2 flex -translate-y-1/2 items-center gap-1 opacity-0 transform translate-x-2 transition-all duration-200 group-hover/item:opacity-100 group-hover/item:translate-x-0">
										<div class="absolute right-0 top-0 h-full w-12 bg-gradient-to-l from-secondary via-secondary/70 to-transparent -z-10 dark:from-secondary dark:via-secondary/80 dark:to-transparent"></div>
										<button
											onclick={(e) => startRenaming(session, e)}
											class="rounded-lg bg-secondary p-1.5 text-text-secondary backdrop-blur-sm transition-all hover:bg-black/10 dark:hover:bg-white/20 hover:text-text-primary hover:scale-105 active:scale-90 shadow-sm"
											title="Rename"
										>
											<Pencil class="h-3.5 w-3.5" />
										</button>
										<button
											onclick={(e) => confirmDelete(session.uuid, e)}
											class="rounded-lg bg-secondary p-1.5 text-text-secondary backdrop-blur-sm transition-all hover:bg-red-500/20 hover:text-red-500 hover:scale-105 active:scale-90 shadow-sm"
											title="Delete"
											aria-label={deleteAriaLabel}
										>
											<Trash2 class="h-3.5 w-3.5" />
										</button>
									</div>
								{/if}
							</a>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Footer / User Section -->
	<div class="flex-shrink-0 p-4">
		{#if auth.user}
			<div class="overflow-hidden rounded-2xl border border-black/10 bg-tertiary backdrop-blur-md transition-all duration-300 hover:bg-tertiary/90 hover:shadow-lg hover:shadow-black/5 dark:border-white/5 dark:hover:shadow-black/20">
				
				<!-- Menu Links -->
				<div class="grid grid-cols-3 gap-[1px] bg-black/10 p-[1px] dark:bg-white/5">
					<button
						onclick={() => setShowSettingsModal(true)}
						class="flex flex-col items-center justify-center gap-1 bg-secondary py-3 transition-colors hover:bg-secondary/80 group"
						title={$_('chat.instructions_title')}
					>
						<Settings class="h-4 w-4 text-text-secondary transition-colors group-hover:text-accent" />
					</button>
					<button
						onclick={() => setShowUserSettingsModal(true)}
						class="flex flex-col items-center justify-center gap-1 bg-secondary py-3 transition-colors hover:bg-secondary/80 group"
						title={$_('sidebar.user_settings')}
					>
						<User class="h-4 w-4 text-text-secondary transition-colors group-hover:text-accent" />
					</button>
					<button
						onclick={() => setShowUserStatsModal(true)}
						class="flex flex-col items-center justify-center gap-1 bg-secondary py-3 transition-colors hover:bg-secondary/80 group"
						title={$_('sidebar.user_stats')}
					>
						<BarChart3 class="h-4 w-4 text-text-secondary transition-colors group-hover:text-accent" />
					</button>
				</div>

				<!-- User Profile -->
				<div class="flex items-center justify-between border-t border-black/10 bg-secondary p-3 dark:border-white/5">
					<div class="flex items-center gap-3 overflow-hidden">
						<div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-accent to-purple-600 text-xs font-bold text-white shadow-inner">
							{auth.user.username.substring(0, 2).toUpperCase()}
						</div>
						<div class="flex flex-col truncate">
							<span class="truncate text-xs font-bold text-text-primary">{auth.user.username}</span>
							<span class="text-[10px] text-text-secondary">{$_('sidebar.plan_free')}</span>
						</div>
					</div>
					<button
						onclick={handleLogout}
						class="flex h-8 w-8 items-center justify-center rounded-lg text-text-secondary transition-all hover:bg-red-500/10 hover:text-red-500 active:scale-90"
						title={$_('sidebar.logout')}
					>
						<LogOut class="h-4 w-4" />
					</button>
				</div>
			</div>
		{/if}
	</div>
</aside>

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
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: rgba(156, 163, 175, 0.4);
	}

	.logo-malfunction {
		animation: glitch-malfunction 4s infinite step-end;
		filter: grayscale(0.5) contrast(1.2);
	}

	@keyframes glitch-malfunction {
		0%, 85%, 100% { transform: translate(0); filter: none; opacity: 1; }
		86% { transform: translate(-3px, 1px); filter: hue-rotate(90deg) brightness(1.5); clip-path: inset(10% 0 40% 0); opacity: 0.8; }
		87% { transform: translate(3px, -1px); filter: hue-rotate(-90deg) saturate(3); clip-path: inset(40% 0 10% 0); opacity: 0.9; }
		88% { transform: translate(-1px, 3px); filter: contrast(2) invert(0.1); opacity: 0.7; }
		89% { transform: translate(0); filter: none; clip-path: none; opacity: 1; }
	}
</style>
