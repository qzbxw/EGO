<script lang="ts">
	import { goto } from '$app/navigation';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import { t } from 'svelte-i18n';
	import { Moon, Sun, ChevronLeft } from '@lucide/svelte';
	import { fly } from 'svelte/transition';

	function toggleTheme() {
		setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
	}

	// Generate sections 1 through 10
	const sections = Array.from({ length: 10 }, (_, i) => ({
		id: `section-${i + 1}`,
		title: `legal.privacy_${i + 1}_title`,
		content: `legal.privacy_${i + 1}_content`
	}));
</script>

<svelte:head>
	<title>{$t('legal.privacy_full_title')} • EGO</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div
	class="min-h-screen w-full overflow-x-hidden font-sans transition-colors duration-500 {preferencesStore.theme ===
	'light'
		? 'bg-[#FDFCF8] text-gray-900'
		: 'bg-[#050508] text-white'}"
>
	<!-- Background Effects -->
	<div class="pointer-events-none fixed inset-0 z-0">
		{#if preferencesStore.theme === 'light'}
			<div
				class="absolute inset-0 bg-gradient-to-b from-[#F7F5F0] via-[#FDFCF8] to-[#FDFCF8]"
			></div>
			<div
				class="absolute right-[-10%] top-[-20%] h-[70vw] w-[70vw] rounded-full bg-orange-100/30 blur-[120px]"
			></div>
			<div
				class="absolute bottom-[-10%] left-[-10%] h-[60vw] w-[60vw] rounded-full bg-purple-100/30 blur-[150px]"
			></div>
		{:else}
			<div class="absolute inset-0 bg-[#050508]"></div>
			<div
				class="animate-pulse-slow absolute left-[-10%] top-[-10%] h-[60vw] w-[60vw] rounded-full bg-purple-900/10 blur-[180px]"
			></div>
			<div
				class="animate-pulse-slow absolute bottom-[-10%] right-[-10%] h-[50vw] w-[50vw] rounded-full bg-indigo-900/10 blur-[150px]"
				style="animation-delay: 2s"
			></div>
		{/if}
		<div
			class="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-[0.03] mix-blend-overlay"
		></div>
	</div>

	<!-- Navigation -->
	<nav class="fixed left-0 right-0 top-0 z-50 py-6 transition-all duration-300">
		<div class="mx-auto flex max-w-4xl items-center justify-between px-6">
			<button
				class="group flex items-center gap-2 opacity-60 transition-opacity hover:opacity-100"
				onclick={() => void goto('/')}
			>
				<div
					class="rounded-full bg-black/5 p-2 transition-colors group-hover:bg-black/10 dark:bg-white/5 dark:group-hover:bg-white/10"
				>
					<ChevronLeft class="h-5 w-5" />
				</div>
				<span class="hidden font-medium sm:block">Back to Home</span>
			</button>

			<div class="flex items-center gap-4">
				<button
					onclick={toggleTheme}
					class="rounded-full p-2 transition-colors hover:bg-black/5 dark:hover:bg-white/5"
				>
					{#if preferencesStore.theme === 'light'}
						<Moon class="h-5 w-5" />
					{:else}
						<Sun class="h-5 w-5" />
					{/if}
				</button>
				<LanguageSwitcher />
			</div>
		</div>
	</nav>

	<!-- Content -->
	<main class="relative z-10 px-6 pb-20 pt-32">
		<div class="mx-auto max-w-3xl" in:fly={{ y: 20, duration: 600, delay: 200 }}>
			<!-- Header -->
			<header class="mb-12 text-center md:mb-16">
				<h1
					class="mb-6 break-words px-2 text-3xl font-black tracking-tight sm:text-4xl md:text-6xl"
				>
					{$t('legal.privacy_full_title')}
				</h1>
				<div class="inline-flex flex-col items-center gap-2 opacity-80">
					<div class="text-xs font-bold uppercase tracking-[0.2em] text-accent">
						{$t('legal.doc_version')}
					</div>
					<div class="text-base font-medium">{$t('legal.doc_effective')}</div>
				</div>
			</header>

			<!-- Intro -->
			<div
				class="prose prose-lg max-w-none {preferencesStore.theme === 'dark'
					? 'prose-invert'
					: ''} mb-12 leading-relaxed opacity-90"
			>
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html $t('legal.privacy_intro')}
			</div>

			<!-- Legal Text -->
			<div
				class="prose prose-lg max-w-none {preferencesStore.theme === 'dark'
					? 'prose-invert'
					: ''} prose-headings:font-bold prose-headings:tracking-tight prose-a:text-primary prose-img:rounded-2xl"
			>
				{#each sections as section (section.id)}
					<section id={section.id} class="mb-12 scroll-mt-32">
						<h2>{$t(section.title)}</h2>
						<div class="leading-relaxed opacity-80">
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html $t(section.content)}
						</div>
					</section>
				{/each}
			</div>

			<!-- Footer -->
			<div
				class="mt-20 border-t border-black/10 pt-10 text-center text-sm opacity-40 dark:border-white/10"
			>
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
		0%,
		100% {
			opacity: 0.1;
		}
		50% {
			opacity: 0.15;
		}
	}
</style>
