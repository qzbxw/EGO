<script lang="ts">
import ChatSidebar from '$lib/components/ChatSidebar.svelte';
import { Menu } from '@lucide/svelte';
import { fly } from 'svelte/transition';
import { page } from '$app/stores';
import { Toaster } from 'svelte-sonner';
import { locale } from 'svelte-i18n';
import { browser } from '$app/environment';
import { streamStore } from '$lib/stores/stream.svelte.ts';
import { loadUserSettings } from '$lib/stores/user-settings.svelte.ts';
import { onMount } from 'svelte';
import { preferencesStore, loadPreferences } from '$lib/stores/preferences.svelte.ts';
import GlobalLoader from '$lib/components/GlobalLoader.svelte';
import { loadingStore } from '$lib/stores/loading.svelte.ts';
import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
import { goto } from '$app/navigation';
import UserStatsModal from '$lib/components/UserStatsModal.svelte';
import UserSettingsModal from '$lib/components/UserSettingsModal.svelte';
    const { children } = $props();
    let showSidebar = $state(true);
    	let isMobile = $state(false);
	let isAuthPage = $derived($page.route.id === '/login' || $page.route.id === '/register');
	let isMaintenancePage = $derived($page.url.pathname === '/maintenance');
	let isAuthenticatingPage = $derived($page.url.pathname === '/authenticating');
	let hideSidebar = $derived(
		isAuthPage ||
		isAuthenticatingPage ||
		$page.route.id === '/knowego' ||
		isMaintenancePage ||
		['/terms', '/privacy'].includes($page.url.pathname)
	);
    onMount(async () => {
        if (browser) {
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
                    } catch {}
                }
            } catch {}
            loadUserSettings();
            loadPreferences();
            maintenanceStore.checkStatus();
			if (!maintenanceStore.isMaintenanceActive) {
			}
        }
    });
	$effect(() => {
		if (browser && $locale) {
			localStorage.setItem('locale', $locale);
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
    $effect(() => {
        if (browser && maintenanceStore) {
            if (maintenanceStore.isMaintenanceActive && !isMaintenancePage) {
                console.log('Maintenance is active, redirecting to /maintenance');
                const params = new URLSearchParams(window.location.search);
                const urlToken = params.get('bypass_token') || params.get('token');
                const target = urlToken ? `/maintenance?bypass_token=${encodeURIComponent(urlToken)}` : '/maintenance';
                goto(target);
            } 
            else if (!maintenanceStore.status.maintenance && isMaintenancePage) {
                console.log('Maintenance is disabled, redirecting to /');
                goto('/');
            }
        }
    });
</script>
<svelte:head>
    <link rel="manifest" href="/manifest.json" />
    <link rel="icon" href="/agentmodelogodark.png" type="image/png" />
    <meta name="theme-color" content={preferencesStore.theme === 'light' ? '#fff4e6' : '#000000'} />
</svelte:head>
<Toaster position="top-right" richColors closeButton duration={4200} theme={preferencesStore.theme} visibleToasts={6} toastOptions={{ class: 'border border-tertiary/70 shadow-2xl shadow-black/40' }} />
<div
    class="fixed inset-0 z-0 overflow-hidden bg-orb will-change-transform pointer-events-none"
    class:theme-light={preferencesStore.theme === 'light'}
    class:thinking={!streamStore.isDone}
    style="--dur: {streamStore.isDone ? '36s' : '12s'}; --jitter-duration: {streamStore.isDone ? '7s' : '4s'}; transition: opacity 400ms ease, transform 600ms ease;"
    class:hidden={!preferencesStore.backgroundSpheres}
>
    <div class="bg-circle -right-[20vw] -top-[20vh]" style="animation-delay: -10s;"></div>
    <div class="bg-circle secondary -left-[20vw] -bottom-[20vh]"></div>
    <div class="pointer-events-none absolute inset-0" style="animation: jitter var(--jitter-duration) ease-in-out infinite alternate; will-change: transform;"></div>
</div>
<div class="min-h-screen w-full font-sans text-text-primary" class:theme-light={preferencesStore.theme === 'light'} style="background-color: rgb(var(--color-primary-rgb)); --sidebar-offset: {(!hideSidebar && showSidebar && !isMobile) ? '18rem' : '0px'};">
    {#if !hideSidebar}
        <div
            class="fixed top-0 left-0 h-full w-72 z-40 transition-transform duration-400 ease-[cubic-bezier(0.22,1,0.36,1)]"
            class:translate-x-0={showSidebar}
            class:-translate-x-full={!showSidebar}
        >
            <ChatSidebar />
        </div>
    {/if}
    {#if !hideSidebar && isMobile && showSidebar}
        <div
            onclick={() => (showSidebar = false)}
            onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (showSidebar = false)}
            role="button"
            tabindex="0"
            aria-label="Закрыть меню"
            class="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm transition-opacity duration-400 ease-out"
            in:fly={{ duration: 320, opacity: 0 }}
            out:fly={{ duration: 280, opacity: 0 }}
        ></div>
    {/if}
    {#if !hideSidebar}
    <button
        onclick={() => (showSidebar = !showSidebar)}
        class="fixed top-3 left-3 z-50 p-2 rounded-md bg-secondary/50 backdrop-blur-sm hover:bg-tertiary transition-[background-color,transform] duration-200 ease-out active:scale-95"
        title="Переключить меню"
    >
        <Menu class="w-6 h-6" />
    </button>
    {/if}
    <main
        class="relative min-h-screen w-full flex flex-col transition-[padding-left] duration-400 ease-[cubic-bezier(0.22,1,0.36,1)]"
        class:md:pl-72={!hideSidebar && showSidebar && !isMobile}
    >
        <div class="flex flex-col flex-1">
            {@render children?.()}
        </div>
    </main>
    <GlobalLoader 
        show={loadingStore.isLoading} 
        text={loadingStore.text} 
        variant={loadingStore.variant} 
    />
    <UserStatsModal />
    <UserSettingsModal />
</div>