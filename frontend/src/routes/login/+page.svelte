<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { Eye, EyeOff, User, Lock, Sun, Moon, ArrowRight } from '@lucide/svelte';
	import GoogleLogin from '$lib/components/GoogleLogin.svelte';
	import { _ } from 'svelte-i18n';
	import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
	import { setAuthData } from '$lib/stores/auth.svelte.ts';
	import type { AuthResponse } from '$lib/types';
	import { getAppLogo } from '$lib/config';
	import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';

	let username = $state('');
	let password = $state('');
	let isLoading = $state(false);
	let showPassword = $state(false);

	let errors = $state({
		username: '',
		password: ''
	});

	function toggleTheme() {
		setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
	}

	function validateForm() {
		let isValid = true;
		errors = {
			username: '',
			password: ''
		};

		if (!username.trim()) {
			errors.username = $_('errors.field_required');
			isValid = false;
		}

		if (!password) {
			errors.password = $_('errors.field_required');
			isValid = false;
		}

		return isValid;
	}

	async function handleLogin(event: SubmitEvent) {
		event.preventDefault();
		if (isLoading) return;

		if (!validateForm()) {
			return;
		}

		isLoading = true;
		try {
			const response = await api.post<AuthResponse>('/auth/login', { username, password });
			if (response && response.access_token && response.refresh_token && response.user) {
				setAuthData(response.user, response.access_token, response.refresh_token);
				void goto('/chat', { replaceState: true });
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

<div
	class="relative flex min-h-screen w-full flex-col overflow-hidden bg-primary transition-colors duration-700 ease-in-out selection:bg-accent selection:text-white"
>
	<!-- Ambient Background -->
	<div class="pointer-events-none absolute inset-0 z-0 overflow-hidden">
		<div
			class="bg-circle animate-pulse-slow absolute -left-[10%] -top-[10%] h-[60vh] w-[60vh] opacity-40 blur-[80px] transition-all duration-1000 md:blur-[120px]"
		></div>
		<div
			class="bg-circle secondary animate-pulse-slow absolute -bottom-[10%] -right-[10%] h-[60vh] w-[60vh] opacity-40 blur-[80px] transition-all duration-1000 md:blur-[120px]"
			style="animation-delay: 2s;"
		></div>
	</div>

	<!-- Top Controls -->
	<div
		in:fly={{ y: -20, duration: 800, delay: 200, easing: cubicOut }}
		class="absolute right-6 top-6 z-50 flex items-center gap-3"
	>
		<LanguageSwitcher />
		<button
			onclick={toggleTheme}
			class="group relative rounded-full bg-secondary/30 p-2.5 text-text-secondary backdrop-blur-md transition-all duration-300 hover:bg-secondary/60 hover:text-text-primary hover:shadow-[0_0_15px_rgba(0,0,0,0.1)] active:scale-90"
			aria-label="Toggle theme"
		>
			<div
				class="ease-spring relative z-10 transition-transform duration-500 group-hover:rotate-[360deg]"
			>
				{#if preferencesStore.theme === 'light'}
					<Moon class="h-5 w-5" />
				{:else}
					<Sun class="h-5 w-5" />
				{/if}
			</div>
		</button>
	</div>

	<!-- Main Content -->
	<div class="z-10 flex flex-1 items-center justify-center p-4 sm:p-6">
		<div
			in:fly={{ y: 30, duration: 1000, delay: 100, easing: cubicOut }}
			class="w-full max-w-[400px]"
		>
			<!-- Glass Card -->
			<div
				class="relative overflow-hidden rounded-[2rem] border border-white/10 bg-secondary/30 p-8 shadow-2xl backdrop-blur-2xl transition-all duration-500 hover:bg-secondary/40 hover:shadow-[0_0_40px_rgba(var(--color-accent-rgb),0.1)] md:p-10"
			>
				<!-- Header -->
				<div class="mb-10 text-center">
					<div in:fly={{ y: 20, duration: 800, delay: 200, easing: cubicOut }}>
						<button
							class="mx-auto mb-6 block cursor-default transition-transform duration-500 hover:rotate-6 hover:scale-110 active:rotate-12 active:scale-90"
						>
							<img
								src={getAppLogo(preferencesStore.theme)}
								alt="EGO Logo"
								class="h-16 w-16 drop-shadow-2xl"
							/>
						</button>
					</div>
					<h1
						in:fly={{ y: 20, duration: 800, delay: 300, easing: cubicOut }}
						class="mb-2 text-3xl font-bold tracking-tight text-text-primary"
					>
						{$_('login.welcome')}
					</h1>
					<p
						in:fly={{ y: 20, duration: 800, delay: 400, easing: cubicOut }}
						class="text-sm font-medium text-text-secondary/80"
					>
						{$_('login.prompt')}
					</p>
				</div>

				<!-- Form -->
				<form onsubmit={handleLogin} class="space-y-6" novalidate>
					<div in:fly={{ y: 20, duration: 800, delay: 500, easing: cubicOut }} class="group/input">
						<label
							for="username"
							class="mb-2 block text-xs font-bold uppercase tracking-widest text-text-secondary/60 transition-colors group-focus-within/input:text-accent"
						>
							{$_('login.username')}
						</label>
						<div
							class="relative transition-all duration-300 ease-out focus-within:-translate-y-1 focus-within:scale-[1.02]"
						>
							<div
								class="absolute inset-y-0 left-0 flex items-center pl-4 text-text-secondary transition-all duration-300 group-focus-within/input:-rotate-12 group-focus-within/input:scale-110 group-focus-within/input:text-accent"
							>
								<User class="h-5 w-5" />
							</div>
							<input
								id="username"
								type="text"
								bind:value={username}
								class="w-full rounded-2xl border border-transparent bg-tertiary/40 py-4 pl-12 pr-4 text-sm font-medium text-text-primary outline-none transition-all duration-300 placeholder:text-text-secondary/40 focus:border-accent/30 focus:bg-tertiary/70 focus:shadow-[0_4px_20px_rgba(0,0,0,0.1)] {errors.username
									? '!border-red-500/50 !bg-red-500/5 !shadow-[0_0_15px_rgba(239,68,68,0.2)]'
									: ''}"
								placeholder={$_('login.username_placeholder')}
								oninput={() => (errors.username = '')}
							/>
						</div>
						{#if errors.username}
							<p
								class="mt-2 animate-pulse pl-1 text-xs font-bold text-red-500"
								transition:fly={{ y: -5, duration: 200 }}
							>
								{errors.username}
							</p>
						{/if}
					</div>

					<div in:fly={{ y: 20, duration: 800, delay: 600, easing: cubicOut }} class="group/input">
						<label
							for="password"
							class="mb-2 block text-xs font-bold uppercase tracking-widest text-text-secondary/60 transition-colors group-focus-within/input:text-accent"
						>
							{$_('login.password')}
						</label>
						<div
							class="relative transition-all duration-300 ease-out focus-within:-translate-y-1 focus-within:scale-[1.02]"
						>
							<div
								class="absolute inset-y-0 left-0 flex items-center pl-4 text-text-secondary transition-all duration-300 group-focus-within/input:-rotate-12 group-focus-within/input:scale-110 group-focus-within/input:text-accent"
							>
								<Lock class="h-5 w-5" />
							</div>
							<input
								id="password"
								type={showPassword ? 'text' : 'password'}
								bind:value={password}
								class="w-full rounded-2xl border border-transparent bg-tertiary/40 py-4 pl-12 pr-12 text-sm font-medium text-text-primary outline-none transition-all duration-300 placeholder:text-text-secondary/40 focus:border-accent/30 focus:bg-tertiary/70 focus:shadow-[0_4px_20px_rgba(0,0,0,0.1)] {errors.password
									? '!border-red-500/50 !bg-red-500/5 !shadow-[0_0_15px_rgba(239,68,68,0.2)]'
									: ''}"
								placeholder="••••••••"
								oninput={() => (errors.password = '')}
							/>
							<button
								type="button"
								onclick={() => (showPassword = !showPassword)}
								class="absolute inset-y-0 right-0 flex items-center pr-4 text-text-secondary transition-all duration-300 hover:scale-110 hover:text-text-primary active:scale-90"
							>
								{#if showPassword}
									<EyeOff class="h-5 w-5" />
								{:else}
									<Eye class="h-5 w-5" />
								{/if}
							</button>
						</div>
						{#if errors.password}
							<p
								class="mt-2 animate-pulse pl-1 text-xs font-bold text-red-500"
								transition:fly={{ y: -5, duration: 200 }}
							>
								{errors.password}
							</p>
						{/if}
					</div>

					<div in:fly={{ y: 20, duration: 800, delay: 700, easing: cubicOut }} class="pt-4">
						<button
							type="submit"
							disabled={isLoading}
							class="group relative w-full overflow-hidden rounded-2xl bg-gradient-to-r from-accent to-accent-hover py-4 text-sm font-bold text-white shadow-[0_4px_15px_rgba(var(--color-accent-rgb),0.3)] transition-all duration-300 ease-out hover:-translate-y-1 hover:shadow-[0_8px_30px_rgba(var(--color-accent-rgb),0.5)] active:translate-y-0 active:scale-[0.96] active:shadow-inner disabled:transform-none disabled:cursor-not-allowed disabled:opacity-70"
						>
							<div
								class="absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/25 to-transparent transition-transform duration-1000 ease-in-out group-hover:translate-x-full"
							></div>
							<span class="relative flex items-center justify-center gap-3">
								{#if isLoading}
									<div
										class="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white"
									></div>
									{$_('login.loading_button')}
								{:else}
									{$_('login.login_button')}
									<ArrowRight
										class="ease-spring h-5 w-5 transition-transform duration-300 group-hover:translate-x-2"
									/>
								{/if}
							</span>
						</button>
					</div>
				</form>

				<!-- Divider -->
				<div
					in:fly={{ y: 20, duration: 800, delay: 800, easing: cubicOut }}
					class="my-8 flex items-center gap-4"
				>
					<div
						class="h-px flex-1 bg-gradient-to-r from-transparent via-tertiary to-transparent opacity-50"
					></div>
					<span class="text-[10px] font-bold uppercase tracking-widest text-text-secondary/40"
						>{$_('login.or')}</span
					>
					<div
						class="h-px flex-1 bg-gradient-to-r from-transparent via-tertiary to-transparent opacity-50"
					></div>
				</div>

				<!-- Social Login -->
				<div
					in:fly={{ y: 20, duration: 800, delay: 900, easing: cubicOut }}
					class="transition-transform duration-300 hover:scale-[1.02] active:scale-[0.98]"
				>
					<GoogleLogin />
				</div>

				<!-- Footer -->
				<div
					in:fly={{ y: 20, duration: 800, delay: 1000, easing: cubicOut }}
					class="mt-10 text-center"
				>
					<p class="text-sm font-medium text-text-secondary">
						{$_('login.no_account')}
						<a
							href="/register"
							class="group relative ml-1 inline-block font-bold text-accent transition-colors hover:text-accent-hover"
						>
							<span class="relative z-10">{$_('login.register')}</span>
							<span
								class="absolute bottom-0 left-0 h-[2px] w-0 bg-accent transition-all duration-300 ease-out group-hover:w-full"
							></span>
						</a>
					</p>

					<div
						class="mt-6 flex justify-center gap-4 text-[11px] font-medium text-text-secondary/50"
					>
						<div class="flex items-center gap-1">
							<span>{$_('auth.agree_prefix')}</span>
							<a
								href="/terms"
								target="_blank"
								class="decoration-text-primary/30 underline-offset-2 transition-colors hover:text-text-primary hover:underline"
								>{$_('legal.terms_title')}</a
							>
						</div>
						<div class="flex items-center gap-1">
							<span>{$_('auth.and')}</span>
							<a
								href="/privacy"
								target="_blank"
								class="decoration-text-primary/30 underline-offset-2 transition-colors hover:text-text-primary hover:underline"
								>{$_('legal.privacy_title')}</a
							>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	/* Custom easing for bouncy interactions */
	:global(.ease-spring) {
		transition-timing-function: cubic-bezier(0.175, 0.885, 0.32, 1.275);
	}
</style>
