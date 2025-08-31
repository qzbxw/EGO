<script lang="ts">
	import { setTheme, setBackgroundSpheres } from '$lib/stores/preferences.svelte.ts';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { _ } from 'svelte-i18n';
	function handleThemeChange(theme: 'light' | 'dark') {
		setTheme(theme);
	}
	function handleSpheresToggle(event: Event) {
		const checked = (event.target as HTMLInputElement).checked;
		setBackgroundSpheres(checked);
	}
</script>
<div class="grid grid-cols-1 md:grid-cols-2 gap-3 mb-6 max-w-5xl">
	<div class="rounded-xl border border-tertiary p-3 bg-primary/60">
		<div class="text-sm font-semibold mb-1">{$_('settings.theme')}</div>
		<div class="text-xs text-text-secondary mb-2">{$_('settings.theme_hint')}</div>
		<div class="inline-flex rounded-lg overflow-hidden border border-tertiary">
			<button
				class="px-3 py-1.5 text-sm font-medium transition-colors {preferencesStore.theme === 'light' ? 'bg-accent text-white' : 'bg-secondary hover:bg-tertiary'}"
				onclick={() => handleThemeChange('light')}
				aria-pressed={preferencesStore.theme === 'light'}
			>{$_('settings.light')}</button>
			<button
				class="px-3 py-1.5 text-sm font-medium transition-colors {preferencesStore.theme === 'dark' ? 'bg-accent text-white' : 'bg-secondary hover:bg-tertiary'}"
				onclick={() => handleThemeChange('dark')}
				aria-pressed={preferencesStore.theme === 'dark'}
			>{$_('settings.dark')}</button>
		</div>
	</div>
	<div class="rounded-xl border border-tertiary p-3 bg-primary/60">
		<div class="flex items-center justify-between gap-3">
			<div>
				<div class="text-sm font-semibold">{$_('settings.spheres')}</div>
				<div class="text-xs text-text-secondary">{$_('settings.spheres_hint')}</div>
			</div>
			<label class="inline-flex items-center cursor-pointer select-none">
				<input type="checkbox" class="sr-only" checked={preferencesStore.backgroundSpheres} onchange={handleSpheresToggle} />
				<span class="w-11 h-6 rounded-full relative transition-colors duration-200 {preferencesStore.backgroundSpheres ? 'bg-accent' : 'bg-tertiary'}">
					<span class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white transition-transform duration-200 {preferencesStore.backgroundSpheres ? 'translate-x-5' : 'translate-x-0'}"></span>
				</span>
			</label>
		</div>
	</div>
</div>