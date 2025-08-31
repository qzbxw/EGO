<script lang="ts">
    import { goto } from '$app/navigation';
    import { getAppLogo } from '$lib/config';
    import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
    import { _ } from 'svelte-i18n';
</script>
<div class="flex-1 flex flex-col items-center justify-center p-8 text-center">
	<img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-16 h-16 mb-4 text-accent" />
	<h2 class="text-3xl font-bold text-text-primary">EGO</h2>
	<p class="mt-2 text-text-secondary">{$_('chat.landing_subtitle')}</p>
	<button
		onclick={() => goto('/chat/new')}
		class="mt-6 px-6 py-3 bg-accent text-white font-semibold rounded-lg shadow-lg hover:bg-accent-hover transition-colors focus:outline-none focus:ring-2 focus:ring-accent focus:ring-opacity-50"
	>
		{$_('chat.landing_button')}
	</button>
	<a href="/knowego" class="mt-3 text-sm text-text-secondary hover:text-text-primary">?</a>
</div>