<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { Eye, EyeOff, User, Lock, Sun, Moon } from '@lucide/svelte';
	import GoogleLogin from '$lib/components/GoogleLogin.svelte';
	import { _ } from 'svelte-i18n';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import { setAuthData } from '$lib/stores/auth.svelte.ts';
	import type { AuthResponse } from '$lib/types';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
	let username = $state('');
	let password = $state('');
	let isLoading = $state(false);
	let showPassword = $state(false);
	function toggleTheme() {
		setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
	}
    async function handleLogin(event: SubmitEvent) {
		event.preventDefault();
		if (isLoading || !username || !password) return;
		isLoading = true;
		try {
			const response = await api.post<AuthResponse>('/auth/login', { username, password });
			if (response && response.access_token && response.refresh_token && response.user) {
				setAuthData(response.user, response.access_token, response.refresh_token);
				goto('/authenticating?redirect=' + encodeURIComponent('/chat'), { replaceState: true });
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
</script>
<svelte:head>
	<title>{$_('login.login_button')} - EGO</title>
</svelte:head>
<div class="relative w-full min-h-screen flex flex-col">
	<div class="absolute top-4 right-4 z-10 flex items-center gap-2">
		<LanguageSwitcher />
		<button onclick={toggleTheme} class="p-2 rounded-lg bg-secondary hover:bg-tertiary transition-colors" aria-label="Toggle theme">
			{#if preferencesStore.theme === 'light'}<Moon class="w-4 h-4" />{:else}<Sun class="w-4 h-4" />{/if}
		</button>
		<a href="/knowego" class="p-2 rounded-lg bg-secondary hover:bg-tertiary transition-colors" title="About">?</a>
	</div>
    <div class="flex-1 flex items-center justify-center p-3 sm:p-4">
        <div class="animated-border-card w-full max-w-md animate-fade-in-up">
            <div class="w-full bg-secondary/95 p-5 sm:p-6 md:p-8 rounded-xl">
				<div class="text-center mb-6 md:mb-8">
					<img src={getAppLogo(preferencesStore.theme)} alt="EGO Logo" class="w-12 h-12 mx-auto mb-2" />
					<h1 class="text-2xl md:text-3xl font-bold text-text-primary">{$_('login.welcome')}</h1>
					<p class="text-text-secondary">{$_('login.prompt')}</p>
				</div>
				<form onsubmit={handleLogin} class="space-y-4 md:space-y-6">
					<div>
						<label for="username" class="block text-sm font-medium text-text_secondary mb-2">{$_('login.username')}</label>
						<div class="auth-input-wrapper">
							<input id="username" type="text" bind:value={username} required class="auth-input" placeholder={$_('login.username_placeholder')} />
							<User class="auth-input-icon w-5 h-5" />
						</div>
					</div>
					<div>
						<label for="password" class="block text-sm font-medium text-text_secondary mb-2">{$_('login.password')}</label>
						<div class="auth-input-wrapper relative">
							<input id="password" type={showPassword ? 'text' : 'password'} bind:value={password} required class="auth-input pr-10" placeholder="••••••••" />
							<Lock class="auth-input-icon w-5 h-5" />
							<button type="button" onclick={() => (showPassword = !showPassword)} class="absolute top-1/2 right-3 -translate-y-1/2 text-text-secondary hover:text-text-primary p-1 z-10" aria-label={$_('login.toggle_password')}>
								{#if showPassword} <EyeOff class="w-5 h-5" /> {:else} <Eye class="w-5 h-5" /> {/if}
							</button>
						</div>
					</div>
					<button type="submit" disabled={isLoading} class="w-full py-3 btn-gradient disabled:opacity-50 disabled:cursor-not-allowed hover:scale-105 active:scale-100">
						{isLoading ? $_('login.loading_button') : $_('login.login_button')}
					</button>
				</form>
				<div class="flex items-center my-4 md:my-6">
					<div class="flex-grow border-t border-tertiary"></div>
					<span class="flex-shrink mx-4 text-text-secondary text-xs uppercase">{$_('login.or')}</span>
					<div class="flex-grow border-t border-tertiary"></div>
				</div>
				<GoogleLogin />
				<div class="mt-6 text-center">
					<p class="text-sm text-text-secondary">
						{$_('login.no_account')}
						<a href="/register" class="font-semibold text-accent hover:text-accent-hover hover:underline">{$_('login.register')}</a>
					</p>
				</div>
				<div class="mt-2 text-[11px] text-text-secondary">
					{$_('auth.agree_prefix')} <a href="/terms" target="_blank" class="underline hover:opacity-80">{$_('legal.terms_title')}</a> {$_('auth.and')} <a href="/privacy" target="_blank" class="underline hover:opacity-80">{$_('legal.privacy_title')}</a>.
				</div>
			</div>
		</div>
	</div>
</div>