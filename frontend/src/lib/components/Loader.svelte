<script lang="ts">
	import { Loader2, Zap, Brain, Sparkles } from '@lucide/svelte';
	interface Props {
		size?: 'sm' | 'md' | 'lg' | 'xl';
		variant?: 'spinner' | 'pulse' | 'dots' | 'brain' | 'sparkle';
		text?: string;
		color?: 'primary' | 'accent' | 'secondary';
	}
	let { size = 'md', variant = 'spinner', text = '', color = 'accent' }: Props = $props();
	const sizeClasses = {
		sm: 'w-4 h-4',
		md: 'w-6 h-6',
		lg: 'w-8 h-8',
		xl: 'w-12 h-12'
	};
	const colorClasses = {
		primary: 'text-text-primary',
		accent: 'text-accent',
		secondary: 'text-text-secondary'
	};
</script>

<div class="flex items-center gap-2">
	{#if variant === 'spinner'}
		<Loader2 class="{sizeClasses[size]} {colorClasses[color]} animate-spin" />
	{:else if variant === 'pulse'}
		<div
			class="{sizeClasses[size]} {colorClasses[color]} animate-pulse rounded-full bg-current"
		></div>
	{:else if variant === 'dots'}
		<div class="flex gap-1">
			<div
				class="h-2 w-2 {colorClasses[color]} animate-bounce rounded-full bg-current"
				style="animation-delay: 0ms;"
			></div>
			<div
				class="h-2 w-2 {colorClasses[color]} animate-bounce rounded-full bg-current"
				style="animation-delay: 150ms;"
			></div>
			<div
				class="h-2 w-2 {colorClasses[color]} animate-bounce rounded-full bg-current"
				style="animation-delay: 300ms;"
			></div>
		</div>
	{:else if variant === 'brain'}
		<Brain class="{sizeClasses[size]} {colorClasses[color]} animate-pulse" />
	{:else if variant === 'sparkle'}
		<Sparkles
			class="{sizeClasses[size]} {colorClasses[color]} animate-spin"
			style="animation-duration: 2s;"
		/>
	{/if}
	{#if text}
		<span class="text-sm {colorClasses[color]} animate-pulse">{text}</span>
	{/if}
</div>

<style>
	@keyframes bounce {
		0%,
		80%,
		100% {
			transform: scale(0);
		}
		40% {
			transform: scale(1);
		}
	}
	.animate-bounce {
		animation: bounce 1.4s infinite ease-in-out both;
	}
</style>
