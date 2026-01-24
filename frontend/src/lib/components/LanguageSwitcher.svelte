<script lang="ts">
	import { locale } from 'svelte-i18n';
	function toggleLocale() {
		$locale = $locale === 'ru' ? 'en' : 'ru';
	}
</script>

<button
	onclick={toggleLocale}
	class="rounded-full p-2 transition-colors hover:bg-tertiary"
	title="Сменить язык / Change language"
>
	<div class="flex h-5 w-5 items-center justify-center font-semibold">
		{$locale === 'ru' ? 'EN' : 'RU'}
	</div>
</button>
