<script lang="ts">
	import { goto } from '$app/navigation';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import { getAppLogo } from '$lib/config';
	import { t, locale } from 'svelte-i18n';
	import {
		Brain,
		Terminal,
		Shield,
		Database,
		ArrowRight,
		CheckCircle,
		Zap,
		Code,
		Layers,
		Moon,
		Sun,
		Sparkles,
		Activity,
		Lock,
		Search,
		Check,
		Youtube,
		Calculator,
		BookOpen,
		Cpu,
		Globe,
		MessageSquare,
		ChevronDown,
		Menu,
		X
	} from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { fade, fly, slide } from 'svelte/transition';

	let scrolled = $state(false);
	let activeFaq = $state<number | null>(null);
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
		element.classList.add('opacity-0', 'translate-y-8', 'transition-all', 'duration-700', 'ease-out');

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
		{ id: 'egocodeexec', icon: Code, color: 'green', tag: 'COMPUTE', status: 'Isolated Python Sandbox' },
		{ id: 'egomemory', icon: Database, color: 'purple', tag: 'NEURAL', status: 'Vector-based Recall' },
		{ id: 'egocalc', icon: Calculator, color: 'orange', status: 'Symbolic Mathematics' },
		{ id: 'egowiki', icon: BookOpen, color: 'amber', tag: 'KNOWLEDGE', status: 'Encyclopedic Verification' }
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
		const _ = $locale; // Dependency on locale
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
	<meta name="description" content={$t('knowego.hero_subtitle').replace(/<\/?[^>]+(>|$)/g, "")} />
</svelte:head>

<div
	class="min-h-screen w-full overflow-x-hidden font-sans selection:bg-primary/30 selection:text-primary transition-colors duration-700 ease-in-out {preferencesStore.theme ===
	'light'
		? 'bg-[#FDFCF8] text-gray-900'
		: 'bg-[#050508] text-white'}"
>
	<!-- Global Background -->
	<div class="fixed inset-0 z-0 pointer-events-none transition-all duration-700 overflow-hidden">
		{#if preferencesStore.theme === 'light'}
			<div class="absolute inset-0 bg-gradient-to-b from-[#FFFFFF] via-[#FAFAFA] to-[#F5F5F5]"></div>
			<div class="absolute top-[-10%] right-[-10%] w-[60vw] h-[60vw] bg-purple-500/5 blur-[120px] rounded-full"></div>
			<div class="absolute bottom-[-10%] left-[-10%] w-[50vw] h-[50vw] bg-blue-500/5 blur-[150px] rounded-full"></div>
		{:else}
			<!-- Grid Pattern -->
			<div class="absolute inset-0 bg-[linear-gradient(to_right,#ffffff05_1px,transparent_1px),linear-gradient(to_bottom,#ffffff05_1px,transparent_1px)] bg-[size:32px_32px] opacity-20"></div>
			<!-- Ambient Glow -->
			<div class="absolute top-[-10%] left-1/2 -translate-x-1/2 w-[80vw] h-[600px] bg-indigo-900/20 blur-[130px] rounded-full animate-pulse-slow"></div>
			<div class="absolute bottom-0 right-0 w-[600px] h-[600px] bg-purple-900/10 blur-[150px] rounded-full"></div>
		{/if}
	</div>

	<!-- Navigation -->
	<nav
		class="fixed top-0 left-0 right-0 z-50 transition-all duration-500 border-b {scrolled
			? preferencesStore.theme === 'light' 
				? 'py-3 bg-white backdrop-blur-xl border-black/[0.05] shadow-sm' 
				: 'py-3 bg-black/80 backdrop-blur-xl border-white/5 shadow-2xl'
			: 'py-6 bg-transparent border-transparent'}"
	>
		<div class="max-w-7xl mx-auto px-6 flex items-center justify-between">
			<!-- Logo -->
			<button 
				class="flex items-center gap-3 group cursor-pointer" 
				onclick={() => goto('/')}
				in:fly={{ y: -20, duration: 600, delay: 0, easing: cubicOut }}
			>
				<div class="relative w-9 h-9 flex items-center justify-center transition-transform duration-500 group-hover:scale-110 group-hover:rotate-12">
					<div class="absolute inset-0 bg-primary blur-lg opacity-20 group-hover:opacity-40 transition-opacity"></div>
					{#key preferencesStore.theme}
						<img src={getAppLogo(preferencesStore.theme)} alt="EGO" class="relative w-full h-full object-contain drop-shadow-lg" />
					{/key}
				</div>
				<span class="text-xl font-bold tracking-tight font-mono">EGO</span>
			</button>

			<!-- Menu Items (Desktop) -->
			<div class="hidden md:flex items-center gap-1" in:fly={{ y: -20, duration: 600, delay: 100, easing: cubicOut }}>
				{#each navItems as item}
					<a
						href={item.href}
						class="px-4 py-2 text-sm font-medium opacity-60 hover:opacity-100 hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-all"
					>
						{$t(`knowego.nav_${item.key}`)}
					</a>
				{/each}
			</div>

			<!-- Actions -->
			<div class="flex items-center gap-2 md:gap-3" in:fly={{ y: -20, duration: 600, delay: 200, easing: cubicOut }}>
				<button
					onclick={toggleTheme}
					class="p-2.5 rounded-lg hover:bg-black/5 dark:hover:bg-white/5 transition-all text-text-secondary hover:text-text-primary group active:scale-90"
					aria-label="Toggle Theme"
				>
					<div class="transition-transform duration-500 ease-spring group-hover:rotate-[360deg]">
						{#if preferencesStore.theme === 'light'}
							<Moon class="w-5 h-5 text-gray-700" />
						{:else}
							<Sun class="w-5 h-5 text-gray-300" />
						{/if}
					</div>
				</button>
				<div class="hidden sm:block">
					<LanguageSwitcher />
				</div>
				<div class="h-6 w-px bg-black/[0.08] dark:bg-white/10 mx-1 hidden sm:block"></div>
				<button
					onclick={() => goto('/chat/new')}
					class="group relative overflow-hidden px-6 py-2.5 rounded-xl bg-accent dark:bg-white text-white dark:text-black font-bold text-sm transition-all hover:scale-[1.02] active:scale-[0.98] shadow-lg shadow-accent/20 dark:shadow-white/10"
				>
					<span class="relative z-10 flex items-center gap-2">
						{$t('knowego.cta_start')}
						<ArrowRight class="w-4 h-4 transition-transform duration-300 group-hover:translate-x-1" />
					</span>
					<div class="absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/20 to-transparent transition-transform duration-700 ease-in-out group-hover:translate-x-full"></div>
				</button>

				<!-- Mobile Menu Toggle -->
				<button
					class="md:hidden p-2 rounded-lg hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
					onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
				>
					{#if mobileMenuOpen}
						<X class="w-6 h-6" />
					{:else}
						<Menu class="w-6 h-6" />
					{/if}
				</button>
			</div>
		</div>

		<!-- Mobile Menu Overlay -->
		{#if mobileMenuOpen}
			<div
				class="md:hidden absolute top-full left-0 right-0 border-b backdrop-blur-2xl transition-all h-screen {preferencesStore.theme === 'light' ? 'bg-white/95 border-black/5' : 'bg-black/95 border-white/5'}"
				transition:slide={{ duration: 300, axis: 'y' }}
			>
				<div class="flex flex-col p-6 gap-2">
					{#each navItems as item}
						<a
							href={item.href}
							class="text-lg font-medium p-4 rounded-xl hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
							onclick={() => (mobileMenuOpen = false)}
						>
							{$t(`knowego.nav_${item.key}`)}
						</a>
					{/each}
					<div class="mt-4 pt-6 border-t border-black/5 dark:border-white/5 flex items-center justify-between px-4">
						<span class="text-sm font-medium opacity-60">Language</span>
						<LanguageSwitcher />
					</div>
				</div>
			</div>
		{/if}
	</nav>


	<!-- Hero Section -->
	<section class="relative z-10 pt-24 pb-20 lg:pt-32 lg:pb-40 px-6 overflow-visible">
		<div class="max-w-7xl mx-auto flex flex-col items-center text-center">
			
			<!-- Text Content -->
			<div class="flex flex-col items-center max-w-4xl z-20" in:fly={{ y: 30, duration: 1000, delay: 300, easing: cubicOut }}>
				<!-- Glitch Logo -->
				<div class="relative mb-12 group" in:scale={{ duration: 1000, delay: 200, start: 0.8, easing: elasticOut }}>
					<div class="absolute inset-0 bg-accent blur-[60px] opacity-20 group-hover:opacity-40 transition-opacity duration-1000"></div>
					{#key preferencesStore.theme}
						<img 
							src={getAppLogo(preferencesStore.theme)} 
							alt="EGO Logo" 
							class="relative w-24 h-24 md:w-32 md:h-32 object-contain logo-malfunction transition-transform duration-700 hover:scale-110" 
						/>
					{/key}
				</div>

				<h1 class="text-5xl sm:text-6xl md:text-7xl lg:text-8xl font-black tracking-[-0.04em] leading-[1.2] mb-8 logo-malfunction drop-shadow-sm flex flex-wrap justify-center gap-x-[0.25em] pb-6 overflow-visible
					{preferencesStore.theme === 'light' ? 'text-slate-950' : 'text-white'}">
					{#key $locale}
						{#each $t('knowego.hero_title').split(' ') as word, i}
							{@const isTarget = word.toLowerCase().includes('прозрачными') || word.toLowerCase().includes('transparent')}
							{@const isAI = word.includes('AI')}
							{@const isReasoning = word.toLowerCase().includes('рассуждениями') || word.toLowerCase().includes('reasoning')}
							
							<span class="inline-block {isTarget && isChanging ? 'animate-glitch-fast text-accent' : ''}" in:fly={{ y: 30, duration: 1000, delay: 400 + (i * 120), easing: cubicOut }}>
								{#if isAI}
									<span class="relative inline-flex items-center font-mono font-black text-accent bg-accent/[0.03] dark:bg-accent/10 px-3 py-1 rounded-2xl border border-accent/20 shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.1)] mr-1">AI</span>{word.replace('AI', '')}
								{:else if isTarget}
									<!-- Stable container for dynamic words -->
									<span class="inline-grid grid-cols-1 grid-rows-1 text-center">
										<span class="invisible pointer-events-none row-start-1 col-start-1 px-1">{$locale === 'ru' ? 'прозрачными' : 'transparent'}</span>
										<span class="row-start-1 col-start-1">
											{($locale === 'ru' ? alternateWordsRu : alternateWordsEn)[currentWordIdx]}
										</span>
									</span>
								{:else if isReasoning}
									<span class="relative inline-block">
										{word}
										<span class="absolute -bottom-2 left-0 w-full h-[3px] bg-accent/20 rounded-full"></span>
									</span>
								{:else}
									{word}
								{/if}
							</span>
						{/each}
					{/key}
				</h1>

				<p class="text-lg md:text-2xl opacity-60 leading-relaxed mb-10 max-w-2xl font-light" in:fly={{ y: 20, duration: 800, delay: 600 }}>
					{@html $t('knowego.hero_subtitle')}
				</p>

				<div class="flex flex-wrap justify-center gap-4" in:fly={{ y: 20, duration: 800, delay: 700 }}>
					<button
						onclick={() => goto('/chat/new')}
						class="px-10 py-5 rounded-xl bg-black text-white dark:bg-white dark:text-black font-bold text-lg hover:shadow-[0_8px_30px_rgba(0,0,0,0.12)] hover:-translate-y-1 active:translate-y-0 transition-all flex items-center gap-2 group"
					>
						{$t('knowego.cta_start')}
						<ArrowRight class="w-5 h-5 group-hover:translate-x-1 transition-transform" />
					</button>
					<button
						onclick={() => document.getElementById('how-it-works')?.scrollIntoView({ behavior: 'smooth' })}
						class="px-10 py-5 rounded-xl border border-black/10 dark:border-white/10 font-bold text-lg hover:bg-black/5 dark:hover:bg-white/5 transition-all flex items-center gap-2"
					>
						{$t('knowego.hero_meet_ego')}
					</button>
				</div>
			</div>
		</div>
	</section>

	<!-- Capabilities / Tools -->
	<section id="tools" class="relative z-10 py-24 px-6 border-t border-black/[0.03] dark:border-white/5">
		<div class="max-w-7xl mx-auto">
			<div class="mb-16 md:mb-24 text-center" use:viewport>
				<h2 class="text-4xl md:text-6xl font-black tracking-tight mb-6">{$t('knowego.features_capabilities_title')}</h2>
				<p class="text-lg md:text-xl opacity-60 max-w-2xl mx-auto leading-relaxed font-light">{$t('knowego.features_capabilities_subtitle')}</p>
			</div>

			<!-- Module Grid -->
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
				{#each tools as tool, i}
					<div
						class="group relative flex flex-col p-8 rounded-[2.5rem] border transition-all duration-500 hover:-translate-y-1 
						{preferencesStore.theme === 'light' 
							? 'bg-[#FBFBFD] border-black/[0.03] hover:bg-white hover:shadow-[0_30px_60px_rgba(0,0,0,0.06)]' 
							: 'bg-[#0A0A0A] border-white/5 hover:bg-[#0F0F0F] hover:shadow-[0_30px_60px_rgba(0,0,0,0.4)]'}"
						use:viewport
					>
						<!-- Header -->
						<div class="flex items-start justify-between mb-12">
							<div class="w-16 h-16 rounded-2xl flex items-center justify-center transition-all duration-500 group-hover:scale-110
								{preferencesStore.theme === 'light' ? `bg-${tool.color}-500/[0.08]` : `bg-${tool.color}-500/10`}">
								<tool.icon class="w-8 h-8 text-{tool.color}-500" />
							</div>
							<div class="px-3 py-1 rounded-full border 
								{preferencesStore.theme === 'light' ? 'bg-black/[0.03] border-black/[0.05]' : 'bg-white/[0.03] border-white/[0.05]'}">
								<span class="text-[10px] font-bold opacity-40 tracking-widest uppercase">{tool.tag}</span>
							</div>
						</div>

						<!-- Content -->
						<div class="mt-auto">
							<h3 class="text-2xl font-bold mb-4 tracking-tight">{$t(`knowego.tool_${tool.id}_title`)}</h3>
							<p class="opacity-60 leading-relaxed text-sm md:text-base font-medium">{$t(`knowego.tool_${tool.id}_desc`)}</p>
							
							<!-- Hover-only info -->
							<div class="mt-6 flex items-center gap-2 font-mono text-[10px] font-bold tracking-widest text-accent opacity-0 group-hover:opacity-100 translate-y-2 group-hover:translate-y-0 transition-all duration-300">
								<div class="w-1 h-1 rounded-full bg-accent"></div>
								{tool.status}
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Agentic Loop Section -->
	<section id="agentic-loop" class="relative z-10 py-24 px-6 border-t border-black/[0.03] dark:border-white/5 overflow-hidden">
		<div class="max-w-7xl mx-auto">
			<div class="mb-20 text-center" use:viewport>
				<h2 class="text-4xl md:text-6xl font-black tracking-tight mb-6">{$t('knowego.agentic_loop_title')}</h2>
				<p class="text-lg md:text-xl opacity-60 max-w-2xl mx-auto font-light">{$t('knowego.agentic_loop_subtitle')}</p>
			</div>

			<div class="relative grid grid-cols-1 md:grid-cols-4 gap-8">
				<!-- Connecting Line (Desktop) -->
				<div class="hidden md:block absolute top-12 left-[12%] right-[12%] h-0.5 bg-gradient-to-r from-transparent via-accent/30 to-transparent"></div>
				
				<!-- Step 1: Gather Context -->
				<div class="relative flex flex-col items-center text-center group" use:viewport>
					<div class="w-24 h-24 rounded-full border-4 border-white/5 bg-black/5 dark:bg-white/5 flex items-center justify-center mb-8 relative z-10 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)]">
						<Search class="w-10 h-10 opacity-60 group-hover:opacity-100 group-hover:text-accent transition-all" />
						<div class="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-accent text-white dark:text-black font-bold flex items-center justify-center text-sm shadow-lg">1</div>
					</div>
					<h3 class="text-xl font-bold mb-3">{$t('knowego.agentic_step1_title')}</h3>
					<p class="text-sm opacity-60 max-w-[200px]">{$t('knowego.agentic_step1_desc')}</p>
				</div>

				<!-- Step 2: Take Action -->
				<div class="relative flex flex-col items-center text-center group" use:viewport>
					<div class="w-24 h-24 rounded-full border-4 border-white/5 bg-black/5 dark:bg-white/5 flex items-center justify-center mb-8 relative z-10 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)]">
						<Zap class="w-10 h-10 opacity-60 group-hover:opacity-100 group-hover:text-accent transition-all" />
						<div class="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-accent text-white dark:text-black font-bold flex items-center justify-center text-sm shadow-lg">2</div>
					</div>
					<h3 class="text-xl font-bold mb-3">{$t('knowego.agentic_step2_title')}</h3>
					<p class="text-sm opacity-60 max-w-[200px]">{$t('knowego.agentic_step2_desc')}</p>
				</div>

				<!-- Step 3: Verify Results -->
				<div class="relative flex flex-col items-center text-center group" use:viewport>
					<div class="w-24 h-24 rounded-full border-4 border-white/5 bg-black/5 dark:bg-white/5 flex items-center justify-center mb-8 relative z-10 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)]">
						<Shield class="w-10 h-10 opacity-60 group-hover:opacity-100 group-hover:text-accent transition-all" />
						<div class="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-accent text-white dark:text-black font-bold flex items-center justify-center text-sm shadow-lg">3</div>
					</div>
					<h3 class="text-xl font-bold mb-3">{$t('knowego.agentic_step3_title')}</h3>
					<p class="text-sm opacity-60 max-w-[200px]">{$t('knowego.agentic_step3_desc')}</p>
				</div>

				<!-- Step 4: Reflect & Refine -->
				<div class="relative flex flex-col items-center text-center group" use:viewport>
					<div class="w-24 h-24 rounded-full border-4 border-white/5 bg-black/5 dark:bg-white/5 flex items-center justify-center mb-8 relative z-10 transition-all duration-500 group-hover:scale-110 group-hover:border-accent/50 group-hover:shadow-[0_0_30px_rgba(var(--color-accent-rgb),0.3)]">
						<Brain class="w-10 h-10 opacity-60 group-hover:opacity-100 group-hover:text-accent transition-all" />
						<div class="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-accent text-white dark:text-black font-bold flex items-center justify-center text-sm shadow-lg">4</div>
					</div>
					<h3 class="text-xl font-bold mb-3">{$t('knowego.agentic_step4_title')}</h3>
					<p class="text-sm opacity-60 max-w-[200px]">{$t('knowego.agentic_step4_desc')}</p>
				</div>
			</div>
			
			<!-- Infinite Loop Animation Overlay -->
			<div class="mt-16 flex justify-center opacity-30">
				<div class="flex items-center gap-2 text-xs font-mono uppercase tracking-[0.3em]">
					<div class="w-2 h-2 bg-accent rounded-full animate-pulse"></div>
					Running Agentic Process
				</div>
			</div>
		</div>
	</section>

	<!-- Features (Bento Grid) -->
	<section id="features" class="relative z-10 py-24 px-6 border-t border-black/5 dark:border-white/5">
		<div class="max-w-7xl mx-auto">
			<div class="mb-20" use:viewport>
				<h2 class="text-3xl md:text-5xl font-bold tracking-tight mb-6">{$t('knowego.features_title')}</h2>
				<p class="text-xl opacity-60 max-w-xl">{$t('knowego.features_subtitle')}</p>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-3 gap-6 md:auto-rows-fr auto-rows-auto">
				<!-- Feature 1: Thinking (Large) -->
				<div 
					class="md:col-span-2 md:row-span-2 rounded-[2.5rem] p-8 md:p-12 relative overflow-hidden border transition-all duration-500 group
					{preferencesStore.theme === 'light' 
						? 'bg-white border-black/[0.03] shadow-lg shadow-black/[0.02]' 
						: 'bg-gradient-to-br from-purple-500/5 via-transparent to-transparent border-white/5'}"
					use:viewport
				>
					<div class="relative z-10 flex flex-col h-full">
						<div class="w-12 h-12 rounded-2xl bg-purple-500/10 flex items-center justify-center mb-8">
							<Brain class="w-6 h-6 text-purple-500" />
						</div>
						<h3 class="text-2xl md:text-4xl font-bold mb-4 tracking-tight">{$t('knowego.feature_thinking_title')}</h3>
						<p class="text-base md:text-lg opacity-60 max-w-lg leading-relaxed mb-10">{@html $t('knowego.feature_thinking_desc')}</p>						
						
						<!-- Thinking Stack Visualization -->
						<div class="mt-auto relative h-32 md:h-48 w-full max-w-2xl group/stack">
							<!-- Bottom Card (Synthesis) -->
							<div class="absolute bottom-0 left-4 right-4 p-4 rounded-2xl border transition-all duration-700 translate-y-4 opacity-40 group-hover/stack:translate-y-12 group-hover/stack:opacity-60
								{preferencesStore.theme === 'light' ? 'bg-white border-black/[0.05]' : 'bg-[#0F0F0F] border-white/5'}">
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-3">
										<div class="w-7 h-7 rounded-lg bg-green-500/10 flex items-center justify-center text-green-500">
											<Check class="w-4 h-4" />
										</div>
										<span class="text-xs font-bold opacity-60">Synthesis</span>
									</div>
								</div>
							</div>

							<!-- Middle Card (Execution) -->
							<div class="absolute bottom-4 left-2 right-2 p-4 rounded-2xl border transition-all duration-500 translate-y-2 opacity-80 group-hover/stack:translate-y-6 group-hover/stack:opacity-100
								{preferencesStore.theme === 'light' ? 'bg-white border-black/[0.05] shadow-lg' : 'bg-[#141414] border-white/10 shadow-2xl'}">
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-3">
										<div class="w-7 h-7 rounded-lg bg-blue-500/10 flex items-center justify-center text-blue-500">
											<Search class="w-4 h-4" />
										</div>
										<span class="text-xs font-bold">Tool Execution</span>
									</div>
									<div class="px-2 py-0.5 rounded bg-blue-500/10 text-blue-500 text-[9px] font-black uppercase tracking-tighter">Running</div>
								</div>
							</div>

							<!-- Top Card (Analysis) -->
							<div class="absolute bottom-8 left-0 right-0 p-5 rounded-2xl border transition-all duration-500 group-hover/stack:-translate-y-2
								{preferencesStore.theme === 'light' ? 'bg-white border-black/[0.08] shadow-xl' : 'bg-[#1A1A1A] border-white/20 shadow-2xl'}">
								<div class="flex items-center justify-between mb-4">
									<div class="flex items-center gap-3">
										<div class="w-8 h-8 rounded-lg bg-purple-500/10 flex items-center justify-center text-purple-500">
											<Activity class="w-4 h-4" />
										</div>
										<span class="text-sm font-bold tracking-tight">Strategic Planning</span>
									</div>
									<span class="text-[9px] font-bold opacity-30 uppercase tracking-[0.2em]">Step 01</span>
								</div>
								<div class="space-y-2">
									<div class="h-1.5 w-full bg-black/[0.03] dark:bg-white/5 rounded-full overflow-hidden">
										<div class="h-full w-2/3 bg-accent animate-pulse-gentle"></div>
									</div>
									<div class="flex justify-between text-[10px] font-bold opacity-40 uppercase tracking-widest">
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
					class="rounded-[2.5rem] p-8 md:p-10 border transition-all duration-300 group hover:-translate-y-1
					{preferencesStore.theme === 'light'
						? 'bg-blue-500/[0.03] border-black/[0.03] hover:bg-white hover:shadow-xl hover:shadow-blue-500/[0.05]'
						: 'bg-blue-500/5 border-white/5 hover:bg-blue-500/[0.08]'}"
					use:viewport
				>
					<Database class="w-10 h-10 text-blue-500 mb-6 group-hover:rotate-12 transition-transform duration-500" />
					<h3 class="text-xl md:text-2xl font-bold mb-4 tracking-tight">{$t('knowego.feature_memory_title')}</h3>
					<p class="opacity-60 leading-relaxed text-sm md:text-base font-medium mb-6">{@html $t('knowego.feature_memory_desc')}</p>
					<div class="font-mono text-[10px] font-bold tracking-widest text-blue-500 opacity-0 group-hover:opacity-100 translate-y-2 group-hover:translate-y-0 transition-all duration-300">
						PERMANENT VECTOR CONTEXT
					</div>
				</div>

				<!-- Feature 3: Tools -->
				<div 
					class="rounded-[2.5rem] p-8 md:p-10 border transition-all duration-300 group hover:-translate-y-1
					{preferencesStore.theme === 'light'
						? 'bg-emerald-500/[0.03] border-black/[0.03] hover:bg-white hover:shadow-xl hover:shadow-emerald-500/[0.05]'
						: 'bg-emerald-500/5 border-white/5 hover:bg-emerald-500/[0.08]'}"
					use:viewport
				>
					<Cpu class="w-10 h-10 text-emerald-500 mb-6 group-hover:scale-110 transition-transform duration-500" />
					<h3 class="text-xl md:text-2xl font-bold mb-4 tracking-tight">{$t('knowego.feature_tools_title')}</h3>
					<p class="opacity-60 leading-relaxed text-sm md:text-base font-medium mb-6">{@html $t('knowego.feature_tools_desc')}</p>
					<div class="font-mono text-[10px] font-bold tracking-widest text-emerald-500 opacity-0 group-hover:opacity-100 translate-y-2 group-hover:translate-y-0 transition-all duration-300">
						FULL API INTEGRATION
					</div>
				</div>

				<!-- Feature 4: Models (Wide) -->
				<div 
					class="md:col-span-3 rounded-[2.5rem] p-8 md:p-12 flex flex-col md:flex-row items-center gap-10 border backdrop-blur-sm
					{preferencesStore.theme === 'light'
						? 'bg-white border-black/[0.03]'
						: 'bg-white/[0.02] border-white/5'}"
					use:viewport
				>
					<div class="flex-1 text-center md:text-left">
						<div class="w-12 h-12 rounded-2xl bg-amber-500/10 flex items-center justify-center mb-6 mx-auto md:mx-0">
							<Globe class="w-6 h-6 text-amber-500" />
						</div>
						<h3 class="text-2xl md:text-3xl font-bold mb-4 tracking-tight">{$t('knowego.feature_models_title')}</h3>
						<p class="text-base md:text-lg opacity-60 leading-relaxed font-medium">{$t('knowego.feature_models_desc')}</p>
					</div>
					<div class="flex-1 flex flex-wrap gap-3 justify-center md:justify-end">
						{#each ['EGO', 'OpenAI', 'Claude', 'Gemini', 'Grok'] as model}
							<div class="px-5 py-2.5 rounded-xl border text-sm font-bold opacity-60 hover:opacity-100 hover:scale-105 transition-all cursor-default shadow-sm
								{preferencesStore.theme === 'light' ? 'bg-white border-black/[0.05]' : 'bg-white/5 border-white/5'}">
								{model}
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- How it works (Pipeline) -->
	<section id="how-it-works" class="relative z-10 py-24 px-6 border-t border-black/[0.03] dark:border-white/5 overflow-hidden">
		<div class="max-w-7xl mx-auto">
			<div class="mb-20 md:mb-32 text-center" use:viewport>
				<h2 class="text-4xl md:text-6xl font-black tracking-tight mb-6">{$t('knowego.how_title')}</h2>
				<p class="text-lg md:text-xl opacity-60 max-w-2xl mx-auto font-light">{$t('knowego.how_subtitle')}</p>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-3 gap-12 md:gap-8">
				{#each [{ step: 1, color: 'purple', icon: MessageSquare }, { step: 2, color: 'blue', icon: Brain }, { step: 3, color: 'green', icon: Database }] as { step, color, icon }, i}
					{@const Icon = icon}
					<div class="relative flex flex-col group" in:fly={{ y: 30, duration: 800, delay: 200 * i, easing: cubicOut }}>
						<!-- Large Background Number -->
						<div class="absolute -top-12 -left-4 text-8xl md:text-9xl font-black opacity-[0.03] dark:opacity-[0.05] pointer-events-none transition-all duration-700 group-hover:opacity-10 group-hover:-translate-y-2">
							0{step}
						</div>

						<div class="relative z-10">
							<!-- Icon Sphere -->
							<div class="w-16 h-16 rounded-3xl mb-8 flex items-center justify-center transition-all duration-500 group-hover:scale-110 group-hover:rotate-3
								{preferencesStore.theme === 'light' 
									? 'bg-white shadow-xl shadow-black/[0.03] border border-black/[0.03]' 
									: 'bg-white/5 border border-white/5 shadow-2xl'}">
								<Icon class="w-7 h-7 text-{color}-500" />
							</div>

							<h3 class="text-2xl font-bold mb-4 tracking-tight">{$t(`knowego.how_step${step}_title`)}</h3>
							<p class="opacity-60 leading-relaxed text-sm md:text-base font-medium mb-8">{$t(`knowego.how_step${step}_desc`)}</p>

							<!-- Visual element for each step -->
							<div class="mt-auto h-1 w-full bg-black/[0.03] dark:bg-white/5 rounded-full overflow-hidden">
								<div class="h-full w-0 group-hover:w-full transition-all duration-1000 ease-in-out bg-{color}-500/40"></div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Modes Section -->
	<section id="modes" class="relative z-10 py-24 px-6 border-t border-black/[0.03] dark:border-white/5 overflow-hidden">
		<div class="max-w-7xl mx-auto">
			<div class="mb-16 md:mb-20 text-center" use:viewport>
				<h2 class="text-4xl md:text-6xl font-black tracking-tight mb-6">{$t('knowego.modes_title')}</h2>
				<p class="text-lg md:text-xl opacity-60 max-w-2xl mx-auto font-light">{$t('knowego.modes_subtitle')}</p>
			</div>

			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
				{#each modes as mode, i}
					{@const ModeIcon = mode.icon}
					<div 
						class="group relative flex flex-col p-8 rounded-[2.5rem] border transition-all duration-500 hover:-translate-y-1
						{preferencesStore.theme === 'light' 
							? 'bg-[#FBFBFD] border-black/[0.03] hover:bg-white hover:shadow-[0_30px_60px_rgba(0,0,0,0.05)]' 
							: 'bg-[#0A0A0A] border-white/5 hover:bg-[#0F0F0F] hover:shadow-[0_30px_60px_rgba(0,0,0,0.4)]'}"
						in:fly={{ y: 20, duration: 800, delay: 100 * i, easing: cubicOut }}
					>
						<div class="w-12 h-12 rounded-xl flex items-center justify-center mb-8 transition-transform duration-500 group-hover:scale-110
							{preferencesStore.theme === 'light' ? `bg-${mode.color}-500/[0.08]` : `bg-${mode.color}-500/10`}">
							<ModeIcon class="w-6 h-6 text-{mode.color}-500" />
						</div>
						
						<h3 class="text-xl font-bold mb-3 tracking-tight">{$t(`knowego.mode_${mode.id}_title`)}</h3>
						<p class="text-sm opacity-60 leading-relaxed font-medium mb-6">{$t(`knowego.mode_${mode.id}_desc`)}</p>
						
						<div class="mt-auto flex items-center gap-2 font-mono text-[10px] font-bold tracking-widest text-accent opacity-0 group-hover:opacity-100 translate-y-2 group-hover:translate-y-0 transition-all duration-300">
							<div class="w-1 h-1 rounded-full bg-accent"></div>
							{mode.detail}
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- CTA Section -->
	<section class="relative z-10 py-32 px-6">
		<div 
			class="max-w-6xl mx-auto rounded-[3rem] p-12 md:p-32 text-center relative overflow-hidden transition-all duration-700
			{preferencesStore.theme === 'light' 
				? 'bg-white border border-black/[0.03] shadow-[0_40px_100px_rgba(0,0,0,0.04)]' 
				: 'bg-[#050508] border border-white/5 shadow-2xl'}"
			use:viewport
		>
			<!-- Animated Ambient Glow for CTA -->
			<div class="absolute inset-0 pointer-events-none">
				<div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[80%] h-[80%] blur-[120px] rounded-full opacity-20
					{preferencesStore.theme === 'light' ? 'bg-accent/20' : 'bg-accent/30'}"></div>
			</div>

			<div class="relative z-10 flex flex-col items-center">
				<h2 class="text-4xl md:text-7xl font-black mb-8 tracking-[-0.04em] leading-tight logo-malfunction flex flex-col items-center
					{preferencesStore.theme === 'light' ? 'text-slate-950' : 'text-white'}">
					<span>{$t('knowego.cta_block_title')}</span>
					
					{#key $locale}
						<!-- Stable container for dynamic CTA word -->
						<span class="inline-grid grid-cols-1 grid-rows-1 text-center min-w-[3ch] mt-2">
							<span class="invisible pointer-events-none row-start-1 col-start-1">{$locale === 'ru' ? 'исследовать?' : 'accelerate?'}</span>
							<span class="row-start-1 col-start-1 {isChanging ? 'animate-glitch-fast text-accent' : ''}">
								{($locale === 'ru' ? ctaWordsRu : ctaWordsEn)[currentCtaWordIdx]}
							</span>
						</span>
					{/key}
				</h2>
				
				<p class="text-lg md:text-2xl opacity-60 mb-12 max-w-2xl mx-auto font-light">
					{$t('knowego.cta_block_sub')}
				</p>
				
				<button
					onclick={() => goto('/chat/new')}
					class="group relative overflow-hidden px-12 py-5 rounded-2xl font-black text-xl transition-all hover:scale-105 active:scale-95 shadow-2xl
					{preferencesStore.theme === 'light' 
						? 'bg-slate-950 text-white shadow-black/10' 
						: 'bg-white text-black shadow-white/5'}"
				>
					<span class="relative z-10 flex items-center gap-3">
						{$t('knowego.cta_start')}
						<ArrowRight class="w-6 h-6 transition-transform duration-300 group-hover:translate-x-1" />
					</span>
					<!-- Shimmer Effect on Button -->
					<div class="absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/10 dark:via-black/5 to-transparent transition-transform duration-1000 ease-in-out group-hover:translate-x-full"></div>
				</button>
			</div>
		</div>
	</section>

	<!-- Footer -->
	<footer class="relative z-10 py-20 px-6 border-t backdrop-blur-xl
		{preferencesStore.theme === 'light' 
			? 'bg-white border-black/[0.05] text-slate-900' 
			: 'bg-black/20 border-white/5 text-white'}">
		<div class="max-w-7xl mx-auto">
			<div class="flex flex-col md:flex-row justify-between items-start gap-12 mb-20">
				<div class="w-full md:max-w-sm">
					<div class="flex items-center gap-2.5 mb-6">
						<div class="w-8 h-8 rounded-lg flex items-center justify-center p-1.5 shadow-xl transition-colors duration-300
							{preferencesStore.theme === 'light' ? 'bg-white shadow-black/5' : 'bg-black'}">
							{#key preferencesStore.theme}
								<!-- In light theme (white bg), we need dark logo. In dark theme (black bg), we need light logo. -->
								<img src={getAppLogo(preferencesStore.theme === 'light' ? 'light' : 'dark')} alt="EGO" class="w-full h-full object-contain" />
							{/key}
						</div>
						<span class="text-xl font-bold tracking-tight">EGO</span>
					</div>
					<p class="text-sm leading-relaxed mb-8 font-medium {preferencesStore.theme === 'light' ? 'opacity-60' : 'opacity-70'}">
						{$t('knowego.footer_disclaimer')}
					</p>
				</div>

				<div class="w-full md:w-auto grid grid-cols-2 sm:grid-cols-3 gap-10 md:gap-24">
					<div class="space-y-4">
						<h4 class="font-bold text-lg">{$t('knowego.footer_product')}</h4>
						<ul class="space-y-3 text-sm font-medium {preferencesStore.theme === 'light' ? 'opacity-60' : 'opacity-70'}">
							<li><a href="#features" class="hover:text-accent transition-colors">{$t('knowego.footer_features')}</a></li>
							<li><a href="#tools" class="hover:text-accent transition-colors">{$t('knowego.footer_tools')}</a></li>
							<li><a href="#modes" class="hover:text-accent transition-colors">{$t('knowego.footer_modes')}</a></li>
						</ul>
					</div>
					<div class="space-y-4">
						<h4 class="font-bold text-lg">{$t('knowego.footer_legal')}</h4>
						<ul class="space-y-3 text-sm font-medium {preferencesStore.theme === 'light' ? 'opacity-60' : 'opacity-70'}">
							<li><a href="/terms" class="hover:text-accent transition-colors">{$t('knowego.footer_terms_of_service')}</a></li>
							<li><a href="/privacy" class="hover:text-accent transition-colors">{$t('knowego.footer_privacy_policy')}</a></li>
						</ul>
					</div>
					<div class="space-y-4">
						<h4 class="font-bold text-lg">{$t('knowego.footer_contact')}</h4>
						<ul class="space-y-3 text-sm font-medium {preferencesStore.theme === 'light' ? 'opacity-60' : 'opacity-70'}">
							<li><a href="mailto:support@askego.xyz" class="hover:text-accent transition-colors">support@askego.xyz</a></li>
						</ul>
					</div>
				</div>
			</div>

			<div class="pt-10 border-t flex flex-col sm:flex-row justify-between gap-6 text-xs font-medium
				{preferencesStore.theme === 'light' ? 'border-black/[0.05] opacity-50' : 'border-white/5 opacity-60'}">
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
		0%, 100% { opacity: 0.1; transform: scale(1); }
		50% { opacity: 0.15; transform: scale(1.1); }
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
		0% { background-position: 0% 50%; }
		50% { background-position: 100% 50%; }
		100% { background-position: 0% 50%; }
	}

	/* Fix for dynamic colors in tailwind - if they are not picked up, we use style or safelist */
	.bg-blue-500\/10 { background-color: rgba(59, 130, 246, 0.1); }
	.bg-red-500\/10 { background-color: rgba(239, 68, 68, 0.1); }
	.bg-green-500\/10 { background-color: rgba(34, 197, 94, 0.1); }
	.bg-purple-500\/10 { background-color: rgba(168, 85, 247, 0.1); }
	.bg-orange-500\/10 { background-color: rgba(249, 115, 22, 0.1); }
	.bg-amber-500\/10 { background-color: rgba(245, 158, 11, 0.1); }
	.bg-yellow-500\/10 { background-color: rgba(234, 179, 8, 0.1); }
	.bg-indigo-500\/10 { background-color: rgba(99, 102, 241, 0.1); }
	.bg-emerald-500\/10 { background-color: rgba(16, 185, 129, 0.1); }

	.text-blue-500 { color: #3b82f6; }
	.text-red-500 { color: #ef4444; }
	.text-green-500 { color: #22c55e; }
	.text-purple-500 { color: #a855f7; }
	.text-orange-500 { color: #f97316; }
	.text-amber-500 { color: #f59e0b; }
	.text-yellow-500 { color: #eab308; }
	.text-indigo-500 { color: #6366f1; }
	.text-emerald-500 { color: #10b981; }

	.border-blue-500\/10 { border-color: rgba(59, 130, 246, 0.1); }
	.border-red-500\/10 { border-color: rgba(239, 68, 68, 0.1); }
	.border-green-500\/10 { border-color: rgba(34, 197, 94, 0.1); }
	.border-purple-500\/10 { border-color: rgba(168, 85, 247, 0.1); }
	.border-orange-500\/10 { border-color: rgba(249, 115, 22, 0.1); }
	.border-amber-500\/10 { border-color: rgba(245, 158, 11, 0.1); }

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
		0%, 86%, 100% { transform: translate(0); filter: none; opacity: 1; text-shadow: none; }
		87% { transform: translate(-2px, 1px); filter: hue-rotate(90deg) saturate(1.5); clip-path: inset(10% 0 40% 0); color: #ff003c; opacity: 0.9; text-shadow: 2px 0 #00fff2; }
		88% { transform: translate(2px, -1px); filter: hue-rotate(-90deg) contrast(2); clip-path: inset(40% 0 10% 0); color: #ff003c; opacity: 0.8; text-shadow: -2px 0 #00fff2; }
		89% { transform: translate(0); filter: none; clip-path: none; opacity: 1; text-shadow: none; }
		90% { transform: translate(-3px, 2px); filter: hue-rotate(45deg); clip-path: inset(5% 0 60% 0); color: #ff003c; opacity: 0.7; }
		91% { transform: translate(0); filter: none; clip-path: none; opacity: 1; }
	}

	.animate-glitch-fast {
		animation: glitch-fast 0.2s linear infinite;
	}

	@keyframes glitch-fast {
		0% { transform: translate(0); filter: hue-rotate(0deg); }
		25% { transform: translate(2px, -2px); filter: hue-rotate(90deg); }
		50% { transform: translate(-2px, 2px); filter: contrast(2); }
		75% { transform: translate(1px, -1px); filter: invert(0.1); }
		100% { transform: translate(0); filter: none; }
	}
</style>