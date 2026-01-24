<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte';
	import { getAppLogo } from '$lib/config';
	import { fade, scale } from 'svelte/transition';

	let isMaintenanceMode = $derived(maintenanceStore.status.maintenance);
	let statusText = $derived(isMaintenanceMode ? 'MAINTENANCE' : 'SYSTEM ACTIVE');
	let subtitle = $derived(
		isMaintenanceMode
			? "We'll be back shortly"
			: 'Synchronizing...'
	);
	let inputToken = $state('');
	let tokenError = $state<string | null>(null);
	let isSubmitting = $state(false);

	async function submitToken(e: Event) {
		e.preventDefault();
		tokenError = null;
		const token = inputToken.trim();
		if (!token) return;

		isSubmitting = true;
		const ok = await maintenanceStore.setBypassToken(token);
		isSubmitting = false;

		if (ok) {
			goto('/');
		} else {
			tokenError = 'UNAUTHORIZED_ACCESS';
		}
	}

	onMount(() => {
		maintenanceStore.checkStatus();
		const interval = setInterval(async () => {
			await maintenanceStore.checkStatus();
			if (!maintenanceStore.isMaintenanceActive) {
				clearInterval(interval);
				goto('/');
			}
		}, 10000);

		return () => clearInterval(interval);
	});
</script>

<svelte:head>
	<title>EGO | {isMaintenanceMode ? 'Maintenance' : 'Online'}</title>
</svelte:head>

<div
	class="relative flex min-h-screen flex-col items-center justify-center overflow-hidden p-6 font-mono transition-colors duration-1000"
	class:theme-dark={preferencesStore.theme === 'dark'}
	class:theme-light={preferencesStore.theme === 'light'}
>
	<!-- Ambient Background -->
	<div class="bg-orb pointer-events-none fixed inset-0 z-0 opacity-40">
		<div class="bg-circle" style="--dur: 25s; top: -10%; left: -10%;"></div>
		<div class="bg-circle secondary" style="--dur: 35s; bottom: -15%; right: -5%;"></div>
	</div>

	<!-- Scanline Effect -->
	<div class="scanlines pointer-events-none fixed inset-0 z-50 opacity-[0.03]"></div>

	<!-- Top Bar -->
	<div class="fixed left-0 right-0 top-0 z-20 flex items-center justify-between p-6">
		<div class="flex items-center gap-4">
			<div class="flex items-center gap-2 opacity-60">
				<div
					class="h-1.5 w-1.5 rounded-full"
					class:bg-red-500={isMaintenanceMode}
					class:bg-green-500={!isMaintenanceMode}
					class:animate-pulse={isMaintenanceMode}
				></div>
				<span class="text-[10px] font-black tracking-[0.3em] uppercase">EGO_v2</span>
			</div>
		</div>
		<button
			class="border-current/10 hover:border-current/40 flex h-10 w-10 items-center justify-center rounded-full border bg-secondary/20 backdrop-blur-md transition-all hover:scale-105"
			onclick={() => setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light')}
		>
			<span class="text-sm">{preferencesStore.theme === 'light' ? '🌙' : '☀️'}</span>
		</button>
	</div>

	<!-- Main Content -->
	<div class="relative z-10 w-full max-w-xl" in:scale={{ duration: 800, start: 0.98 }}>
		<div class="animated-border-card group flex flex-col items-center overflow-hidden p-10 md:p-16">
			<!-- Logo -->
			<div class="relative mb-10">
				<div
					class="absolute inset-0 scale-150 rounded-full bg-accent/10 blur-3xl transition-all group-hover:bg-accent/20"
				></div>
				<div class="logo-container" class:is-glitching={isMaintenanceMode}>
					<img
						src={getAppLogo(preferencesStore.theme)}
						alt="EGO Logo"
						class="relative h-24 w-24 object-contain transition-all"
					/>
				</div>
			</div>

			<!-- Status Info -->
			<div class="mb-10 text-center">
				<h1
					class="mb-2 text-4xl font-bold tracking-tight sm:text-6xl uppercase"
					class:logo-malfunction={isMaintenanceMode}
					class:text-red-500={isMaintenanceMode}
					class:text-green-500={!isMaintenanceMode}
				>
					{statusText}
				</h1>
				<p class="text-text-secondary text-xs font-medium tracking-widest uppercase opacity-60">
					{subtitle}
				</p>
			</div>

			<!-- Bypass Form -->
			{#if isMaintenanceMode}
				<form
					class="mt-2 flex w-full max-w-xs flex-col gap-3"
					onsubmit={submitToken}
					autocomplete="off"
					in:fade={{ delay: 400 }}
				>
					<div class="relative">
						<input
							type="password"
							class="auth-input !pl-0 !pr-0 text-center text-xs tracking-widest placeholder:opacity-30"
							placeholder="Access key"
							bind:value={inputToken}
							spellcheck="false"
							autocapitalize="off"
						/>
					</div>

					<button
						type="submit"
						class="btn-gradient px-6 py-3 text-xs font-bold tracking-widest uppercase disabled:opacity-30"
						disabled={isSubmitting || !inputToken}
					>
						{isSubmitting ? 'Verifying...' : 'Unlock'}
					</button>

					{#if tokenError}
						<p class="text-center text-[10px] font-bold text-red-500 uppercase tracking-widest" in:fade>
							Invalid key
						</p>
					{/if}
				</form>
			{/if}
		</div>
	</div>
</div>

<style>
	.logo-container {
		position: relative;
		display: flex;
		justify-content: center;
		align-items: center;
	}

	.logo-container.is-glitching img {
		animation: logo-glitch 4s infinite step-end;
	}

	@keyframes logo-glitch {
		0%, 85%, 100% { transform: translate(0); filter: none; }
		86% { transform: translate(-5px, 2px); filter: hue-rotate(90deg) brightness(1.3); clip-path: inset(10% 0 40% 0); }
		87% { transform: translate(5px, -2px); filter: hue-rotate(-90deg) saturate(3); clip-path: inset(40% 0 10% 0); }
		88% { transform: translate(-2px, 5px); filter: contrast(2); }
		89% { transform: translate(0); filter: none; clip-path: none; }
	}

	.logo-malfunction {
		animation: glitch-malfunction 6s infinite step-end;
		filter: contrast(1.1);
	}

	@keyframes glitch-malfunction {
		0%, 86%, 100% { transform: translate(0); filter: none; opacity: 1; text-shadow: none; }
		87% { transform: translate(-2px, 1px); filter: hue-rotate(90deg) saturate(1.5); clip-path: inset(10% 0 40% 0); color: #ff003c; opacity: 0.9; text-shadow: 2px 0 #00fff2; }
		88% { transform: translate(2px, -1px); filter: hue-rotate(-90deg) contrast(2); clip-path: inset(40% 0 10% 0); color: #ff003c; opacity: 0.8; text-shadow: -2px 0 #00fff2; }
		89% { transform: translate(0); filter: none; clip-path: none; opacity: 1; text-shadow: none; }
		90% { transform: translate(-3px, 2px); filter: hue-rotate(45deg); clip-path: inset(5% 0 60% 0); color: #ff003c; opacity: 0.7; }
		91% { transform: translate(0); filter: none; clip-path: none; opacity: 1; }
	}

	.scanlines {
		background: linear-gradient(
			to bottom,
			rgba(255, 255, 255, 0) 50%,
			rgba(0, 0, 0, 0.1) 50%
		);
		background-size: 100% 4px;
	}

	:global(body) {
		background-color: #000;
	}

	.theme-light { background-color: #f8fafc; }
	.theme-dark { background-color: #000; }
</style>