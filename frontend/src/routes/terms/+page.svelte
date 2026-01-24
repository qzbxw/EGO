<script lang="ts">
	import { goto } from '$app/navigation';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import { getAppLogo } from '$lib/config';
	import { t } from 'svelte-i18n';
	import { Moon, Sun, ArrowLeft, ChevronLeft } from '@lucide/svelte';
	import { fade, fly } from 'svelte/transition';

	function toggleTheme() {
		setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
	}

	// Generate sections 1 through 12
	const sections = Array.from({ length: 12 }, (_, i) => ({
		id: `section-${i + 1}`,
		title: `legal.terms_${i + 1}_title`,
		content: `legal.terms_${i + 1}_content`
	}));
</script>

<svelte:head>
	<title>{$t('legal.terms_full_title')} • EGO</title>
</svelte:head>

<div
	class="min-h-screen w-full overflow-x-hidden transition-colors duration-500 font-sans {preferencesStore.theme ===
	'light'
		? 'bg-[#FDFCF8] text-gray-900'
		: 'bg-[#050508] text-white'}"
>
	<!-- Background Effects -->
	<div class="fixed inset-0 z-0 pointer-events-none">
		{#if preferencesStore.theme === 'light'}
			<div class="absolute inset-0 bg-gradient-to-b from-[#F7F5F0] via-[#FDFCF8] to-[#FDFCF8]"></div>
			<div class="absolute top-[-20%] right-[-10%] w-[70vw] h-[70vw] bg-orange-100/30 blur-[120px] rounded-full"></div>
			<div class="absolute bottom-[-10%] left-[-10%] w-[60vw] h-[60vw] bg-purple-100/30 blur-[150px] rounded-full"></div>
		{:else}
			<div class="absolute inset-0 bg-[#050508]"></div>
			<div class="absolute top-[-10%] left-[-10%] w-[60vw] h-[60vw] bg-purple-900/10 blur-[180px] rounded-full animate-pulse-slow"></div>
			<div class="absolute bottom-[-10%] right-[-10%] w-[50vw] h-[50vw] bg-indigo-900/10 blur-[150px] rounded-full animate-pulse-slow" style="animation-delay: 2s"></div>
		{/if}
		<div class="absolute inset-0 opacity-[0.03] mix-blend-overlay bg-[url('https://grainy-gradients.vercel.app/noise.svg')]"></div>
	</div>

	<!-- Navigation -->
	<nav class="fixed top-0 left-0 right-0 z-50 py-6 transition-all duration-300">
		<div class="max-w-4xl mx-auto px-6 flex items-center justify-between">
			<button 
				class="flex items-center gap-2 group opacity-60 hover:opacity-100 transition-opacity"
				onclick={() => goto('/')}
			>
				<div class="p-2 rounded-full bg-black/5 dark:bg-white/5 group-hover:bg-black/10 dark:group-hover:bg-white/10 transition-colors">
					<ChevronLeft class="w-5 h-5" />
				</div>
				<span class="font-medium hidden sm:block">Back to Home</span>
			</button>

			<div class="flex items-center gap-4">
				<button
					onclick={toggleTheme}
					class="p-2 rounded-full hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
				>
					{#if preferencesStore.theme === 'light'}
						<Moon class="w-5 h-5" />
					{:else}
						<Sun class="w-5 h-5" />
					{/if}
				</button>
				<LanguageSwitcher />
			</div>
		</div>
	</nav>

	<!-- Content -->
	<main class="relative z-10 pt-32 pb-20 px-6">
		<div class="max-w-3xl mx-auto" in:fly={{ y: 20, duration: 600, delay: 200 }}>
			<!-- Header -->
			<header class="mb-12 md:mb-16 text-center">
				<h1 class="text-3xl sm:text-4xl md:text-6xl font-black tracking-tight mb-6 break-words px-2">
					{$t('legal.terms_full_title')}
				</h1>
				<div class="inline-flex flex-col items-center gap-2 opacity-80">
					<div class="text-xs font-bold uppercase tracking-[0.2em] text-accent">{$t('legal.doc_version')}</div>
					<div class="text-base font-medium">{$t('legal.doc_effective')}</div>
				</div>
			</header>

			<!-- Intro -->
			<div class="prose prose-lg max-w-none {preferencesStore.theme === 'dark' ? 'prose-invert' : ''} mb-12 opacity-90 leading-relaxed">
				{@html $t('legal.terms_intro')}
			</div>

			<!-- Legal Text -->
			<div class="prose prose-lg max-w-none {preferencesStore.theme === 'dark' ? 'prose-invert' : ''} prose-headings:font-bold prose-headings:tracking-tight prose-a:text-primary prose-img:rounded-2xl">
				{#each sections as section}
					<section id={section.id} class="mb-12 scroll-mt-32">
						<h2>{$t(section.title)}</h2>
						<div class="opacity-80 leading-relaxed">
							{@html $t(section.content)}
						</div>
					</section>
				{/each}
			</div>

			<!-- Footer -->
			<div class="mt-20 pt-10 border-t border-black/10 dark:border-white/10 text-center opacity-40 text-sm">
				<p>{$t('knowego.footer_copyright_text')}</p>
			</div>
		</div>
	</main>
</div>

<style>
	:global(html) {
		scroll-behavior: smooth;
	}
	.animate-pulse-slow {
		animation: pulse 8s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}
	@keyframes pulse {
		0%, 100% { opacity: 0.1; }
		50% { opacity: 0.15; }
	}
</style>
