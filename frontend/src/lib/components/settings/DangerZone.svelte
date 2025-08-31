<script lang="ts">
	import { api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
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
<div class="rounded-xl danger-zone p-3 mt-6">
	<div class="text-sm font-semibold danger-zone-title mb-2">{$_('settings.danger_zone')}</div>
	<div class="flex items-center justify-between gap-3">
		<div class="text-xs danger-zone-desc">{$_('settings.delete_account_desc')}</div>
		<button class="px-3 py-1.5 rounded-lg text-sm font-semibold danger-zone-btn" onclick={handleDeleteAccount}>{$_('settings.delete_account_button')}</button>
	</div>
</div>