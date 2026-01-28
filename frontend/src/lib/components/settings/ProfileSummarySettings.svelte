<script lang="ts">
	import { api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { Eye, EyeOff, Save, Trash2 } from '@lucide/svelte';

	let profileSummary = $state('');
	let isLoading = $state(false);
	let isSaving = $state(false);
	let isExpanded = $state(false);
	let originalSummary = '';

	async function loadProfileSummary() {
		isLoading = true;
		try {
			const response = await api.get<{ profile_summary: string }>('/user/profile-summary');
			profileSummary = response.profile_summary || '';
			originalSummary = profileSummary;
		} catch (e: unknown) {
			if (e instanceof Error) {
				toast.error(e.message);
			} else {
				toast.error($_('errors.generic'));
			}
		} finally {
			isLoading = false;
		}
	}

	async function handleSave() {
		isSaving = true;
		try {
			await api.put('/user/profile-summary', { profile_summary: profileSummary });
			originalSummary = profileSummary;
			toast.success($_('settings.profile_summary_saved'));
		} catch (e: unknown) {
			if (e instanceof Error) {
				toast.error(e.message);
			} else {
				toast.error($_('settings.profile_summary_save_failed'));
			}
		} finally {
			isSaving = false;
		}
	}

	async function handleClear() {
		toast.error($_('settings.profile_summary_clear_confirm'), {
			action: {
				label: $_('settings.clear_action'),
				onClick: async () => {
					try {
						await api.delete('/user/profile-summary');
						profileSummary = '';
						originalSummary = '';
						toast.success($_('settings.profile_summary_cleared'));
					} catch (e: unknown) {
						if (e instanceof Error) {
							toast.error(e.message);
						} else {
							toast.error($_('settings.profile_summary_clear_failed'));
						}
					}
				}
			},
			cancel: {
				label: $_('sidebar.cancel_action'),
				onClick: () => {}
			}
		});
	}

	function toggleExpanded() {
		if (!isExpanded) {
			loadProfileSummary();
		}
		isExpanded = !isExpanded;
	}

	$effect(() => {
		if (isExpanded) {
			loadProfileSummary();
		}
	});

	const hasChanges = $derived(profileSummary !== originalSummary);
</script>

<div class="space-y-4">
	<div
		class="rounded-2xl border border-tertiary bg-secondary/30 p-4 transition-all hover:border-accent/40 hover:bg-secondary/50"
	>
		<button
			type="button"
			class="mb-2 flex w-full items-start justify-between"
			onclick={toggleExpanded}
		>
			<div class="max-w-[80%] text-left">
				<div class="mb-1 text-sm font-medium text-text-primary">
					{$_('settings.profile_summary')}
				</div>
				<p class="text-xs leading-relaxed text-text-secondary">
					{$_('settings.profile_summary_hint')}
				</p>
			</div>
			<div class="flex items-center gap-2">
				{#if isExpanded}
					<EyeOff class="h-4 w-4 text-text-secondary" />
				{:else}
					<Eye class="h-4 w-4 text-text-secondary" />
				{/if}
			</div>
		</button>

		{#if isExpanded}
			<div class="mt-4 space-y-3">
				{#if isLoading}
					<div class="flex items-center justify-center py-8">
						<div
							class="h-6 w-6 animate-spin rounded-full border-2 border-accent border-t-transparent"
						></div>
					</div>
				{:else}
					<textarea
						class="w-full rounded-xl border border-tertiary bg-primary/50 px-4 py-3 text-sm text-text-primary placeholder-text-secondary/50 transition-all focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20"
						rows="6"
						placeholder={$_('settings.profile_summary_placeholder')}
						bind:value={profileSummary}
					></textarea>

					<div class="flex flex-wrap gap-2">
						<button
							type="button"
							class="flex items-center gap-2 rounded-xl bg-accent/10 px-4 py-2 text-xs font-semibold text-accent transition-all hover:bg-accent hover:text-white disabled:opacity-50"
							onclick={handleSave}
							disabled={isSaving || !hasChanges}
						>
							<Save class="h-3.5 w-3.5" />
							{#if isSaving}{$_('settings.saving')}
							{:else}{$_('settings.save')}{/if}
						</button>
						<button
							type="button"
							class="flex items-center gap-2 rounded-xl bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-500 transition-all hover:bg-red-500 hover:text-white disabled:opacity-50"
							onclick={handleClear}
							disabled={!profileSummary}
						>
							<Trash2 class="h-3.5 w-3.5" />
							{$_('settings.clear')}
						</button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
