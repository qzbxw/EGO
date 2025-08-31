<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	let { show = false }: { show: boolean } = $props();
</script>
{#if show}
	<div class="absolute inset-0 z-40 flex items-center justify-center bg-primary/60 backdrop-blur-sm">
		<div class="flex flex-col items-center justify-center text-center not-prose px-4">
			<img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-16 h-16 text-accent mb-3" />
			<div class="rounded-xl px-3.5 py-2 bg-secondary shadow-lg inline-flex items-center gap-2 loading-bubble-animation mx-1 md:mx-2">
				<div class="text-sm font-medium text-transparent bg-clip-text bg-gradient-to-r from-text-secondary/70 via-text-primary to-text-secondary/70 bg-[length:200%_100%] animate-shine">
					{$_('chat.creating_session') || 'Создаём сессию для вас…'}
				</div>
			</div>
		</div>
	</div>
{/if}