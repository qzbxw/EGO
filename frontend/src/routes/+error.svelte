<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { _ } from 'svelte-i18n';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
</script>
<svelte:head>
	<title>Ошибка: {$page.status} - EGO</title>
</svelte:head>
<div class="h-screen w-full flex items-center justify-center">
	<div class="flex-1 flex flex-col items-center justify-center p-8 text-center max-w-lg mx-auto">
		<a href="/">
			<img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-16 h-16 mb-4 text-accent" />
		</a>
		<h1 class="text-5xl font-bold text-red-400">{$page.status}</h1>
		<h2 class="mt-4 text-2xl font-semibold text-text-primary">Произошла ошибка</h2>
		<p class="mt-2 text-text-secondary">
			{$page.error?.message || 'Что-то пошло не так на сервере.'}
		</p>
		<div class="flex items-center gap-4 mt-8">
			<button
				onclick={() => goto('/chat/new', { replaceState: true })}
				class="px-5 py-2.5 bg-accent text-white font-semibold rounded-lg shadow-lg hover:bg-accent-hover transition-colors focus:outline-none focus:ring-2 focus:ring-accent focus:ring-opacity-50"
			>
				Новый чат
			</button>
			<button
				onclick={() => history.back()}
				class="px-5 py-2.5 bg-secondary text-text-primary font-semibold rounded-lg shadow-lg hover:bg-tertiary transition-colors focus:outline-none focus:ring-2 focus:ring-tertiary focus:ring-opacity-50"
			>
				Вернуться назад
			</button>
		</div>
	</div>
</div>