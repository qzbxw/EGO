<script lang="ts">
	import { goto } from '$app/navigation';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { _ } from 'svelte-i18n';
	import { fly, fade, scale } from 'svelte/transition';
	import { cubicOut, elasticOut } from 'svelte/easing';
	import { Sparkles, Brain, Terminal, Search, ArrowRight } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import ChatMaintenanceOverlay from '$lib/components/ChatMaintenanceOverlay.svelte';

	let visible = $state(false);

	onMount(() => {
		visible = true;
	});

	const features = [
		{ icon: Brain, label: 'Memory', color: 'text-purple-400', bg: 'bg-purple-500/10', delay: 400, pos: 'top-0 left-1/4 -translate-y-12 -translate-x-12' },
		{ icon: Search, label: 'Search', color: 'text-blue-400', bg: 'bg-blue-500/10', delay: 500, pos: 'top-1/4 right-0 translate-x-12' },
		{ icon: Terminal, label: 'Code', color: 'text-emerald-400', bg: 'bg-emerald-500/10', delay: 600, pos: 'bottom-0 left-0 -translate-x-4 translate-y-4' },
		{ icon: Sparkles, label: 'Reasoning', color: 'text-amber-400', bg: 'bg-amber-500/10', delay: 700, pos: 'bottom-1/4 right-1/4 translate-y-12 translate-x-8' }
	];
</script>

<div class="relative flex flex-1 flex-col items-center justify-center overflow-hidden p-6">
	
	<!-- Central Content -->
	<div class="relative z-10 flex max-w-2xl flex-col items-center text-center">
		
		<!-- Floating Feature Chips (Decorative) -->
		{#if visible}
			<div class="absolute inset-0 pointer-events-none hidden md:block">
				{#each features as feature, i}
					<div
						class="absolute {feature.pos} flex items-center gap-2 rounded-full border border-white/5 {feature.bg} px-3 py-1.5 backdrop-blur-sm shadow-lg"
						in:fly={{ y: 20, duration: 800, delay: feature.delay, easing: cubicOut }}
						style="animation: float 6s ease-in-out infinite; animation-delay: {i * 1.5}s;"
					>
						<feature.icon class="h-3.5 w-3.5 {feature.color}" />
						<span class="text-xs font-semibold text-text-secondary/80">{feature.label}</span>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Logo -->
		{#if visible}
			<div 
				class="group relative mb-8"
				in:scale={{ duration: 600, start: 0.8, easing: elasticOut }}
			>
				<div class="absolute -inset-8 rounded-full bg-accent/20 blur-3xl transition-all duration-700 group-hover:bg-accent/30 group-hover:blur-[40px]"></div>
				<img 
					src={getAppLogo(preferencesStore.theme)} 
					alt="EGO Logo" 
					class="relative h-32 w-32 drop-shadow-2xl transition-transform duration-700 ease-in-out group-hover:rotate-[10deg] group-hover:scale-110"
					class:logo-malfunction={maintenanceStore.isChatMaintenanceActive}
					style="will-change: transform;"
				/>
			</div>
		{/if}

		<!-- Title -->
		{#if visible}
			<h1 
				class="mb-4 text-5xl font-extrabold tracking-tight text-transparent bg-clip-text bg-gradient-to-br from-text-primary via-text-primary to-text-secondary md:text-7xl"
				in:fly={{ y: 20, duration: 600, delay: 200, easing: cubicOut }}
			>
				EGO
			</h1>
		{/if}

		<!-- Subtitle -->
		{#if visible}
			<p 
				class="mb-10 max-w-md text-lg text-text-secondary/80 leading-relaxed md:text-xl"
				in:fly={{ y: 20, duration: 600, delay: 300, easing: cubicOut }}
			>
				{$_('chat.landing_subtitle') || 'Your intelligent companion for complex tasks, analysis, and creative work.'}
			</p>
		{/if}

		<!-- CTA Button / Maintenance Overlay -->
		{#if visible}
			<div in:fly={{ y: 20, duration: 600, delay: 400, easing: cubicOut }}>
				{#if maintenanceStore.isChatMaintenanceActive}
					<div class="mt-4 max-w-lg mx-auto">
						<ChatMaintenanceOverlay />
					</div>
				{:else}
					<button
						onclick={() => goto('/chat/new')}
						class="group relative overflow-hidden rounded-full bg-accent px-8 py-4 font-bold text-white shadow-xl shadow-accent/25 transition-all duration-300 hover:scale-105 hover:shadow-accent/40 active:scale-95"
					>
						<div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full transition-transform duration-1000 group-hover:translate-x-full"></div>
						<span class="relative flex items-center gap-2">
							{$_('chat.landing_button')}
							<ArrowRight class="h-5 w-5 transition-transform duration-300 group-hover:translate-x-1" />
						</span>
					</button>
				{/if}
			</div>
		{/if}
	</div>
</div>

<style>
	@keyframes float {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(-10px); }
	}

	.logo-malfunction {
		animation: glitch-malfunction 4s infinite step-end;
		filter: grayscale(0.5) contrast(1.2);
	}

	@keyframes glitch-malfunction {
		0%, 85%, 100% { transform: translate(0); filter: none; opacity: 1; }
		86% { transform: translate(-5px, 1px); filter: hue-rotate(90deg) brightness(1.5); clip-path: inset(10% 0 40% 0); opacity: 0.8; }
		87% { transform: translate(5px, -1px); filter: hue-rotate(-90deg) saturate(3); clip-path: inset(40% 0 10% 0); opacity: 0.9; }
		88% { transform: translate(-1px, 5px); filter: contrast(2) invert(0.1); opacity: 0.7; }
		89% { transform: translate(0); filter: none; clip-path: none; opacity: 1; }
	}
</style>
