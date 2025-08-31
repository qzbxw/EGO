<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { uiStore, setShowUserSettingsModal } from '$lib/stores/ui.svelte.ts';
	import { _ } from 'svelte-i18n';
	import { loadUserSettings } from '$lib/stores/user-settings.svelte.ts';
	import AppearanceSettings from '$lib/components/settings/AppearanceSettings.svelte';
	import MemorySettings from '$lib/components/settings/MemorySettings.svelte';
	import LLMProviderSettings from '$lib/components/settings/LLMProviderSettings.svelte';
	import DangerZone from '$lib/components/settings/DangerZone.svelte';
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
<Modal title={$_('settings.title')} show={uiStore.showUserSettingsModal} onclose={() => setShowUserSettingsModal(false)}>
	<div class="p-4">
		<p class="text-text-secondary mb-4">{$_('settings.subtitle')}</p>
		<AppearanceSettings />
		<MemorySettings />
		<LLMProviderSettings />
		<DangerZone />
	</div>
</Modal>