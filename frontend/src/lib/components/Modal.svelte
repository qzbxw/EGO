<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		show,
		title,
		onclose,
		children,
		size = 'lg'
	} = $props<{
		show: boolean;
		title: string;
		onclose: () => void;
		children: Snippet;
		size?: 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl';
	}>();

	const sizeClasses = {
		lg: 'max-w-lg',
		xl: 'max-w-xl',
		'2xl': 'max-w-2xl',
		'3xl': 'max-w-3xl',
		'4xl': 'max-w-4xl',
		'5xl': 'max-w-5xl'
	};
</script>

{#if show}
	<div
		class="fixed inset-0 z-[60] flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<div
			class="absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity duration-300"
			transition:fade={{ duration: 300 }}
			onclick={onclose}
			onkeydown={(e) => e.key === 'Escape' && onclose()}
			role="button"
			tabindex="-1"
		></div>

		<!-- Modal Container -->
		<div
			class="relative flex max-h-[90vh] w-full {sizeClasses[
				size as keyof typeof sizeClasses
			]} flex-col overflow-hidden rounded-[24px] bg-secondary shadow-2xl ring-1 ring-white/5"
			transition:scale={{ duration: 350, start: 0.96, easing: cubicOut, opacity: 0 }}
			role="document"
		>
			<!-- Header (Seamless) -->
			<div class="flex shrink-0 items-start justify-between px-8 pb-4 pt-8">
				<h2 id="modal-title" class="text-3xl font-bold tracking-tight text-text-primary">
					{title}
				</h2>
				<button
					onclick={onclose}
					class="group -mr-2 -mt-2 flex h-10 w-10 items-center justify-center rounded-full bg-tertiary transition-all hover:bg-tertiary/80 active:scale-90"
					aria-label="Close"
				>
					<X class="h-5 w-5 text-text-secondary transition-colors group-hover:text-text-primary" />
				</button>
			</div>

			<!-- Content -->
			<div class="custom-scrollbar overflow-y-auto overflow-x-hidden px-8 pb-8 pt-2">
				{@render children()}
			</div>
		</div>
	</div>
{/if}

<style>
	.custom-scrollbar::-webkit-scrollbar {
		width: 6px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background: rgba(156, 163, 175, 0.2);
		border-radius: 10px;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: rgba(156, 163, 175, 0.4);
	}
</style>
