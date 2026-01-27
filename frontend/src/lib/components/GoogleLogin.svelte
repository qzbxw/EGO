<script lang="ts">
	import { env as publicEnv } from '$env/dynamic/public';
	import { api } from '$lib/api';
	import { goto } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import { setAuthData } from '$lib/stores/auth.svelte.ts';
	import { _ } from 'svelte-i18n';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	import type { AuthResponse, CredentialResponse } from '$lib/types';
	const GOOGLE_CLIENT_ID = publicEnv.PUBLIC_GOOGLE_CLIENT_ID || '';
	let isLoading = $state(false);
	async function handleGoogleCallback(response: CredentialResponse) {
		isLoading = true;
		try {
			const res = await api.post<AuthResponse>('/auth/google', { token: response.credential });
			if (res && res.access_token && res.refresh_token && res.user) {
				setAuthData(res.user, res.access_token, res.refresh_token);
				await goto('/chat', { replaceState: true });
			} else {
				throw new Error($_('errors.server_incomplete_login'));
			}
		} catch (error: unknown) {
			if (error instanceof Error) {
				toast.error(error.message);
			} else {
				toast.error($_('errors.login_failed'));
			}
		} finally {
			isLoading = false;
		}
	}
	function renderButton() {
		const buttonElement = document.getElementById('google-login-button');
		if (!buttonElement || !window.google || !window.google.accounts || !GOOGLE_CLIENT_ID) return;
		buttonElement.innerHTML = '';
		window.google.accounts.id.initialize({
			client_id: GOOGLE_CLIENT_ID,
			callback: handleGoogleCallback
		});
		window.google.accounts.id.renderButton(buttonElement, {
			theme: preferencesStore.theme === 'light' ? 'outline' : 'filled_black',
			size: 'large',
			type: 'standard',
			shape: 'pill',
			text: 'signin_with',
			logo_alignment: 'left'
		});
	}
	$effect(() => {
		const interval = setInterval(() => {
			if (window.google && window.google.accounts) {
				clearInterval(interval);
				renderButton();
			}
		}, 100);
		return () => clearInterval(interval);
	});
	$effect(() => {
		try {
			renderButton();
		} catch (e) {
			console.debug('Google login button render failed:', e);
		}
	});
</script>

<div class="relative flex min-h-[44px] w-full items-center justify-center">
	{#if isLoading}
		<div
			class="absolute inset-0 flex animate-pulse items-center justify-center text-text-secondary"
		>
			{$_('login.loading_button')}
		</div>
	{/if}
	<div id="google-login-button" class:opacity-0={isLoading}></div>
</div>
