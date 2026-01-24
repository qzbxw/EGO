<script lang="ts">
	import { api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { UserMinus } from '@lucide/svelte';

	async function handleDeleteAccount() {
		toast.error($_('settings.delete_account_confirm'), {
			action: {
				label: $_('settings.delete_action'),
				onClick: async () => {
					try {
						await api.delete('/user/account');
						const { logout } = await import('$lib/stores/auth.svelte.ts');
						logout();
					} catch (e: unknown) {
						if (e instanceof Error) {
							toast.error(e.message);
						} else {
							toast.error($_('settings.delete_account_failed'));
						}
					}
				}
			},
			cancel: { label: $_('sidebar.cancel_action'), onClick: () => {} }
		});
	}
</script>

<div
	class="overflow-hidden rounded-2xl border border-red-500/20 bg-red-500/5 transition-all hover:border-red-500/40"
>
	<div class="flex flex-col items-start justify-between gap-4 p-4 sm:flex-row sm:items-center">
		<div class="space-y-1">
			<div class="text-sm font-semibold text-red-500">{$_('settings.delete_account_button')}</div>
			<p class="text-xs leading-relaxed text-text-secondary">
				{$_('settings.delete_account_desc')}
			</p>
		</div>
		<button
			class="flex items-center gap-2 rounded-xl bg-red-500 px-4 py-2 text-xs font-bold text-white shadow-lg shadow-red-500/20 transition-all hover:bg-red-600 hover:shadow-red-500/40 active:scale-[0.98]"
			onclick={handleDeleteAccount}
		>
			<UserMinus class="h-3.5 w-3.5" />
			{$_('settings.delete_account_button')}
		</button>
	</div>
</div>
