<script lang="ts">
	import ChatSidebar from '$lib/components/ChatSidebar.svelte';
	import { Menu } from '@lucide/svelte';
	import { fly, fade } from 'svelte/transition';
	import { cubicOut, quintOut, backOut } from 'svelte/easing';
	import { page } from '$app/stores';
	import { Toaster } from 'svelte-sonner';
	import { locale } from 'svelte-i18n';
	import { browser } from '$app/environment';
	import { streamStore } from '$lib/stores/stream.svelte.ts';
	import { loadUserSettings } from '$lib/stores/user-settings.svelte.ts';
	import { onMount, onDestroy } from 'svelte';
	import { preferencesStore, loadPreferences } from '$lib/stores/preferences.svelte.ts';
	import GlobalLoader from '$lib/components/GlobalLoader.svelte';
	import { loadingStore } from '$lib/stores/loading.svelte.ts';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import { goto } from '$app/navigation';
	import UserStatsModal from '$lib/components/UserStatsModal.svelte';
	import UserSettingsModal from '$lib/components/UserSettingsModal.svelte';
	import { connectionManager } from '$lib/connection-manager';
	import { auth, initAuthStore } from '$lib/stores/auth.svelte.ts';
	import type { LayoutData } from './$types';
	const { children, data } = $props<{ children?: any; data: LayoutData }>();
	
	// Layout state
	let showSidebar = $state(true);
	let isMobile = $state(false);

	// Derived values
	let isAuthPage = $derived($page.route.id === '/login' || $page.route.id === '/register');
	let isMaintenancePage = $derived($page.url.pathname === '/maintenance');
	let hideSidebar = $derived(
		isAuthPage ||
			isMaintenancePage ||
			['/', '/terms', '/privacy', '/knowego'].includes($page.url.pathname)
	);
	onMount(async () => {
		if (browser) {
			// Ensure auth store is initialized from localStorage on first client load
			initAuthStore();
			try {
				const params = new URLSearchParams(window.location.search);
				const urlToken = params.get('bypass_token') || params.get('token');
				if (urlToken) {
					const { storeBypassToken } = await import('$lib/api/maintenance');
					console.log('🔧 Layout mount: storing bypass_token from URL');
					storeBypassToken(urlToken);
					try {
						const u = new URL(window.location.href);
						u.searchParams.delete('bypass_token');
						u.searchParams.delete('token');
						window.history.replaceState({}, '', u.toString());
					} catch (error) {
						console.error('[Layout] Failed to clean URL parameters:', error);
					}
				}
			} catch (error) {
				console.error('[Layout] Failed to process bypass token:', error);
			}
			loadUserSettings();
			// Load preferences with SSR data to prevent FOUC
			loadPreferences({
				theme: data.theme,
				backgroundSpheres: data.backgroundSpheres,
				memoryEnabled: data.memoryEnabled
			});
			// Note: maintenanceStore.checkStatus() is already called in store initialization
			// No need to call it here to avoid duplicate requests

			// Initialize WebSocket connection manager
			connectionManager.initializeVisibilityHandling();

			// Note: WebSocket connection is handled by the $effect block below
			// to react to auth and maintenance status changes
		}
	});

	onDestroy(() => {
		if (browser) {
			connectionManager.destroyVisibilityHandling();
		}
	});
	if (browser) {
		$effect(() => {
			if ($locale) {
				localStorage.setItem('locale', $locale);
			}
		});
		
		// Client-side guard for protected routes (e.g. /chat)
		$effect(() => {
			const pathname = $page.url.pathname;
			const isProtectedRoute = ![
				'/login',
				'/register',
				'/',
				'/terms',
				'/privacy',
				'/knowego',
				'/maintenance'
			].includes(pathname);
			if (!isProtectedRoute) return;
			// If there are no tokens at all, user is effectively logged out
			const hasAnyToken = Boolean(auth.accessToken || auth.refreshToken);
			if (!hasAnyToken) {
				goto('/login');
			}
		});
		
		$effect(() => {
			const mediaQuery = window.matchMedia('(max-width: 768px)');
			function handleResize(e: MediaQueryListEvent | MediaQueryList) {
				isMobile = e.matches;
			}
			mediaQuery.addEventListener('change', handleResize);
			handleResize(mediaQuery);
			return () => {
				mediaQuery.removeEventListener('change', handleResize);
			};
		});
	}

			// Single effect to manage auth and maintenance status
		if (browser) {
			$effect(() => {
				if (maintenanceStore) {
					if (maintenanceStore.isMaintenanceActive && !isMaintenancePage) {
						console.log('Full maintenance is active, redirecting to /maintenance');
						const params = new URLSearchParams(window.location.search);
						const urlToken = params.get('bypass_token') || params.get('token');
						const target = urlToken
							? `/maintenance?bypass_token=${encodeURIComponent(urlToken)}`
							: '/maintenance';
						goto(target);
					} else if (!maintenanceStore.status.maintenance && isMaintenancePage) {
						console.log('Maintenance is disabled or bypassed, redirecting to /');
						goto('/');
					}
				}
			});
		}</script>

<svelte:head>
	<title>EGO</title>
	<link rel="manifest" href="/manifest.json" />
	<link rel="icon" href={data.theme === 'light' ? '/logolight.png' : '/logodark.png'} type="image/png" />
	<meta name="theme-color" content={data.theme === 'light' ? '#fff4e6' : '#000000'} />
	{#if data.theme === 'light'}
		<script>
			// Apply theme class immediately to prevent FOUC
			if (typeof document !== 'undefined') {
				document.documentElement.classList.add('theme-light');
			}
		</script>
	{/if}
</svelte:head>
<Toaster
	position="top-right"
	richColors
	closeButton
	duration={4200}
	theme={browser ? preferencesStore.theme : 'dark'}
	visibleToasts={6}
/>
<div
	class="bg-orb pointer-events-none fixed inset-0 z-0 overflow-hidden will-change-transform"
	class:theme-light={browser ? preferencesStore.theme === 'light' : false}
	class:thinking={browser ? !streamStore.isDone : false}
	style="--dur: {browser ? (streamStore.isDone ? '36s' : '12s') : '36s'}; --jitter-duration: {browser ? (streamStore.isDone
		? '7s'
		: '4s') : '7s'}; transition: opacity 400ms ease, transform 600ms ease;"
	class:hidden={browser ? !preferencesStore.backgroundSpheres : true}
>
	<div class="bg-circle -right-[20vw] -top-[20vh]" style="animation-delay: -10s;"></div>
	<div class="bg-circle secondary -bottom-[20vh] -left-[20vw]"></div>
	<div
		class="pointer-events-none absolute inset-0"
		style="animation: jitter var(--jitter-duration) ease-in-out infinite alternate; will-change: transform;"
	></div>
</div>
<div
	class="min-h-screen w-full font-sans text-text-primary"
	class:theme-light={browser ? preferencesStore.theme === 'light' : false}
	style="background-color: rgb(var(--color-primary-rgb)); --sidebar-offset: {browser && !hideSidebar && showSidebar && !isMobile
		? '18rem'
		: '0px'};"
>
	{#if browser && !hideSidebar}
		<div
			class="duration-500 fixed left-0 top-0 z-40 h-full w-72 transform shadow-2xl transition-all ease-[cubic-bezier(0.25,0.8,0.25,1)] will-change-transform"
			class:translate-x-0={showSidebar}
			class:-translate-x-full={!showSidebar}
			class:opacity-100={showSidebar}
			class:opacity-0={!showSidebar}
		>
			<ChatSidebar />
		</div>
	{/if}
	{#if browser && !hideSidebar && isMobile && showSidebar}
		<div
			onclick={() => (showSidebar = false)}
			onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (showSidebar = false)}
			role="button"
			tabindex="0"
			aria-label="Закрыть меню"
			class="duration-500 fixed inset-0 z-30 bg-black/60 backdrop-blur-[2px] transition-all ease-out"
			in:fade={{ duration: 400 }}
			out:fade={{ duration: 300 }}
		></div>
	{/if}
	{#if browser && !hideSidebar}
		<button
			onclick={() => (showSidebar = !showSidebar)}
			class="fixed left-3 top-3 z-50 rounded-full bg-secondary/50 p-2.5 backdrop-blur-md transition-all duration-300 ease-out hover:bg-tertiary hover:scale-110 active:scale-95 shadow-lg border border-black/5 dark:border-white/5"
			title="Переключить меню"
		>
			<Menu class="h-5 w-5" />
		</button>
	{/if}
	<main
		class="duration-500 relative flex min-h-screen w-full flex-col transition-[padding-left] ease-[cubic-bezier(0.25,0.8,0.25,1)]"
		class:md:pl-72={browser && !hideSidebar && showSidebar && !isMobile}
	>
		<div class="flex flex-1 flex-col" in:fade={{ duration: 300, delay: 100 }}>
			{@render children?.()}
		</div>
	</main>
	{#if browser}
		<GlobalLoader
			show={loadingStore.isLoading}
			text={loadingStore.text}
			variant={loadingStore.variant}
		/>
		<UserStatsModal />
		<UserSettingsModal />
	{/if}
</div>
