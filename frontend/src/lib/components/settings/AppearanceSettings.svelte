<script lang="ts">
	import { setTheme, setBackgroundSpheres } from '$lib/stores/preferences.svelte.ts';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { _ } from 'svelte-i18n';
	import { Sun, Moon, Sparkles } from '@lucide/svelte';

	function handleThemeChange(theme: 'light' | 'dark') {
		setTheme(theme);
	}

	function handleSpheresToggle(event: Event) {
		const checked = (event.target as HTMLInputElement).checked;
		setBackgroundSpheres(checked);
	}
</script>

<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
	<div
		class="group rounded-2xl border border-tertiary bg-secondary/30 p-4 transition-all hover:border-accent/40 hover:bg-secondary/50"
	>
		<div class="mb-3 flex items-center justify-between">
			<span class="text-sm font-medium text-text-primary">{$_('settings.theme')}</span>
			<div class="flex gap-1 rounded-xl bg-primary/50 p-1">
				<button
					class="flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all {preferencesStore.theme ===
					'light'
						? 'bg-accent text-white shadow-lg shadow-accent/20'
						: 'text-text-secondary hover:bg-tertiary hover:text-text-primary'}"
					onclick={() => handleThemeChange('light')}
					aria-pressed={preferencesStore.theme === 'light'}
				>
					<Sun class="h-3.5 w-3.5" />
					{$_('settings.light')}
				</button>
				<button
					class="flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all {preferencesStore.theme ===
					'dark'
						? 'bg-accent text-white shadow-lg shadow-accent/20'
						: 'text-text-secondary hover:bg-tertiary hover:text-text-primary'}"
					onclick={() => handleThemeChange('dark')}
					aria-pressed={preferencesStore.theme === 'dark'}
				>
					<Moon class="h-3.5 w-3.5" />
					{$_('settings.dark')}
				</button>
			</div>
		</div>
		<p class="text-xs leading-relaxed text-text-secondary">{$_('settings.theme_hint')}</p>
	</div>

	<div
		class="group rounded-2xl border border-tertiary bg-secondary/30 p-4 transition-all hover:border-accent/40 hover:bg-secondary/50"
	>
		<div class="mb-3 flex items-center justify-between">
			<div class="flex items-center gap-2">
				<Sparkles class="h-4 w-4 text-accent" />
				<span class="text-sm font-medium text-text-primary">{$_('settings.spheres')}</span>
			</div>
			<label class="relative inline-flex cursor-pointer items-center">
				<input
					type="checkbox"
					class="sr-only"
					checked={preferencesStore.backgroundSpheres}
					onchange={handleSpheresToggle}
				/>
				<div
					class="h-6 w-11 rounded-full border border-tertiary/50 transition-colors duration-200 {preferencesStore.backgroundSpheres
						? 'bg-accent'
						: 'bg-tertiary/50'}"
				>
					<div
						class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform duration-200 {preferencesStore.backgroundSpheres
							? 'translate-x-5'
							: 'translate-x-0'}"
					></div>
				</div>
			</label>
		</div>
		<p class="text-xs leading-relaxed text-text-secondary">{$_('settings.spheres_hint')}</p>
	</div>
</div>
