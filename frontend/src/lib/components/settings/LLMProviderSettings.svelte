<script lang="ts">
	import { api, apiPy } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { userSettingsStore, updateUserSettings, clearUserSettings } from '$lib/stores/user-settings.svelte.ts';
	import type { ValidateApiKeyResponse, ListModelsResponse } from '$lib/types';
	let selectedProvider = $state('');
	let apiKey = $state('');
	let selectedModel = $state('');
	let models = $state<string[]>([]);
	let isValidating = $state(false);
	let isLoading = $state(false);
	$effect(() => {
		const s = userSettingsStore.settings;
		if (s) {
			const storeProvider = s.llm_provider || 'ego';
			const storeModel = s.llm_model || '';
			const initialSync = selectedProvider === '';
			if (initialSync) {
				selectedProvider = storeProvider;
				selectedModel = storeModel;
				if (storeProvider === 'ego') {
					apiKey = '';
					models = [];
				} else {
					apiKey = s.api_key || '';
				}
			}
		} else {
			if (selectedProvider === '') {
				selectedProvider = 'ego';
				selectedModel = '';
				apiKey = '';
				models = [];
			}
		}
	});
	async function handleValidate() {
		if (selectedProvider === 'ego') {
			toast.error($_('settings.error_provider_ego_no_validation'));
			return;
		}
		if (!apiKey) {
			toast.error($_('settings.error_enter_api_key'));
			return;
		}
		isValidating = true;
		try {
			const response = await apiPy.validateApiKey(selectedProvider, apiKey);
			if (response.is_valid) {
				toast.success($_('settings.validate_success'));
				const modelsResponse = await apiPy.listModels(selectedProvider, apiKey);
				models = modelsResponse.models;
				if (models.length > 0) {
					selectedModel = models[0];
				}
				updateUserSettings({ llm_provider: selectedProvider, llm_model: selectedModel, api_key: apiKey });
			} else {
				toast.error($_('settings.validate_error'));
			}
		} catch (e: unknown) {
			toast.error($_('settings.validate_failed'));
		} finally {
			isValidating = false;
		}
	}
	async function handleSave() {
		if (selectedProvider !== 'ego' && (!apiKey || !selectedModel)) {
			toast.error($_('settings.warning_validate_first_full'));
			return;
		}
		isLoading = true;
		try {
			await api.updateUserSettings({ 
				llm_provider: selectedProvider,
				api_key: selectedProvider === 'ego' ? '' : apiKey,
				llm_model: selectedProvider === 'ego' ? '' : selectedModel
			});
			updateUserSettings({ llm_provider: selectedProvider, llm_model: selectedProvider === 'ego' ? '' : selectedModel, api_key: selectedProvider === 'ego' ? '' : apiKey });
			toast.success($_('settings.save_success'));
		} catch (e: unknown) {
			toast.error($_('settings.save_failed'));
		} finally {
			isLoading = false;
		}
	}
	async function handleDelete() {
		toast.error($_('settings.delete_confirm'), {
			action: {
				label: $_('settings.delete_action'),
				onClick: async () => {
					isLoading = true;
					try {
						await api.deleteUserSettings();
						toast.success($_('settings.delete_success'));
						selectedProvider = 'ego';
						apiKey = '';
						selectedModel = '';
						models = [];
						clearUserSettings();
					} catch (e: unknown) {
						if (e instanceof Error) {
							toast.error(e.message);
						} else {
							toast.error($_('settings.delete_failed'));
						}
					} finally {
						isLoading = false;
					}
				}
			},
			cancel: {
				label: $_('sidebar.cancel_action'),
				onClick: () => {}
			}
		});
	}
	async function handleProviderChange() {
		if (selectedProvider === 'ego' || selectedProvider === '') {
			apiKey = '';
			models = [];
			isLoading = true;
			try {
				await api.updateUserSettings({ llm_provider: 'ego', api_key: '', llm_model: '' });
				updateUserSettings({ llm_provider: 'ego', llm_model: '', api_key: '' });
				toast.success($_('settings.save_success'));
			} catch (e: unknown) {
				toast.error($_('settings.save_failed'));
			} finally {
				isLoading = false;
			}
		} else {
			models = [];
			apiKey = '';
			updateUserSettings({ llm_provider: selectedProvider, llm_model: selectedModel || '', api_key: '' });
			toast.message($_('settings.provider_selected'));
		}
	}
</script>
<div class="space-y-4 max-w-3xl">
	<div>
		<div class="text-sm font-semibold mb-1">{$_('settings.llm_provider')}</div>
		<div class="text-xs text-text-secondary mb-2">{$_('settings.llm_provider_hint')}</div>
		<label for="provider" class="sr-only">{$_('settings.provider')}</label>
		<select id="provider" bind:value={selectedProvider} class="w-full bg-primary border-2 border-tertiary rounded-lg p-2 focus:ring-2 focus:ring-accent/60 focus:outline-none" onchange={() => handleProviderChange()}>
			<option value="ego">{$_('settings.ego_default')}</option>
			<option value="openai">OpenAI</option>
			<option value="anthropic">Anthropic</option>
			<option value="gemini">Google Gemini</option>
			<option value="grok">Grok (xAI)</option>
			<option value="openrouter_free">OpenRouter (Free)</option>
			<option value="openrouter_billing">OpenRouter (Billing)</option>
		</select>
	</div>
	{#if selectedProvider !== 'ego'}
		<div>
			<label for="api_key" class="block text-sm font-semibold text-text-primary mb-1">{$_('settings.api_key')}</label>
			<input type="password" id="api_key" bind:value={apiKey} class="w-full bg-primary border-2 border-tertiary rounded-lg p-2 focus:ring-2 focus:ring-accent/60 focus:outline-none" placeholder={$_('settings.api_key_placeholder')}>
		</div>
		<div>
			<label for="model" class="block text-sm font-semibold text-text-primary mb-1">{$_('settings.model')}</label>
			<select id="model" bind:value={selectedModel} class="w-full bg-primary border-2 border-tertiary rounded-lg p-2 focus:ring-2 focus:ring-accent/60 focus:outline-none disabled:opacity-60" disabled={models.length === 0}>
				{#if models.length > 0}
					{#each models as model}
						<option value={model}>{model}</option>
					{/each}
				{:else}
					<option value="">{$_('settings.validate_first')}</option>
				{/if}
			</select>
		</div>
		<div class="text-xs text-text-secondary italic">{$_('settings.disclaimer')}</div>
		<div class="text-[11px] text-text-secondary mt-1">{$_('settings.key_safety')}</div>
		{#if selectedProvider !== 'ego' && (!apiKey || !selectedModel)}
			<div class="text-xs text-yellow-600 mt-2">{$_('settings.warning_validate_first_full')}</div>
		{/if}
		<div class="flex flex-col sm:flex-row gap-3 justify-between items-stretch sm:items-center pt-4">
			<button onclick={handleDelete} class="px-4 py-2 rounded-lg text-white text-sm font-semibold bg-red-600 hover:bg-red-700 active:scale-[0.99] transition-all disabled:opacity-60 disabled:cursor-not-allowed" disabled={isLoading || isValidating}>{$_('settings.delete_button')}</button>
			<div class="flex gap-2 w-full sm:w-auto">
				<button onclick={handleValidate} class="px-4 py-2 rounded-lg text-sm font-semibold active:scale-[0.99] transition-all disabled:opacity-60 disabled:cursor-not-allowed bg-accent text-white hover:bg-accent-hover" disabled={isLoading || isValidating}>
					{#if isValidating}{$_('settings.loading')}{:else}{$_('settings.validate_button')}{/if}
				</button>
				<button onclick={handleSave} class="px-4 py-2 rounded-lg text-white text-sm font-semibold bg-accent hover:bg-accent-hover active:scale-[0.99] transition-all disabled:opacity-60 disabled:cursor-not-allowed" disabled={isLoading || isValidating || (selectedProvider !== 'ego' && (!apiKey || !selectedModel))}>
					{#if isLoading}{$_('settings.saving')}{:else}{$_('settings.save_button')}{/if}
				</button>
			</div>
		</div>
	{/if}
</div>