<script lang="ts">
	import Loader from './Loader.svelte';
	import type { Snippet } from 'svelte';
	interface Props {
		loading?: boolean;
		disabled?: boolean;
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md' | 'lg';
		onclick?: () => void | Promise<void>;
		type?: 'button' | 'submit';
		class?: string;
		style?: string;
		loadingText?: string;
		loaderVariant?: 'spinner' | 'pulse' | 'dots' | 'brain' | 'sparkle';
		children?: Snippet;
		loading_content?: Snippet;
	}
	let {
		loading = false,
		disabled = false,
		variant = 'primary',
		size = 'md',
		onclick,
		type = 'button',
		class: className = '',
		style = '',
		loadingText = '',
		loaderVariant = 'spinner',
		children,
		loading_content
	}: Props = $props();
	let isLoading = $state(loading);
	const variantClasses = {
		primary:
			'bg-accent hover:bg-accent-hover text-white shadow-lg shadow-accent/25 hover:shadow-xl hover:shadow-accent/30',
		secondary: 'bg-secondary hover:bg-tertiary text-text-primary border border-border',
		ghost: 'hover:bg-secondary text-text-primary',
		danger: 'bg-red-500 hover:bg-red-600 text-white shadow-lg shadow-red-500/25'
	};
	const sizeClasses = {
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2 text-base',
		lg: 'px-6 py-3 text-lg'
	};
	async function handleClick() {
		if (isLoading || disabled || !onclick) return;
		try {
			isLoading = true;
			await onclick();
		} catch (error) {
			console.error('Button action failed:', error);
		} finally {
			isLoading = false;
		}
	}
</script>

<button
	{type}
	{style}
	onclick={handleClick}
	disabled={disabled || isLoading}
	class="
    relative inline-flex items-center justify-center gap-2 rounded-xl font-semibold
    transition-all duration-300 hover:scale-105 active:scale-95
    disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:scale-100
    {variantClasses[variant]} {sizeClasses[size]} {className}
  "
>
	{#if isLoading}
		<Loader variant={loaderVariant} size="sm" color="primary" />
		{#if loadingText}
			<span>{loadingText}</span>
		{:else if loading_content}
			{@render loading_content()}
		{:else if children}
			{@render children()}
		{/if}
	{:else if children}
		{@render children()}
	{/if}
</button>

<style>
	button:disabled {
		pointer-events: none;
	}
</style>
