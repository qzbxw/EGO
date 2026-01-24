<script lang="ts">
	import { Brain, Sparkles, Zap } from '@lucide/svelte';
	interface Props {
		show?: boolean;
		text?: string;
		variant?: 'brain' | 'sparkles' | 'zap' | 'pulse';
	}
	let { show = false, text = 'Загрузка...', variant = 'brain' }: Props = $props();
</script>

{#if show}
	<div
		class="fixed inset-0 z-[9999] flex items-center justify-center bg-primary/80 backdrop-blur-sm"
	>
		<div
			class="flex flex-col items-center gap-4 rounded-2xl border border-tertiary bg-secondary/95 p-8 shadow-2xl backdrop-blur-md"
		>
			{#if variant === 'brain'}
				<div class="relative">
					<Brain class="h-12 w-12 animate-pulse text-accent" />
					<div
						class="absolute inset-0 h-12 w-12 animate-spin rounded-full border-2 border-accent/30"
					></div>
					<div
						class="absolute inset-2 h-8 w-8 animate-spin rounded-full border-2 border-accent/50"
						style="animation-direction: reverse; animation-duration: 1.5s;"
					></div>
				</div>
			{:else if variant === 'sparkles'}
				<div class="relative">
					<Sparkles class="h-12 w-12 animate-spin text-accent" style="animation-duration: 2s;" />
					<div
						class="absolute -right-2 -top-2 h-4 w-4 animate-ping rounded-full bg-accent/60"
					></div>
					<div
						class="absolute -bottom-2 -left-2 h-3 w-3 animate-ping rounded-full bg-accent/40"
						style="animation-delay: 0.5s;"
					></div>
				</div>
			{:else if variant === 'zap'}
				<div class="relative">
					<Zap class="h-12 w-12 animate-bounce text-accent" />
					<div class="absolute inset-0 h-12 w-12 animate-pulse rounded-full bg-accent/20"></div>
				</div>
			{:else if variant === 'pulse'}
				<div class="flex gap-2">
					<div
						class="h-4 w-4 animate-bounce rounded-full bg-accent"
						style="animation-delay: 0ms;"
					></div>
					<div
						class="h-4 w-4 animate-bounce rounded-full bg-accent"
						style="animation-delay: 150ms;"
					></div>
					<div
						class="h-4 w-4 animate-bounce rounded-full bg-accent"
						style="animation-delay: 300ms;"
					></div>
				</div>
			{/if}
			<p class="animate-pulse font-medium text-text-primary">{text}</p>
		</div>
	</div>
{/if}

<style>
	@keyframes bounce {
		0%,
		80%,
		100% {
			transform: translateY(0);
		}
		40% {
			transform: translateY(-10px);
		}
	}
	.animate-bounce {
		animation: bounce 1.4s infinite ease-in-out;
	}
</style>
