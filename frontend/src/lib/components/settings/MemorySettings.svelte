<script lang="ts">
	import { setMemoryEnabled } from '$lib/stores/preferences.svelte.ts';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { apiPy, api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { History, Brain } from '@lucide/svelte';

	let isClearing = $state(false);

	function handleMemoryToggle(event: Event) {
		const checked = (event.target as HTMLInputElement).checked;
		setMemoryEnabled(checked);
	}

	async function handleClearMemory() {
		toast.error($_('settings.clear_confirm'), {
			action: {
				label: $_('settings.clear_action'),
				onClick: async () => {
					isClearing = true;
					try {
						await apiPy.clearMemory();
						toast.success($_('settings.clear_success'));
					} catch (e: unknown) {
						if (e instanceof Error) {
							toast.error(e.message);
						} else {
							toast.error($_('settings.clear_failed'));
						}
					} finally {
						isClearing = false;
					}
				}
			},
			cancel: {
				label: $_('sidebar.cancel_action'),
				onClick: () => {}
			}
		});
	}

	async function handleClearAllSessions() {
		toast.error($_('sidebar.delete_confirm'), {
			action: {
				label: $_('sidebar.delete_action'),
				onClick: async () => {
					try {
						await api.delete('/sessions');
						const { clearUserSessions } = await import('$lib/stores/sessions.svelte.ts');
						clearUserSessions();
						toast.success($_('sidebar.session_deleted'));
					} catch (e: unknown) {
						if (e instanceof Error) {
							toast.error(e.message);
						} else {
							toast.error($_('errors.generic'));
						}
					}
				}
			},
			cancel: { label: $_('sidebar.cancel_action'), onClick: () => {} }
		});
	}
</script>

<div class="space-y-4">
	<div
		class="rounded-2xl border border-tertiary bg-secondary/30 p-4 transition-all hover:border-accent/40 hover:bg-secondary/50"
	>
		<div class="mb-4 flex items-start justify-between">
			<div class="max-w-[70%]">
				<div class="mb-1 text-sm font-medium text-text-primary">{$_('settings.cross_memory')}</div>
				<p class="text-xs leading-relaxed text-text-secondary">
					{$_('settings.cross_memory_hint')}
				</p>
			</div>
			<label class="relative inline-flex cursor-pointer items-center">
				<input
					type="checkbox"
					class="sr-only"
					checked={preferencesStore.memoryEnabled}
					onchange={handleMemoryToggle}
				/>
				<div
					class="h-6 w-11 rounded-full border border-tertiary/50 transition-colors duration-200 {preferencesStore.memoryEnabled
						? 'bg-accent'
						: 'bg-tertiary/50'}"
				>
					<div
						class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform duration-200 {preferencesStore.memoryEnabled
							? 'translate-x-5'
							: 'translate-x-0'}"
					></div>
				</div>
			</label>
		</div>

		<div class="flex flex-wrap gap-2 pt-2">
			<button
				class="flex items-center gap-2 rounded-xl bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-500 transition-all hover:bg-red-500 hover:text-white disabled:opacity-50"
				onclick={handleClearMemory}
				disabled={isClearing}
			>
				<Brain class="h-3.5 w-3.5" />
				{#if isClearing}{$_('settings.clearing')}
				{:else}{$_('settings.clear_memory')}{/if}
			</button>
			<button
				class="flex items-center gap-2 rounded-xl bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-500 transition-all hover:bg-red-500 hover:text-white"
				onclick={handleClearAllSessions}
			>
				<History class="h-3.5 w-3.5" />
				{$_('sidebar.delete_sessions')}
			</button>
		</div>
	</div>
</div>
