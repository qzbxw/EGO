<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { uiStore, setShowUserSettingsModal } from '$lib/stores/ui.svelte.ts';
	import { _ } from 'svelte-i18n';
	import { loadUserSettings } from '$lib/stores/user-settings.svelte.ts';
	import AppearanceSettings from '$lib/components/settings/AppearanceSettings.svelte';
	import MemorySettings from '$lib/components/settings/MemorySettings.svelte';
	import LLMProviderSettings from '$lib/components/settings/LLMProviderSettings.svelte';
	import DangerZone from '$lib/components/settings/DangerZone.svelte';
	import { Palette, Brain, Cpu, AlertTriangle } from '@lucide/svelte';

	let hasLoadedOnOpen = $state(false);

	$effect(() => {
		if (uiStore.showUserSettingsModal && !hasLoadedOnOpen) {
			hasLoadedOnOpen = true;
			loadUserSettings(false);
		}
		if (!uiStore.showUserSettingsModal) {
			hasLoadedOnOpen = false;
		}
	});
</script>

<Modal
	title={$_('settings.title')}
	show={uiStore.showUserSettingsModal}
	onclose={() => setShowUserSettingsModal(false)}
	size="2xl"
>
	<div class="space-y-10">
		<p class="-mt-4 text-base text-text-secondary">{$_('settings.subtitle')}</p>

		<section>
			<div class="mb-5 flex items-center gap-3 text-text-primary">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10">
					<Palette class="h-5 w-5 text-accent" />
				</div>
				<h3 class="text-lg font-bold tracking-tight">{$_('settings.theme')}</h3>
			</div>
			<AppearanceSettings />
		</section>

		<section>
			<div class="mb-5 flex items-center gap-3 text-text-primary">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10">
					<Brain class="h-5 w-5 text-accent" />
				</div>
				<h3 class="text-lg font-bold tracking-tight">{$_('settings.cross_memory')}</h3>
			</div>
			<MemorySettings />
		</section>

		<section>
			<div class="mb-5 flex items-center gap-3 text-text-primary">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10">
					<Cpu class="h-5 w-5 text-accent" />
				</div>
				<h3 class="text-lg font-bold tracking-tight">{$_('settings.llm_provider')}</h3>
			</div>
			<LLMProviderSettings />
		</section>

		<section class="pt-2">
			<div class="mb-5 flex items-center gap-3 text-red-500">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-red-500/10">
					<AlertTriangle class="h-5 w-5" />
				</div>
				<h3 class="text-lg font-bold tracking-tight">{$_('settings.danger_zone')}</h3>
			</div>
			<DangerZone />
		</section>
	</div>
</Modal>
