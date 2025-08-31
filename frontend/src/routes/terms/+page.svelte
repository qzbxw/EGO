<script lang="ts">
    import { _ } from 'svelte-i18n';
    import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
    import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
    let view = $state<'tldr' | 'full'>('tldr');
    const isDark = $derived(preferencesStore.theme !== 'light');
    const fullSections = $derived([
        { id: 'service', title: $_('legal.service_title'), content: $_('legal.terms_service') },
        { id: 'account', title: $_('legal.account_title'), content: $_('legal.terms_account') },
        { id: 'user_content', title: $_('legal.user_content_title'), content: $_('legal.terms_user_content') },
        { id: 'fair_use', title: $_('legal.fair_use_title'), content: $_('legal.terms_fair_use') },
        { id: 'third_party', title: $_('legal.third_party_title'), content: $_('legal.terms_third_party') },
        { id: 'disclaimer', title: $_('legal.disclaimer_title'), content: $_('legal.terms_disclaimer') },
        { id: 'security', title: $_('legal.security_title'), content: $_('legal.terms_security') },
        { id: 'termination', title: $_('legal.termination_title'), content: $_('legal.terms_termination') },
        { id: 'changes', title: $_('legal.changes_title'), content: $_('legal.terms_changes') }
    ]);
    function toggleTheme() {
        setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
    }
</script>
<svelte:head>
    <title>{$_('legal.terms_full_title')} • EGO</title>
    <meta name="robots" content="noindex" />
</svelte:head>
<div class={`relative w-full min-h-screen overflow-y-auto ${isDark ? 'bg-black' : 'bg-white'}`}>
	<div class="mx-auto max-w-5xl px-6 py-10 lg:grid lg:grid-cols-[200px_1fr] lg:gap-10">
		<aside class="hidden lg:block sticky top-10 self-start">
			{#if view === 'full'}
				<h3 class={`text-sm font-semibold mb-2 ${isDark ? 'text-white/80' : 'text-gray-700'}`}>{$_('legal.toc_title') ?? 'On this page'}</h3>
				<ul class="space-y-1.5">
					{#each fullSections as section}
						<li>
							<a href="#{section.id}" class={`text-sm transition-colors ${isDark ? 'text-white/60 hover:text-white' : 'text-gray-600 hover:text-gray-900'}`}>{section.title}</a>
						</li>
					{/each}
				</ul>
			{/if}
		</aside>
		<main class="min-w-0">
			<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4">
				<div class="min-w-0">
					<h1 class={`text-3xl font-extrabold whitespace-pre-wrap break-words ${isDark ? 'text-white' : 'text-gray-900'}`}>{$_('legal.terms_full_title')}</h1>
					<div class={`mt-1 text-sm ${isDark ? 'text-white/50' : 'text-gray-500'}`}>{$_('legal.doc_version')} • {$_('legal.doc_effective')}</div>
				</div>
				<div class="flex items-center gap-2 self-start sm:self-auto">
					<button onclick={toggleTheme} class={`p-2 rounded-full transition-colors ${isDark ? 'hover:bg-white/10 text-white' : 'hover:bg-black/10 text-gray-700'}`} title="Toggle theme">
						{preferencesStore.theme === 'light' ? '🌙' : '☀️'}
					</button>
					<LanguageSwitcher />
				</div>
			</div>
			<div class="mb-4 flex items-center gap-2">
				<button
					class={`px-3 py-1.5 rounded-md text-sm font-semibold transition-colors border ${isDark ? 'border-white/20' : 'border-gray-300'} ${view === 'tldr' ? `bg-accent ${isDark ? 'text-white' : 'text-gray-900'}` : (isDark ? 'hover:bg-white/10 text-white/80' : 'hover:bg-black/5 text-gray-700')}`}
					onclick={() => (view = 'tldr')}
					aria-pressed={view === 'tldr'}
				>{$_('legal.terms_tldr_title')}</button>
				<button
					class={`px-3 py-1.5 rounded-md text-sm font-semibold transition-colors border ${isDark ? 'border-white/20' : 'border-gray-300'} ${view === 'full' ? `bg-accent ${isDark ? 'text-white' : 'text-gray-900'}` : (isDark ? 'hover:bg-white/10 text-white/80' : 'hover:bg-black/5 text-gray-700')}`}
					onclick={() => (view = 'full')}
					aria-pressed={view === 'full'}
				>{$_('legal.full_text_title') ?? 'Full Text'}</button>
			</div>
			<div class={`rounded-2xl overflow-hidden shadow-2xl ${isDark ? 'border border-white/20 bg-white/5 shadow-black/20' : 'border border-gray-200 bg-white shadow-gray-200/40'}`}>
				<article class={`p-6 prose max-w-none ${isDark ? 'prose-invert text-white/70' : 'text-gray-700'}`}>
					{#if view === 'tldr'}
						<h2 class={`mt-0 ${isDark ? 'text-white' : 'text-gray-900'}`}>{$_('legal.terms_tldr_title')}</h2>
						<div class={`not-prose text-[15px] leading-relaxed ${isDark ? 'text-white/80' : 'text-gray-800'}`}>
							{@html $_('legal.terms_tldr_html')}
						</div>
					{:else}
						<div class="not-prose">
							<h2 class={`mt-0 ${isDark ? 'text-white' : 'text-gray-900'}`}>{$_('legal.terms_full_title')}</h2>
							<p class={`text-sm ${isDark ? 'text-white/60' : 'text-gray-500'}`}>{$_('legal.doc_version')} • {$_('legal.doc_effective')}</p>
						</div>
						<div class="mt-4 space-y-8">
							{#each fullSections as section}
								<section class="scroll-mt-20" id={section.id}>
									<h3 class={`text-lg font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>{section.title}</h3>
									<div class={`mt-2 prose max-w-none ${isDark ? 'prose-invert text-white/70' : 'text-gray-700'}`}>{@html section.content}</div>
								</section>
							{/each}
						</div>
					{/if}
				</article>
			</div>
		</main>
	</div>
</div>