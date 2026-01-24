<script lang="ts">
	import { api, apiPy } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { slide } from 'svelte/transition';
	import {
		userSettingsStore,
		updateUserSettings,
		clearUserSettings
	} from '$lib/stores/user-settings.svelte.ts';
	import CustomSelect from '$lib/components/ui/CustomSelect.svelte';
	import type { ValidateApiKeyResponse, ListModelsResponse } from '$lib/types';
	import { Key, ShieldCheck, Save, Trash2, RefreshCcw, Cpu } from '@lucide/svelte';

	let selectedProvider = $state('');
	let apiKey = $state('');
	let selectedModel = $state('');
	let models = $state<string[]>([]);
	let isValidating = $state(false);
	let isLoading = $state(false);

	// Hardcoded provider list
	const providers = [
		{ value: 'ego', label: 'EGO' },
		{ value: 'gemini', label: 'Gemini' },
		{ value: 'anthropic', label: 'Claude' },
		{ value: 'openai', label: 'ChatGPT' },
		{ value: 'grok', label: 'Grok' }
	];

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
			// Validate the key first
			const response = (await apiPy.validateApiKey(
				selectedProvider,
				apiKey
			)) as ValidateApiKeyResponse;
			if (response.is_valid) {
				toast.success($_('settings.validate_success'));
				const modelsResponse = (await apiPy.listModels(
					selectedProvider,
					apiKey
				)) as ListModelsResponse;
				models = modelsResponse.models;
				if (models.length > 0) {
					selectedModel = models[0];
				}
				updateUserSettings({
					llm_provider: selectedProvider,
					llm_model: selectedModel,
					api_key: apiKey
				});
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
			updateUserSettings({
				llm_provider: selectedProvider,
				llm_model: selectedProvider === 'ego' ? '' : selectedModel,
				api_key: selectedProvider === 'ego' ? '' : apiKey
			});
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
			updateUserSettings({
				llm_provider: selectedProvider,
				llm_model: selectedModel || '',
				api_key: ''
			});
			toast.message($_('settings.provider_selected'));
		}
	}
</script>

<div class="space-y-6">
	<div
		class="rounded-2xl border border-tertiary bg-secondary/30 p-4 transition-all hover:border-accent/40 hover:bg-secondary/50"
	>
		<div class="mb-4">
			<label for="provider" class="mb-2 block text-sm font-medium text-text-primary"
				>{$_('settings.provider')}</label
			>
			<CustomSelect
				bind:value={selectedProvider}
				options={providers}
				on:change={handleProviderChange}
				placeholder={$_('settings.provider')}
			/>
			<p class="mt-2 text-xs text-text-secondary">{$_('settings.llm_provider_hint')}</p>
		</div>

		{#if selectedProvider !== 'ego'}
			<div class="space-y-4 pt-2" in:slide={{ duration: 300 }}>
				<div class="space-y-2">
					<label for="api_key" class="flex items-center gap-2 text-sm font-medium text-text-primary">
						<Key class="h-3.5 w-3.5 text-accent" />
						{$_('settings.api_key')}
					</label>
					<input
						type="password"
						id="api_key"
						bind:value={apiKey}
						class="w-full rounded-xl border border-tertiary bg-primary/50 p-3 text-sm transition-all focus:border-accent/60 focus:outline-none focus:ring-4 focus:ring-accent/10"
						placeholder={$_('settings.api_key_placeholder')}
					/>
				</div>

				<div class="space-y-2">
					<label for="model" class="flex items-center gap-2 text-sm font-medium text-text-primary">
						<Cpu class="h-3.5 w-3.5 text-accent" />
						{$_('settings.model')}
					</label>
					<CustomSelect
						bind:value={selectedModel}
						options={models.map((m) => ({ value: m, label: m }))}
						disabled={models.length === 0}
						placeholder={models.length === 0 ? $_('settings.validate_first') : $_('settings.model')}
					/>
				</div>

				<div class="rounded-xl bg-accent/5 p-3">
					<div class="mb-1 flex items-center gap-2 text-[11px] font-semibold text-accent">
						<ShieldCheck class="h-3 w-3" />
						{$_('settings.key_safety')}
					</div>
					<p class="text-[10px] italic text-text-secondary">{$_('settings.disclaimer')}</p>
				</div>

				<div class="flex flex-wrap items-center justify-between gap-3 pt-2">
					<button
						onclick={handleDelete}
						class="flex items-center gap-2 rounded-xl bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-500 transition-all hover:bg-red-500 hover:text-white disabled:opacity-50"
						disabled={isLoading || isValidating}
					>
						<Trash2 class="h-3.5 w-3.5" />
						{$_('settings.delete_button')}
					</button>

					<div class="flex gap-2">
						<button
							onclick={handleValidate}
							class="flex items-center gap-2 rounded-xl bg-accent/10 px-4 py-2 text-xs font-semibold text-accent transition-all hover:bg-accent hover:text-white disabled:opacity-50"
							disabled={isLoading || isValidating}
						>
							<RefreshCcw class="h-3.5 w-3.5 {isValidating ? 'animate-spin' : ''}" />
							{$_('settings.validate_button')}
						</button>
						<button
							onclick={handleSave}
							class="flex items-center gap-2 rounded-xl bg-accent px-4 py-2 text-xs font-semibold text-white transition-all hover:bg-accent-hover shadow-lg shadow-accent/20 disabled:opacity-50"
							disabled={isLoading ||
								isValidating ||
								(selectedProvider !== 'ego' && (!apiKey || !selectedModel))}
						>
							<Save class="h-3.5 w-3.5" />
							{#if isLoading}{$_('settings.saving')}{:else}{$_('settings.save_button')}{/if}
						</button>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>
