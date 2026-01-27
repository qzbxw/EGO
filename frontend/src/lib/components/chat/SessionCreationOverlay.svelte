<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	let { show = false }: { show: boolean } = $props();
</script>

{#if show}
	<div
		class="absolute inset-0 z-40 flex items-center justify-center bg-primary/60 backdrop-blur-sm"
	>
		<div class="not-prose flex flex-col items-center justify-center px-4 text-center">
			<img
				src={getAppLogo(preferencesStore.theme)}
				alt="EGO Logo"
				class="mb-3 h-16 w-16 text-accent"
				class:logo-malfunction={maintenanceStore.isChatMaintenanceActive}
			/>
			<div
				class="loading-bubble-animation mx-1 inline-flex items-center gap-2 rounded-xl bg-secondary px-3.5 py-2 shadow-lg md:mx-2"
			>
				<div
					class="animate-shine bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] bg-clip-text text-sm font-medium text-transparent"
				>
					{$_('chat.creating_session') || 'Создаём сессию для вас…'}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.logo-malfunction {
		animation: glitch-malfunction 4s infinite step-end;
		filter: grayscale(0.5) contrast(1.2);
	}

	@keyframes glitch-malfunction {
		0%,
		85%,
		100% {
			transform: translate(0);
			filter: none;
			opacity: 1;
		}
		86% {
			transform: translate(-3px, 1px);
			filter: hue-rotate(90deg) brightness(1.5);
			clip-path: inset(10% 0 40% 0);
			opacity: 0.8;
		}
		87% {
			transform: translate(3px, -1px);
			filter: hue-rotate(-90deg) saturate(3);
			clip-path: inset(40% 0 10% 0);
			opacity: 0.9;
		}
		88% {
			transform: translate(-1px, 3px);
			filter: contrast(2) invert(0.1);
			opacity: 0.7;
		}
		89% {
			transform: translate(0);
			filter: none;
			clip-path: none;
			opacity: 1;
		}
	}
</style>
