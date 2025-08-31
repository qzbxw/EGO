<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte';
	import { getAppLogo } from '$lib/config';
	let isMaintenanceMode = $derived(maintenanceStore.status.maintenance);
	let statusText = $derived(isMaintenanceMode ? 'UNAVAILABLE' : 'AVAILABLE');
	let subtitle = $derived(isMaintenanceMode ? 'Sorry, Service Temporarily Unavailable :(' : '');
	let inputToken = '';
	let tokenError: string | null = null;
	async function submitToken(e: Event) {
		e.preventDefault();
		tokenError = null;
		const token = inputToken.trim();
		if (!token) return;
		const ok = await maintenanceStore.setBypassToken(token);
		if (ok) {
			goto('/');
		} else {
			tokenError = 'Invalid token. Please check and try again.';
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
	<title>EGO - {isMaintenanceMode ? 'Service Unavailable' : 'System Available'}</title>
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700;900&display=swap"
		rel="stylesheet"
	/>
</svelte:head>
<div
	class="min-h-screen flex items-center justify-center p-8 relative font-mono"
	class:theme-dark={preferencesStore.theme === 'dark'}
	class:theme-light={preferencesStore.theme === 'light'}
>
	<div class="fixed top-4 right-4 z-20">
		<button
			class="px-3 py-1.5 text-sm rounded border border-current/30 hover:border-current/60 transition-colors"
			aria-label="Toggle theme"
			on:click={() => setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light')}
		>
			{preferencesStore.theme === 'light' ? '🌙' : '☀️'}
		</button>
	</div>
	<div class="text-center relative z-10">
		<div class="relative mb-12">
			<img
				src={getAppLogo(preferencesStore.theme)}
				alt="EGO Logo"
				class="w-24 h-24 mx-auto mb-8 animate-breath"
			/>
		</div>
		<h1 id="ego-heading"
			class="text-8xl md:text-9xl font-black mb-12 tracking-wider text-wipe"
			class:dark-wipe={preferencesStore.theme === 'dark'}
			class:light-wipe={preferencesStore.theme === 'light'}
		>
			EGO
		</h1>
		<div class="relative mb-8">
			<h2 class="text-4xl md:text-5xl font-black mb-6 leading-tight tracking-tight">
				SYSTEM<br />
				<span
					class="animate-pulse"
					class:text-red-500={preferencesStore.theme === 'light' && isMaintenanceMode}
					class:text-red-400={preferencesStore.theme === 'dark' && isMaintenanceMode}
					class:text-green-500={preferencesStore.theme === 'light' && !isMaintenanceMode}
					class:text-green-400={preferencesStore.theme === 'dark' && !isMaintenanceMode}
				>
					{statusText}
				</span>
			</h2>
			{#if subtitle}
				<p class="text-lg md:text-xl font-bold opacity-70 mb-4">
					{subtitle}
				</p>
			{/if}
			<form class="mt-8 max-w-md mx-auto flex flex-col items-stretch gap-3" on:submit={submitToken} autocomplete="off">
				<label class="text-sm opacity-70 text-left" for="bypassToken">Bypass token</label>
				<input
					id="bypassToken"
					name="bypassToken"
					class="px-3 py-2 rounded border border-current/30 bg-transparent outline-none focus:border-current/60"
					placeholder="Paste maintenance bypass token"
					bind:value={inputToken}
					spellcheck="false"
					autocapitalize="off"
					autocomplete="off"
				/>
				<button type="submit" class="px-4 py-2 rounded border border-current/40 hover:border-current/70 transition-colors">
					Enter
				</button>
				{#if tokenError}
					<div class="text-sm text-red-500">{tokenError}</div>
				{/if}
				{#if maintenanceStore.hasValidBypass}
					<div class="text-xs opacity-60">Token set.</div>
				{/if}
			</form>
		</div>
	</div>
</div>
<style>
	@keyframes breath {
		0%,
		100% {
			transform: scale(1);
			opacity: 0.7;
		}
		50% {
			transform: scale(1.05);
			opacity: 1;
		}
	}
	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.4;
		}
	}
	@keyframes wipe-animation {
		0% {
			background-position: -200% 0;
		}
		100% {
			background-position: 200% 0;
		}
	}
	@keyframes move-gradient {
		0% {
			background-position: 0% 0%;
		}
		100% {
			background-position: 100% 100%;
		}
	}
	:global(body) {
		font-family: 'JetBrains Mono', monospace;
	}
	.theme-dark {
		background-color: #000;
		color: #fff;
		background-image: radial-gradient(circle at top left, #1a1a1a 0%, #000000 50%);
		background-size: 300% 300%;
		animation: move-gradient 20s ease-in-out infinite alternate;
		will-change: background-position;
	}
	.theme-light {
		background-color: #fff;
		color: #000;
		background-image: radial-gradient(circle at top left, #f0f0f0 0%, #ffffff 50%);
		background-size: 300% 300%;
		animation: move-gradient 20s ease-in-out infinite alternate;
		will-change: background-position;
	}
	.animate-breath {
		animation: breath 4s ease-in-out infinite;
	}
	.animate-pulse {
		animation: pulse 2.5s ease-in-out infinite;
	}
	.text-wipe {
		background-size: 200% auto;
		background-clip: text;
		-webkit-background-clip: text;
		color: transparent;
		animation: wipe-animation 8s cubic-bezier(0.4, 0, 0.2, 1) infinite reverse;
		will-change: background-position;
	}
	#ego-heading {
		font-weight: 900; 
		font-family: 'JetBrains Mono', monospace;
		-webkit-font-smoothing: antialiased;
		-moz-osx-font-smoothing: grayscale;
		text-rendering: optimizeLegibility;
	}
	.dark-wipe {
		background-image: linear-gradient(110deg, white 40%, black 50%, white 60%);
	}
	.light-wipe {
		background-image: linear-gradient(110deg, black 40%, white 50%, black 60%);
	}
</style>