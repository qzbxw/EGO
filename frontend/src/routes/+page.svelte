<script lang="ts">
	import { goto } from '$app/navigation';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import { getAppLogo } from '$lib/config';
	import { t, locale } from 'svelte-i18n';
	import {
		Brain,
		Shield,
		Database,
		ArrowRight,
		Zap,
		Code,
		Layers,
		Moon,
		Sun,
		Activity,
		Search,
		Check,
		Youtube,
		Calculator,
		BookOpen,
		Cpu,
		Globe,
		MessageSquare,
		Menu,
		X
	} from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { fly, scale, slide } from 'svelte/transition';
	import { cubicOut, elasticOut } from 'svelte/easing';

	let scrolled = $state(false);
	let mobileMenuOpen = $state(false);

	function toggleTheme() {
		setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
	}

	onMount(() => {
		const handleScroll = () => {
			scrolled = window.scrollY > 20;
		};
		window.addEventListener('scroll', handleScroll);
		return () => window.removeEventListener('scroll', handleScroll);
	});

	// Intersection Observer Action for animations
	function viewport(element: HTMLElement) {
		element.classList.add(
			'opacity-0',
			'translate-y-8',
			'transition-all',
			'duration-700',
			'ease-out'
		);

		const observer = new IntersectionObserver(
			(entries) => {
				entries.forEach((entry) => {
					if (entry.isIntersecting) {
						element.classList.remove('opacity-0', 'translate-y-8');
						element.classList.add('opacity-100', 'translate-y-0');
						observer.unobserve(entry.target);
					}
				});
			},
			{ threshold: 0.1 }
		);

		observer.observe(element);
		return {
			destroy() {
				observer.disconnect();
			}
		};
	}

	const tools = [
		{ id: 'egosearch', icon: Search, color: 'blue', tag: 'REAL-TIME', status: 'Global Web Access' },
		{ id: 'egotube', icon: Youtube, color: 'red', tag: 'VISION', status: 'Semantic Video Parsing' },
		{
			id: 'egocodeexec',
			icon: Code,
			color: 'green',
			tag: 'COMPUTE',
			status: 'Isolated Python Sandbox'
		},
		{
			id: 'egomemory',
			icon: Database,
			color: 'purple',
			tag: 'NEURAL',
			status: 'Vector-based Recall'
		},
		{ id: 'egocalc', icon: Calculator, color: 'orange', status: 'Symbolic Mathematics' },
		{
			id: 'egowiki',
			icon: BookOpen,
			color: 'amber',
			tag: 'KNOWLEDGE',
			status: 'Encyclopedic Verification'
		}
	];

	const navItems = [
		{ key: 'features', href: '#features' },
		{ key: 'tools', href: '#tools' },
		{ key: 'how', href: '#how-it-works' }
	];

	const modes = [
		{ id: 'adaptive', icon: Zap, color: 'yellow', detail: 'Low Latency Response' },
		{ id: 'deeper', icon: Layers, color: 'blue', detail: 'Multi-step Reasoning' },
		{ id: 'research', icon: Globe, color: 'indigo', detail: 'Cross-source Synthesis' },
		{ id: 'agent', icon: Cpu, color: 'emerald', detail: 'Self-correcting Cycles' }
	];
	// Word rotation logic
	const alternateWordsRu = ['прозрачными', 'мощными', 'открытыми', 'глубокими', 'гибкими'];
	const alternateWordsEn = ['transparent', 'powerful', 'open', 'deep', 'flexible'];

	const ctaWordsRu = ['начать?', 'создавать?', 'исследовать?', 'ускориться?'];
	const ctaWordsEn = ['start?', 'create?', 'explore?', 'accelerate?'];

	let currentWordIdx = $state(0);
	let currentCtaWordIdx = $state(0);
	let isChanging = $state(false);

	// Reset index when locale changes to avoid out-of-bounds or stuck words
	$effect(() => {
		void $locale; // Dependency on locale
		currentWordIdx = 0;
		currentCtaWordIdx = 0;
	});
	onMount(() => {
		const handleScroll = () => {
			scrolled = window.scrollY > 20;
		};
		window.addEventListener('scroll', handleScroll);

		// Rotate words every 4 seconds
		const interval = setInterval(() => {
			isChanging = true;
			setTimeout(() => {
				const words = $locale === 'ru' ? alternateWordsRu : alternateWordsEn;
				const ctaWords = $locale === 'ru' ? ctaWordsRu : ctaWordsEn;

				currentWordIdx = (currentWordIdx + 1) % words.length;
				currentCtaWordIdx = (currentCtaWordIdx + 1) % ctaWords.length;

				isChanging = false;
			}, 200); // Glitch/change duration
		}, 4000);

		return () => {
			window.removeEventListener('scroll', handleScroll);
			clearInterval(interval);
		};
	});
</script>

<svelte:head>
	<title>{$t('knowego.title')}</title>
	<meta name="description" content={$t('knowego.hero_subtitle').replace(/<\/?[^>]+(>|$)/g, '')} />
</svelte:head>

<div
	class="min-h-screen w-full overflow-x-hidden font-sans transition-colors duration-700 ease-in-out selection:bg-primary/30 selection:text-primary {preferencesStore.theme ===
	'light'
		? 'bg-[#FDFCF8] text-gray-900'
		: 'bg-[#050508] text-white'}"
>
	<!-- Global Background -->
	<div class="pointer-events-none fixed inset-0 z-0 overflow-hidden transition-all duration-700">
		{#if preferencesStore.theme === 'light'}
			<div
				class="absolute inset-0 bg-gradient-to-b from-[#FFFFFF] via-[#FAFAFA] to-[#F5F5F5]"
			></div>
			<div
				class="absolute right-[-10%] top-[-10%] h-[60vw] w-[60vw] rounded-full bg-purple-500/5 blur-[120px]"
			></div>
			<div
				class="absolute bottom-[-10%] left-[-10%] h-[50vw] w-[50vw] rounded-full bg-blue-500/5 blur-[150px]"
			></div>
		{:else}
			<!-- Grid Pattern -->
			<div
				class="absolute inset-0 bg-[linear-gradient(to_right,#ffffff05_1px,transparent_1px),linear-gradient(to_bottom,#ffffff05_1px,transparent_1px)] bg-[size:32px_32px] opacity-20"
			></div>
			<!-- Ambient Glow -->
			<div
				class="animate-pulse-slow absolute left-1/2 top-[-10%] h-[600px] w-[80vw] -translate-x-1/2 rounded-full bg-indigo-900/20 blur-[130px]"
			></div>
			<div
				class="absolute bottom-0 right-0 h-[600px] w-[600px] rounded-full bg-purple-900/10 blur-[150px]"
			></div>
		{/if}
	</div>

	<!-- Navigation -->
	<nav
		class="fixed left-0 right-0 top-0 z-50 border-b transition-all duration-500 {scrolled
			? preferencesStore.theme === 'light'
				? 'border-black/[0.05] bg-white py-3 shadow-sm backdrop-blur-xl'
				: 'border-white/5 bg-black/80 py-3 shadow-2xl backdrop-blur-xl'
			: 'border-transparent bg-transparent py-6'}"
	>
		<div class="mx-auto flex max-w-7xl items-center justify-between px-6">
			<!-- Logo -->
			<button
				class="group flex cursor-pointer items-center gap-3"
				onclick={async () => void goto('/')}
				in:fly={{ y: -20, duration: 600, delay: 0, easing: cubicOut }}
			>
				<div
					class="relative flex h-9 w-9 items-center justify-center transition-transform duration-500 group-hover:rotate-12 group-hover:scale-110"
				>
					<div
						class="absolute inset-0 bg-primary opacity-20 blur-lg transition-opacity group-hover:opacity-40"
					></div>
					{#key preferencesStore.theme}
						<img
							src={getAppLogo(preferencesStore.theme)}
							alt="EGO"
							class="relative h-full w-full object-contain drop-shadow-lg"
						/>
					{/key}
				</div>
				<span class="font-mono text-xl font-bold tracking-tight">EGO</span>
			</button>

			<!-- Menu Items (Desktop) -->
			<div
				class="hidden items-center gap-1 md:flex"
				in:fly={{ y: -20, duration: 600, delay: 100, easing: cubicOut }}
			>
				{#each navItems as item (item.key)}
					<a
						href={item.href}
						class="rounded-lg px-4 py-2 text-sm font-medium opacity-60 transition-all hover:bg-black/5 hover:opacity-100 dark:hover:bg-white/5"
					>
						{$t(`knowego.nav_${item.key}`)}
					</a>
				{/each}
			</div>

			<!-- Actions -->
			<div
				class="flex items-center gap-2 md:gap-3"
				in:fly={{ y: -20, duration: 600, delay: 200, easing: cubicOut }}
			>
				<button
					onclick={toggleTheme}
					class="group rounded-lg p-2.5 text-text-secondary transition-all hover:bg-black/5 hover:text-text-primary active:scale-90 dark:hover:bg-white/5"
					aria-label="Toggle Theme"
				>
					<div class="ease-spring transition-transform duration-500 group-hover:rotate-[360deg]">
						{#if preferencesStore.theme === 'light'}
							<Moon class="h-5 w-5 text-gray-700" />
						{:else}
							<Sun class="h-5 w-5 text-gray-300" />
						{/if}
					</div>
				</button>
				<div class="hidden sm:block">
					<LanguageSwitcher />
				</div>
				<div class="mx-1 hidden h-6 w-px bg-black/[0.08] dark:bg-white/10 sm:block"></div>
				<button
					onclick={async () => void goto('/chat/new')}
					class="group relative overflow-hidden rounded-xl bg-accent px-6 py-2.5 text-sm font-bold text-white shadow-lg shadow-accent/20 transition-all hover:scale-[1.02] active:scale-[0.98] dark:bg-white dark:text-black dark:shadow-white/10"
				>
					<span class="relative z-10 flex items-center gap-2">
						{$t('knowego.cta_start')}
						<ArrowRight
							class="h-4 w-4 transition-transform duration-300 group-hover:translate-x-1"
						/>
					</span>
					<div
						class="absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/20 to-transparent transition-transform duration-700 ease-in-out group-hover:translate-x-full"
					></div>
				</button>

				<!-- Mobile Menu Toggle -->
				<button
					class="rounded-lg p-2 transition-colors hover:bg-black/5 dark:hover:bg-white/5 md:hidden"
					onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
				>
					{#if mobileMenuOpen}
						<X class="h-6 w-6" />
					{:else}
						<Menu class="h-6 w-6" />
					{/if}
				</button>
			</div>
		</div>

		<!-- Mobile Menu Overlay -->
		{#if mobileMenuOpen}
			<div
				class="absolute left-0 right-0 top-full h-screen border-b backdrop-blur-2xl transition-all md:hidden {preferencesStore.theme ===
				'light'
					? 'border-black/5 bg-white/95'
					: 'border-white/5 bg-black/95'}"
				transition:slide={{ duration: 300, axis: 'y' }}
			>
				<div class="flex flex-col gap-2 p-6">
					{#each navItems as item (item.key)}
						<a
							href={item.href}
							class="rounded-xl p-4 text-lg font-medium transition-colors hover:bg-black/5 dark:hover:bg-white/5"
							onclick={() => (mobileMenuOpen = false)}
						>
							{$t(`knowego.nav_${item.key}`)}
						</a>
					{/each}
					<div
						class="mt-4 flex items-center justify-between border-t border-black/5 px-4 pt-6 dark:border-white/5"
					>
						<span class="text-sm font-medium opacity-60">Language</span>
						<LanguageSwitcher />
					</div>
				</div>
			</div>
		{/if}
	</nav>

	<!-- Hero Section -->
	<section class="relative z-10 overflow-visible px-6 pb-20 pt-24 lg:pb-40 lg:pt-32">
		<div class="mx-auto flex max-w-7xl flex-col items-center text-center">
			<!-- Text Content -->
			<div
				class="z-20 flex max-w-4xl flex-col items-center"
				in:fly={{ y: 30, duration: 1000, delay: 300, easing: cubicOut }}
			>
				<!-- Glitch Logo -->
				<div
					class="group relative mb-12"
					in:scale={{ duration: 1000, delay: 200, start: 0.8, easing: elasticOut }}
				>
					<div
						class="absolute inset-0 bg-accent opacity-20 blur-[60px] transition-opacity duration-1000 group-hover:opacity-40"
					></div>
					{#key preferencesStore.theme}
						<img
							src={getAppLogo(preferencesStore.theme)}
							alt="EGO Logo"
							class="logo-malfunction relative h-24 w-24 object-contain transition-transform duration-700 hover:scale-110 md:h-32 md:w-32"
						/>
					{/key}
				</div>

				<h1
					class="logo-malfunction mb-8 flex flex-wrap justify-center gap-x-[0.25em] overflow-visible pb-6 text-5xl font-black leading-[1.2] tracking-[-0.04em] drop-shadow-sm sm:text-6xl md:text-7xl lg:text-8xl
					{preferencesStore.theme === 'light' ? 'text-slate-950' : 'text-white'}"
				>
					{#key $locale}
						{#each $t('knowego.hero_title').split(' ') as word, i (i)}
							{@const isTarget =
								word.toLowerCase().includes('прозрачными') ||
								word.toLowerCase().includes('transparent')}
							{@const isAI = word.includes('AI')}
							{@const isReasoning =
								word.toLowerCase().includes('рассуждениями') ||
								word.toLowerCase().includes('reasoning')}

							<span
								class="inline-block {isTarget && isChanging
									? 'animate-glitch-fast text-accent'
									: ''}"
								in:fly={{ y: 30, duration: 1000, delay: 400 + i * 120, easing: cubicOut }}
							>
								{#if isAI}
									<span
										class="relative mr-1 inline-flex items-center rounded-2xl border border-accent/20 bg-accent/[0.03] px-3 py-1 font-mono font-black text-accent shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.1)] dark:bg-accent/10"
										>AI</span
									>{word.replace('AI', '')}
								{:else if isTarget}
									<!-- Stable container for dynamic words -->
									<span class="inline-grid grid-cols-1 grid-rows-1 text-center">
										<span class="pointer-events-none invisible col-start-1 row-start-1 px-1"
											>{$locale === 'ru' ? 'прозрачными' : 'transparent'}</span
										>
										<span class="col-start-1 row-start-1">
											{($locale === 'ru' ? alternateWordsRu : alternateWordsEn)[currentWordIdx]}
										</span>
									</span>
								{:else if isReasoning}
									<span class="relative inline-block">
										{word}
										<span class="absolute -bottom-2 left-0 h-[3px] w-full rounded-full bg-accent/20"
										></span>
									</span>
								{:else}
									{word}
								{/if}
							</span>
						{/each}
					{/key}
				</h1>

				<p
					class="mb-10 max-w-2xl text-lg font-light leading-relaxed opacity-60 md:text-2xl"
					in:fly={{ y: 20, duration: 800, delay: 600 }}
				>
					{@html $t('knowego.hero_subtitle')}
				</p>

				<div
					class="flex flex-wrap justify-center gap-4"
					in:fly={{ y: 20, duration: 800, delay: 700 }}
				>
					<button
						onclick={async () => void goto('/chat/new')}
						class="group flex items-center gap-2 rounded-xl bg-black px-10 py-5 text-lg font-bold text-white transition-all hover:-translate-y-1 hover:shadow-[0_8px_30px_rgba(0,0,0,0.12)] active:translate-y-0 dark:bg-white dark:text-black"
					>
						{$t('knowego.cta_start')}
						<ArrowRight class="h-5 w-5 transition-transform group-hover:translate-x-1" />
					</button>
					<button
						onclick={() =>
							document.getElementById('how-it-works')?.scrollIntoView({ behavior: 'smooth' })}
						class="flex items-center gap-2 rounded-xl border border-black/10 px-10 py-5 text-lg font-bold transition-all hover:bg-black/5 dark:border-white/10 dark:hover:bg-white/5"
					>
						{$t('knowego.hero_meet_ego')}
					</button>
				</div>
			</div>
		</div>
	</section>

	<!-- Capabilities / Tools -->
	<section
		id="tools"
		class="relative z-10 border-t border-black/[0.03] px-6 py-24 dark:border-white/5"
	>
		<div class="mx-auto max-w-7xl">
			<div class="mb-16 text-center md:mb-24" use:viewport>
				<h2 class="mb-6 text-4xl font-black tracking-tight md:text-6xl">
					{$t('knowego.features_capabilities_title')}
				</h2>
				<p class="mx-auto max-w-2xl text-lg font-light leading-relaxed opacity-60 md:text-xl">
					{$t('knowego.features_capabilities_subtitle')}
				</p>
			</div>

			<!-- Module Grid -->
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
				{#each tools as tool (tool.id)}
					<div
						class="group relative flex flex-col rounded-[2.5rem] border p-8 transition-all duration-500 hover:-translate-y-1
						{preferencesStore.theme === 'light'
							? 'border-black/[0.03] bg-[#FBFBFD] hover:bg-white hover:shadow-[0_30px_60px_rgba(0,0,0,0.06)]'
							: 'border-white/5 bg-[#0A0A0A] hover:bg-[#0F0F0F] hover:shadow-[0_30px_60px_rgba(0,0,0,0.4)]'}"
						use:viewport
					>
						<!-- Header -->
						<div class="mb-12 flex items-start justify-between">
							<div
								class="flex h-16 w-16 items-center justify-center rounded-2xl transition-all duration-500 group-hover:scale-110
								{preferencesStore.theme === 'light' ? `bg-${tool.color}-500/[0.08]` : `bg-${tool.color}-500/10`}"
							>
								<tool.icon class="h-8 w-8 text-{tool.color}-500" />
							</div>
							<div
								class="rounded-full border px-3 py-1
								{preferencesStore.theme === 'light'
									? 'border-black/[0.05] bg-black/[0.03]'
									: 'border-white/[0.05] bg-white/[0.03]'}"
							>
								<span class="text-[10px] font-bold uppercase tracking-widest opacity-40"
									>{tool.tag}</span
								>
							</div>
						</div>

						<!-- Content -->
						<div class="mt-auto">
							<h3 class="mb-4 text-2xl font-bold tracking-tight">
								{$t(`knowego.tool_${tool.id}_title`)}
							</h3>
							<p class="text-sm font-medium leading-relaxed opacity-60 md:text-base">
								{$t(`knowego.tool_${tool.id}_desc`)}
							</p>

							<!-- Hover-only info -->
							<div
								class="mt-6 flex translate-y-2 items-center gap-2 font-mono text-[10px] font-bold tracking-widest text-accent opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100"
							>
								<div class="h-1 w-1 rounded-full bg-accent"></div>
								{tool.status}
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Agentic Loop Section -->
	<section
		id="agentic-loop"
		class="relative z-10 overflow-hidden border-t border-black/[0.03] px-6 py-24 dark:border-white/5"
	>
		<div class="mx-auto max-w-7xl">
			<div class="mb-20 text-center" use:viewport>
				<h2 class="mb-6 text-4xl font-black tracking-tight md:text-6xl">
					{$t('knowego.agentic_loop_title')}
				</h2>
				<p class="mx-auto max-w-2xl text-lg font-light opacity-60 md:text-xl">
					{$t('knowego.agentic_loop_subtitle')}
				</p>
			</div>

			<div class="relative grid grid-cols-1 gap-8 md:grid-cols-4">
				<!-- Connecting Line (Desktop) -->
				<div
					class="absolute left-[12%] right-[12%] top-12 hidden h-0.5 bg-gradient-to-r from-transparent via-accent/30 to-transparent md:block"
				></div>

				<!-- Step 1: Gather Context -->
				<div class="group relative flex flex-col items-center text-center" use:viewport>
					<div
						class="relative z-10 mb-8 flex h-24 w-24 items-center justify-center rounded-full border-4 border-white/5 bg-black/5 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)] dark:bg-white/5"
					>
						<Search
							class="h-10 w-10 opacity-60 transition-all group-hover:text-accent group-hover:opacity-100"
						/>
						<div
							class="absolute -right-3 -top-3 flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-bold text-white shadow-lg dark:text-black"
						>
							1
						</div>
					</div>
					<h3 class="mb-3 text-xl font-bold">{$t('knowego.agentic_step1_title')}</h3>
					<p class="max-w-[200px] text-sm opacity-60">{$t('knowego.agentic_step1_desc')}</p>
				</div>

				<!-- Step 2: Take Action -->
				<div class="group relative flex flex-col items-center text-center" use:viewport>
					<div
						class="relative z-10 mb-8 flex h-24 w-24 items-center justify-center rounded-full border-4 border-white/5 bg-black/5 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)] dark:bg-white/5"
					>
						<Zap
							class="h-10 w-10 opacity-60 transition-all group-hover:text-accent group-hover:opacity-100"
						/>
						<div
							class="absolute -right-3 -top-3 flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-bold text-white shadow-lg dark:text-black"
						>
							2
						</div>
					</div>
					<h3 class="mb-3 text-xl font-bold">{$t('knowego.agentic_step2_title')}</h3>
					<p class="max-w-[200px] text-sm opacity-60">{$t('knowego.agentic_step2_desc')}</p>
				</div>

				<!-- Step 3: Verify Results -->
				<div class="group relative flex flex-col items-center text-center" use:viewport>
					<div
						class="relative z-10 mb-8 flex h-24 w-24 items-center justify-center rounded-full border-4 border-white/5 bg-black/5 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)] dark:bg-white/5"
					>
						<Shield
							class="h-10 w-10 opacity-60 transition-all group-hover:text-accent group-hover:opacity-100"
						/>
						<div
							class="absolute -right-3 -top-3 flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-bold text-white shadow-lg dark:text-black"
						>
							3
						</div>
					</div>
					<h3 class="mb-3 text-xl font-bold">{$t('knowego.agentic_step3_title')}</h3>
					<p class="max-w-[200px] text-sm opacity-60">{$t('knowego.agentic_step3_desc')}</p>
				</div>

				<!-- Step 4: Reflect & Refine -->
				<div class="group relative flex flex-col items-center text-center" use:viewport>
					<div
						class="relative z-10 mb-8 flex h-24 w-24 items-center justify-center rounded-full border-4 border-white/5 bg-black/5 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)] dark:bg-white/5"
					>
						<Brain
							class="h-10 w-10 opacity-60 transition-all group-hover:text-accent group-hover:opacity-100"
						/>
						<div
							class="absolute -right-3 -top-3 flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-bold text-white shadow-lg dark:text-black"
						>
							4
						</div>
					</div>
					<h3 class="mb-3 text-xl font-bold">{$t('knowego.agentic_step4_title')}</h3>
					<p class="max-w-[200px] text-sm opacity-60">{$t('knowego.agentic_step4_desc')}</p>
				</div>
			</div>

			<!-- Infinite Loop Animation Overlay -->
			<div class="mt-16 flex justify-center opacity-30">
				<div class="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.3em]">
					<div class="h-2 w-2 animate-pulse rounded-full bg-accent"></div>
					Running Agentic Process
				</div>
			</div>
		</div>
	</section>

	<!-- Features (Bento Grid) -->
	<section
		id="features"
		class="relative z-10 border-t border-black/5 px-6 py-24 dark:border-white/5"
	>
		<div class="mx-auto max-w-7xl">
			<div class="mb-20" use:viewport>
				<h2 class="mb-6 text-3xl font-bold tracking-tight md:text-5xl">
					{$t('knowego.features_title')}
				</h2>
				<p class="max-w-xl text-xl opacity-60">{$t('knowego.features_subtitle')}</p>
			</div>

			<div class="grid auto-rows-auto grid-cols-1 gap-6 md:auto-rows-fr md:grid-cols-3">
				<!-- Feature 1: Thinking (Large) -->
				<div
					class="group relative overflow-hidden rounded-[2.5rem] border p-8 transition-all duration-500 md:col-span-2 md:row-span-2 md:p-12
					{preferencesStore.theme === 'light'
						? 'border-black/[0.03] bg-white shadow-lg shadow-black/[0.02]'
						: 'border-white/5 bg-gradient-to-br from-purple-500/5 via-transparent to-transparent'}"
					use:viewport
				>
					<div class="relative z-10 flex h-full flex-col">
						<div
							class="mb-8 flex h-12 w-12 items-center justify-center rounded-2xl bg-purple-500/10"
						>
							<Brain class="h-6 w-6 text-purple-500" />
						</div>
						<h3 class="mb-4 text-2xl font-bold tracking-tight md:text-4xl">
							{$t('knowego.feature_thinking_title')}
						</h3>
						<p class="mb-10 max-w-lg text-base leading-relaxed opacity-60 md:text-lg">
							{@html $t('knowego.feature_thinking_desc')}
						</p>

						<!-- Thinking Stack Visualization -->
						<div class="group/stack relative mt-auto h-32 w-full max-w-2xl md:h-48">
							<!-- Bottom Card (Synthesis) -->
							<div
								class="absolute bottom-0 left-4 right-4 translate-y-4 rounded-2xl border p-4 opacity-40 transition-all duration-700 group-hover/stack:translate-y-12 group-hover/stack:opacity-60
								{preferencesStore.theme === 'light'
									? 'border-black/[0.05] bg-white'
									: 'border-white/5 bg-[#0F0F0F]'}"
							>
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-3">
										<div
											class="flex h-7 w-7 items-center justify-center rounded-lg bg-green-500/10 text-green-500"
										>
											<Check class="h-4 w-4" />
										</div>
										<span class="text-xs font-bold opacity-60">Synthesis</span>
									</div>
								</div>
							</div>

							<!-- Middle Card (Execution) -->
							<div
								class="absolute bottom-4 left-2 right-2 translate-y-2 rounded-2xl border p-4 opacity-80 transition-all duration-500 group-hover/stack:translate-y-6 group-hover/stack:opacity-100
								{preferencesStore.theme === 'light'
									? 'border-black/[0.05] bg-white shadow-lg'
									: 'border-white/10 bg-[#141414] shadow-2xl'}"
							>
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-3">
										<div
											class="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500"
										>
											<Search class="h-4 w-4" />
										</div>
										<span class="text-xs font-bold">Tool Execution</span>
									</div>
									<div
										class="rounded bg-blue-500/10 px-2 py-0.5 text-[9px] font-black uppercase tracking-tighter text-blue-500"
									>
										Running
									</div>
								</div>
							</div>

							<!-- Top Card (Analysis) -->
							<div
								class="absolute bottom-8 left-0 right-0 rounded-2xl border p-5 transition-all duration-500 group-hover/stack:-translate-y-2
								{preferencesStore.theme === 'light'
									? 'border-black/[0.08] bg-white shadow-xl'
									: 'border-white/20 bg-[#1A1A1A] shadow-2xl'}"
							>
								<div class="mb-4 flex items-center justify-between">
									<div class="flex items-center gap-3">
										<div
											class="flex h-8 w-8 items-center justify-center rounded-lg bg-purple-500/10 text-purple-500"
										>
											<Activity class="h-4 w-4" />
										</div>
										<span class="text-sm font-bold tracking-tight">Strategic Planning</span>
									</div>
									<span class="text-[9px] font-bold uppercase tracking-[0.2em] opacity-30"
										>Step 01</span
									>
								</div>
								<div class="space-y-2">
									<div
										class="h-1.5 w-full overflow-hidden rounded-full bg-black/[0.03] dark:bg-white/5"
									>
										<div class="animate-pulse-gentle h-full w-2/3 bg-accent"></div>
									</div>
									<div
										class="flex justify-between text-[10px] font-bold uppercase tracking-widest opacity-40"
									>
										<span>Analyzing Context</span>
										<span>65%</span>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Feature 2: Memory -->
				<div
					class="group rounded-[2.5rem] border p-8 transition-all duration-300 hover:-translate-y-1 md:p-10
					{preferencesStore.theme === 'light'
						? 'border-black/[0.03] bg-blue-500/[0.03] hover:bg-white hover:shadow-xl hover:shadow-blue-500/[0.05]'
						: 'border-white/5 bg-blue-500/5 hover:bg-blue-500/[0.08]'}"
					use:viewport
				>
					<Database
						class="mb-6 h-10 w-10 text-blue-500 transition-transform duration-500 group-hover:rotate-12"
					/>
					<h3 class="mb-4 text-xl font-bold tracking-tight md:text-2xl">
						{$t('knowego.feature_memory_title')}
					</h3>
					<p class="mb-6 text-sm font-medium leading-relaxed opacity-60 md:text-base">
						{@html $t('knowego.feature_memory_desc')}
					</p>
					<div
						class="translate-y-2 font-mono text-[10px] font-bold tracking-widest text-blue-500 opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100"
					>
						PERMANENT VECTOR CONTEXT
					</div>
				</div>

				<!-- Feature 3: Tools -->
				<div
					class="group rounded-[2.5rem] border p-8 transition-all duration-300 hover:-translate-y-1 md:p-10
					{preferencesStore.theme === 'light'
						? 'border-black/[0.03] bg-emerald-500/[0.03] hover:bg-white hover:shadow-xl hover:shadow-emerald-500/[0.05]'
						: 'border-white/5 bg-emerald-500/5 hover:bg-emerald-500/[0.08]'}"
					use:viewport
				>
					<Cpu
						class="mb-6 h-10 w-10 text-emerald-500 transition-transform duration-500 group-hover:scale-110"
					/>
					<h3 class="mb-4 text-xl font-bold tracking-tight md:text-2xl">
						{$t('knowego.feature_tools_title')}
					</h3>
					<p class="mb-6 text-sm font-medium leading-relaxed opacity-60 md:text-base">
						{@html $t('knowego.feature_tools_desc')}
					</p>
					<div
						class="translate-y-2 font-mono text-[10px] font-bold tracking-widest text-emerald-500 opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100"
					>
						FULL API INTEGRATION
					</div>
				</div>

				<!-- Feature 4: Models (Wide) -->
				<div
					class="flex flex-col items-center gap-10 rounded-[2.5rem] border p-8 backdrop-blur-sm md:col-span-3 md:flex-row md:p-12
					{preferencesStore.theme === 'light'
						? 'border-black/[0.03] bg-white'
						: 'border-white/5 bg-white/[0.02]'}"
					use:viewport
				>
					<div class="flex-1 text-center md:text-left">
						<div
							class="mx-auto mb-6 flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-500/10 md:mx-0"
						>
							<Globe class="h-6 w-6 text-amber-500" />
						</div>
						<h3 class="mb-4 text-2xl font-bold tracking-tight md:text-3xl">
							{$t('knowego.feature_models_title')}
						</h3>
						<p class="text-base font-medium leading-relaxed opacity-60 md:text-lg">
							{$t('knowego.feature_models_desc')}
						</p>
					</div>
					<div class="flex flex-1 flex-wrap justify-center gap-3 md:justify-end">
						{#each ['EGO', 'OpenAI', 'Claude', 'Gemini', 'Grok'] as model (model)}
							<div
								class="cursor-default rounded-xl border px-5 py-2.5 text-sm font-bold opacity-60 shadow-sm transition-all hover:scale-105 hover:opacity-100
								{preferencesStore.theme === 'light' ? 'border-black/[0.05] bg-white' : 'border-white/5 bg-white/5'}"
							>
								{model}
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- How it works (Pipeline) -->
	<section
		id="how-it-works"
		class="relative z-10 overflow-hidden border-t border-black/[0.03] px-6 py-24 dark:border-white/5"
	>
		<div class="mx-auto max-w-7xl">
			<div class="mb-20 text-center md:mb-32" use:viewport>
				<h2 class="mb-6 text-4xl font-black tracking-tight md:text-6xl">
					{$t('knowego.how_title')}
				</h2>
				<p class="mx-auto max-w-2xl text-lg font-light opacity-60 md:text-xl">
					{$t('knowego.how_subtitle')}
				</p>
			</div>

			<div class="grid grid-cols-1 gap-12 md:grid-cols-3 md:gap-8">
				{#each [{ step: 1, color: 'purple', icon: MessageSquare }, { step: 2, color: 'blue', icon: Brain }, { step: 3, color: 'green', icon: Database }] as { step, color, icon }, unused_i (step)}
					{@const Icon = icon}
					<div
						class="group relative flex flex-col"
						in:fly={{ y: 30, duration: 800, delay: 200 * unused_i, easing: cubicOut }}
					>
						<!-- Large Background Number -->
						<div
							class="pointer-events-none absolute -left-4 -top-12 text-8xl font-black opacity-[0.03] transition-all duration-700 group-hover:-translate-y-2 group-hover:opacity-10 dark:opacity-[0.05] md:text-9xl"
						>
							0{step}
						</div>

						<div class="relative z-10">
							<!-- Icon Sphere -->
							<div
								class="mb-8 flex h-16 w-16 items-center justify-center rounded-3xl transition-all duration-500 group-hover:rotate-3 group-hover:scale-110
								{preferencesStore.theme === 'light'
									? 'border border-black/[0.03] bg-white shadow-xl shadow-black/[0.03]'
									: 'border border-white/5 bg-white/5 shadow-2xl'}"
							>
								<Icon class="h-7 w-7 text-{color}-500" />
							</div>

							<h3 class="mb-4 text-2xl font-bold tracking-tight">
								{$t(`knowego.how_step${step}_title`)}
							</h3>
							<p class="mb-8 text-sm font-medium leading-relaxed opacity-60 md:text-base">
								{$t(`knowego.how_step${step}_desc`)}
							</p>

							<!-- Visual element for each step -->
							<div
								class="mt-auto h-1 w-full overflow-hidden rounded-full bg-black/[0.03] dark:bg-white/5"
							>
								<div
									class="h-full w-0 transition-all duration-1000 ease-in-out group-hover:w-full bg-{color}-500/40"
								></div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Modes Section -->
	<section
		id="modes"
		class="relative z-10 overflow-hidden border-t border-black/[0.03] px-6 py-24 dark:border-white/5"
	>
		<div class="mx-auto max-w-7xl">
			<div class="mb-16 text-center md:mb-20" use:viewport>
				<h2 class="mb-6 text-4xl font-black tracking-tight md:text-6xl">
					{$t('knowego.modes_title')}
				</h2>
				<p class="mx-auto max-w-2xl text-lg font-light opacity-60 md:text-xl">
					{$t('knowego.modes_subtitle')}
				</p>
			</div>

			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
				{#each modes as mode, i (mode.id)}
					{@const ModeIcon = mode.icon}
					<div
						class="group relative flex flex-col rounded-[2.5rem] border p-8 transition-all duration-500 hover:-translate-y-1
						{preferencesStore.theme === 'light'
							? 'border-black/[0.03] bg-[#FBFBFD] hover:bg-white hover:shadow-[0_30px_60px_rgba(0,0,0,0.05)]'
							: 'border-white/5 bg-[#0A0A0A] hover:bg-[#0F0F0F] hover:shadow-[0_30px_60px_rgba(0,0,0,0.4)]'}"
						in:fly={{ y: 20, duration: 800, delay: 100 * i, easing: cubicOut }}
					>
						<div
							class="mb-8 flex h-12 w-12 items-center justify-center rounded-xl transition-transform duration-500 group-hover:scale-110
							{preferencesStore.theme === 'light' ? `bg-${mode.color}-500/[0.08]` : `bg-${mode.color}-500/10`}"
						>
							<ModeIcon class="h-6 w-6 text-{mode.color}-500" />
						</div>

						<h3 class="mb-3 text-xl font-bold tracking-tight">
							{$t(`knowego.mode_${mode.id}_title`)}
						</h3>
						<p class="mb-6 text-sm font-medium leading-relaxed opacity-60">
							{$t(`knowego.mode_${mode.id}_desc`)}
						</p>

						<div
							class="mt-auto flex translate-y-2 items-center gap-2 font-mono text-[10px] font-bold tracking-widest text-accent opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100"
						>
							<div class="h-1 w-1 rounded-full bg-accent"></div>
							{mode.detail}
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- CTA Section -->
	<section class="relative z-10 px-6 py-32">
		<div
			class="relative mx-auto max-w-6xl overflow-hidden rounded-[3rem] p-12 text-center transition-all duration-700 md:p-32
			{preferencesStore.theme === 'light'
				? 'border border-black/[0.03] bg-white shadow-[0_40px_100px_rgba(0,0,0,0.04)]'
				: 'border border-white/5 bg-[#050508] shadow-2xl'}"
			use:viewport
		>
			<!-- Animated Ambient Glow for CTA -->
			<div class="pointer-events-none absolute inset-0">
				<div
					class="absolute left-1/2 top-1/2 h-[80%] w-[80%] -translate-x-1/2 -translate-y-1/2 rounded-full opacity-20 blur-[120px]
					{preferencesStore.theme === 'light' ? 'bg-accent/20' : 'bg-accent/30'}"
				></div>
			</div>

			<div class="relative z-10 flex flex-col items-center">
				<h2
					class="logo-malfunction mb-8 flex flex-col items-center text-4xl font-black leading-tight tracking-[-0.04em] md:text-7xl
					{preferencesStore.theme === 'light' ? 'text-slate-950' : 'text-white'}"
				>
					<span>{$t('knowego.cta_block_title')}</span>

					{#key $locale}
						<!-- Stable container for dynamic CTA word -->
						<span class="mt-2 inline-grid min-w-[3ch] grid-cols-1 grid-rows-1 text-center">
							<span class="pointer-events-none invisible col-start-1 row-start-1"
								>{$locale === 'ru' ? 'исследовать?' : 'accelerate?'}</span
							>
							<span
								class="col-start-1 row-start-1 {isChanging
									? 'animate-glitch-fast text-accent'
									: ''}"
							>
								{($locale === 'ru' ? ctaWordsRu : ctaWordsEn)[currentCtaWordIdx]}
							</span>
						</span>
					{/key}
				</h2>

				<p class="mx-auto mb-12 max-w-2xl text-lg font-light opacity-60 md:text-2xl">
					{$t('knowego.cta_block_sub')}
				</p>

				<button
					onclick={async () => void goto('/chat/new')}
					class="group relative overflow-hidden rounded-2xl px-12 py-5 text-xl font-black shadow-2xl transition-all hover:scale-105 active:scale-95
					{preferencesStore.theme === 'light'
						? 'bg-slate-950 text-white shadow-black/10'
						: 'bg-white text-black shadow-white/5'}"
				>
					<span class="relative z-10 flex items-center gap-3">
						{$t('knowego.cta_start')}
						<ArrowRight
							class="h-6 w-6 transition-transform duration-300 group-hover:translate-x-1"
						/>
					</span>
					<!-- Shimmer Effect on Button -->
					<div
						class="absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/10 to-transparent transition-transform duration-1000 ease-in-out group-hover:translate-x-full dark:via-black/5"
					></div>
				</button>
			</div>
		</div>
	</section>

	<!-- Footer -->
	<footer
		class="relative z-10 border-t px-6 py-20 backdrop-blur-xl
		{preferencesStore.theme === 'light'
			? 'border-black/[0.05] bg-white text-slate-900'
			: 'border-white/5 bg-black/20 text-white'}"
	>
		<div class="mx-auto max-w-7xl">
			<div class="mb-20 flex flex-col items-start justify-between gap-12 md:flex-row">
				<div class="w-full md:max-w-sm">
					<div class="mb-6 flex items-center gap-2.5">
						<div
							class="flex h-8 w-8 items-center justify-center rounded-lg p-1.5 shadow-xl transition-colors duration-300
							{preferencesStore.theme === 'light' ? 'bg-white shadow-black/5' : 'bg-black'}"
						>
							{#key preferencesStore.theme}
								<!-- In light theme (white bg), we need dark logo. In dark theme (black bg), we need light logo. -->
								<img
									src={getAppLogo(preferencesStore.theme === 'light' ? 'light' : 'dark')}
									alt="EGO"
									class="h-full w-full object-contain"
								/>
							{/key}
						</div>
						<span class="text-xl font-bold tracking-tight">EGO</span>
					</div>
					<p
						class="mb-8 text-sm font-medium leading-relaxed {preferencesStore.theme === 'light'
							? 'opacity-60'
							: 'opacity-70'}"
					>
						{$t('knowego.footer_disclaimer')}
					</p>
				</div>

				<div class="grid w-full grid-cols-2 gap-10 sm:grid-cols-3 md:w-auto md:gap-24">
					<div class="space-y-4">
						<h4 class="text-lg font-bold">{$t('knowego.footer_product')}</h4>
						<ul
							class="space-y-3 text-sm font-medium {preferencesStore.theme === 'light'
								? 'opacity-60'
								: 'opacity-70'}"
						>
							<li>
								<a href="#features" class="transition-colors hover:text-accent"
									>{$t('knowego.footer_features')}</a
								>
							</li>
							<li>
								<a href="#tools" class="transition-colors hover:text-accent"
									>{$t('knowego.footer_tools')}</a
								>
							</li>
							<li>
								<a href="#modes" class="transition-colors hover:text-accent"
									>{$t('knowego.footer_modes')}</a
								>
							</li>
						</ul>
					</div>
					<div class="space-y-4">
						<h4 class="text-lg font-bold">{$t('knowego.footer_legal')}</h4>
						<ul
							class="space-y-3 text-sm font-medium {preferencesStore.theme === 'light'
								? 'opacity-60'
								: 'opacity-70'}"
						>
							<li>
								<a href="/terms" class="transition-colors hover:text-accent"
									>{$t('knowego.footer_terms_of_service')}</a
								>
							</li>
							<li>
								<a href="/privacy" class="transition-colors hover:text-accent"
									>{$t('knowego.footer_privacy_policy')}</a
								>
							</li>
						</ul>
					</div>
					<div class="space-y-4">
						<h4 class="text-lg font-bold">{$t('knowego.footer_contact')}</h4>
						<ul
							class="space-y-3 text-sm font-medium {preferencesStore.theme === 'light'
								? 'opacity-60'
								: 'opacity-70'}"
						>
							<li>
								<a href="mailto:support@askego.xyz" class="transition-colors hover:text-accent"
									>support@askego.xyz</a
								>
							</li>
						</ul>
					</div>
				</div>
			</div>

			<div
				class="flex flex-col justify-between gap-6 border-t pt-10 text-xs font-medium sm:flex-row
				{preferencesStore.theme === 'light'
					? 'border-black/[0.05] opacity-50'
					: 'border-white/5 opacity-60'}"
			>
				<span>{$t('knowego.footer_copyright_text')}</span>
				<div class="flex gap-6">
					<span>{$t('knowego.footer_crafted_by')}</span>
				</div>
			</div>
		</div>
	</footer>
</div>

<style>
	:global(html) {
		scroll-behavior: smooth;
	}

	@keyframes pulse-slow {
		0%,
		100% {
			opacity: 0.1;
			transform: scale(1);
		}
		50% {
			opacity: 0.15;
			transform: scale(1.1);
		}
	}

	.animate-pulse-slow {
		animation: pulse-slow 8s ease-in-out infinite;
	}

	.perspective-1000 {
		perspective: 1000px;
	}

	.preserve-3d {
		transform-style: preserve-3d;
	}

	.rotate-x-12 {
		transform: rotateX(12deg);
	}

	.rotate-z-12 {
		transform: rotateZ(12deg);
	}

	@keyframes gradient-flow {
		0% {
			background-position: 0% 50%;
		}
		50% {
			background-position: 100% 50%;
		}
		100% {
			background-position: 0% 50%;
		}
	}

	/* Fix for dynamic colors in tailwind - if they are not picked up, we use style or safelist */
	.bg-blue-500\/10 {
		background-color: rgba(59, 130, 246, 0.1);
	}
	.bg-red-500\/10 {
		background-color: rgba(239, 68, 68, 0.1);
	}
	.bg-green-500\/10 {
		background-color: rgba(34, 197, 94, 0.1);
	}
	.bg-purple-500\/10 {
		background-color: rgba(168, 85, 247, 0.1);
	}
	.bg-orange-500\/10 {
		background-color: rgba(249, 115, 22, 0.1);
	}
	.bg-amber-500\/10 {
		background-color: rgba(245, 158, 11, 0.1);
	}
	.bg-yellow-500\/10 {
		background-color: rgba(234, 179, 8, 0.1);
	}
	.bg-indigo-500\/10 {
		background-color: rgba(99, 102, 241, 0.1);
	}
	.bg-emerald-500\/10 {
		background-color: rgba(16, 185, 129, 0.1);
	}

	.text-blue-500 {
		color: #3b82f6;
	}
	.text-red-500 {
		color: #ef4444;
	}
	.text-green-500 {
		color: #22c55e;
	}
	.text-purple-500 {
		color: #a855f7;
	}
	.text-orange-500 {
		color: #f97316;
	}
	.text-amber-500 {
		color: #f59e0b;
	}
	.text-yellow-500 {
		color: #eab308;
	}
	.text-indigo-500 {
		color: #6366f1;
	}
	.text-emerald-500 {
		color: #10b981;
	}

	.border-blue-500\/10 {
		border-color: rgba(59, 130, 246, 0.1);
	}
	.border-red-500\/10 {
		border-color: rgba(239, 68, 68, 0.1);
	}
	.border-green-500\/10 {
		border-color: rgba(34, 197, 94, 0.1);
	}
	.border-purple-500\/10 {
		border-color: rgba(168, 85, 247, 0.1);
	}
	.border-orange-500\/10 {
		border-color: rgba(249, 115, 22, 0.1);
	}
	.border-amber-500\/10 {
		border-color: rgba(245, 158, 11, 0.1);
	}

	.logo-malfunction {
		animation: glitch-malfunction 6s infinite step-end;
		filter: contrast(1.1);
	}

	.glitch-chip {
		display: inline-flex;
		align-items: baseline;
		font-family: var(--font-mono);
		font-weight: 900;
		color: var(--color-accent);
		background-color: rgba(var(--color-accent-rgb), 0.08);
		padding: 0.1em 0.5em;
		border-radius: 0.4em;
		border: 1px solid rgba(var(--color-accent-rgb), 0.3);
		box-shadow: 0 0 15px rgba(var(--color-accent-rgb), 0.15);
		animation: glitch-malfunction 4s infinite step-end;
		line-height: 1;
		vertical-align: middle;
		margin: 0 0.2em;
		font-size: 0.9em; /* Relative to parent font size */
		transform: translateY(-0.1em); /* Visual optical alignment */
	}

	@keyframes glitch-malfunction {
		0%,
		86%,
		100% {
			transform: translate(0);
			filter: none;
			opacity: 1;
			text-shadow: none;
		}
		87% {
			transform: translate(-2px, 1px);
			filter: hue-rotate(90deg) saturate(1.5);
			clip-path: inset(10% 0 40% 0);
			color: #ff003c;
			opacity: 0.9;
			text-shadow: 2px 0 #00fff2;
		}
		88% {
			transform: translate(2px, -1px);
			filter: hue-rotate(-90deg) contrast(2);
			clip-path: inset(40% 0 10% 0);
			color: #ff003c;
			opacity: 0.8;
			text-shadow: -2px 0 #00fff2;
		}
		89% {
			transform: translate(0);
			filter: none;
			clip-path: none;
			opacity: 1;
			text-shadow: none;
		}
		90% {
			transform: translate(-3px, 2px);
			filter: hue-rotate(45deg);
			clip-path: inset(5% 0 60% 0);
			color: #ff003c;
			opacity: 0.7;
		}
		91% {
			transform: translate(0);
			filter: none;
			clip-path: none;
			opacity: 1;
		}
	}

	.animate-glitch-fast {
		animation: glitch-fast 0.2s linear infinite;
	}

	@keyframes glitch-fast {
		0% {
			transform: translate(0);
			filter: hue-rotate(0deg);
		}
		25% {
			transform: translate(2px, -2px);
			filter: hue-rotate(90deg);
		}
		50% {
			transform: translate(-2px, 2px);
			filter: contrast(2);
		}
		75% {
			transform: translate(1px, -1px);
			filter: invert(0.1);
		}
		100% {
			transform: translate(0);
			filter: none;
		}
	}
</style>
