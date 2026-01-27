<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
</script>

<svelte:head>
	<title>Ошибка: {$page.status} - EGO</title>
</svelte:head>
<div class="flex h-screen w-full items-center justify-center">
	<div class="mx-auto flex max-w-lg flex-1 flex-col items-center justify-center p-8 text-center">
		<a href="/">
			<img
				src={getAppLogo(preferencesStore.theme)}
				alt="EGO Logo"
				class="mb-4 h-16 w-16 text-accent"
			/>
		</a>
		<h1 class="text-5xl font-bold text-red-400">{$page.status}</h1>
		<h2 class="mt-4 text-2xl font-semibold text-text-primary">Произошла ошибка</h2>
		<p class="mt-2 text-text-secondary">
			{$page.error?.message || 'Что-то пошло не так на сервере.'}
		</p>
		<div class="mt-8 flex items-center gap-4">
			<button
				onclick={() => void goto('/chat/new', { replaceState: true })}
				class="rounded-lg bg-accent px-5 py-2.5 font-semibold text-white shadow-lg transition-colors hover:bg-accent-hover focus:outline-none focus:ring-2 focus:ring-accent focus:ring-opacity-50"
			>
				Новый чат
			</button>
			<button
				onclick={() => history.back()}
				class="rounded-lg bg-secondary px-5 py-2.5 font-semibold text-text-primary shadow-lg transition-colors hover:bg-tertiary focus:outline-none focus:ring-2 focus:ring-tertiary focus:ring-opacity-50"
			>
				Вернуться назад
			</button>
		</div>
	</div>
</div>
