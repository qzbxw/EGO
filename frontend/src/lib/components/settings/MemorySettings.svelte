<script lang="ts">
	import { setMemoryEnabled } from '$lib/stores/preferences.svelte.ts';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import { apiPy, api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';

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

<div class="rounded-xl border border-tertiary p-3 bg-primary/60 md:col-span-2">
	<div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
		<div>
			<div class="text-sm font-semibold">{$_('settings.cross_memory')}</div>
			<div class="text-xs text-text-secondary">{$_('settings.cross_memory_hint')}</div>
		</div>
		<div class="flex items-center gap-3">
			<label class="inline-flex items-center cursor-pointer select-none">
				<input type="checkbox" class="sr-only" checked={preferencesStore.memoryEnabled} onchange={handleMemoryToggle} />
				<span class="w-11 h-6 rounded-full relative transition-colors duration-200 {preferencesStore.memoryEnabled ? 'bg-accent' : 'bg-tertiary'}">
					<span class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white transition-transform duration-200 {preferencesStore.memoryEnabled ? 'translate-x-5' : 'translate-x-0'}"></span>
				</span>
			</label>
			<button class="px-3 py-1.5 rounded-lg text-sm font-semibold bg-red-600 hover:bg-red-700 text-white disabled:opacity-60" onclick={handleClearMemory} disabled={isClearing}>
				{#if isClearing}{$_('settings.clearing')} {:else}{$_('settings.clear_memory')}{/if}
			</button>
			<button class="px-3 py-1.5 rounded-lg text-sm font-semibold bg-red-600 hover:bg-red-700 text-white" onclick={handleClearAllSessions}>
				{$_('sidebar.delete_action')} — {$_('sidebar.delete_sessions')}
			</button>
		</div>
	</div>
</div>