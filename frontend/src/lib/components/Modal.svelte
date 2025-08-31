<script lang="ts">
	import { fade } from 'svelte/transition';
	import { X } from '@lucide/svelte';
    import type { Snippet } from 'svelte';
	let { show, title, onclose, children } = $props<{
		show: boolean;
		title: string;
		onclose: () => void;
		children: Snippet;
	}>();
</script>
{#if show}
    <div
        class="fixed inset-0 z-40 flex items-center justify-center"
		transition:fade={{ duration: 150 }}
		onclick={onclose}
		role="dialog"
		aria-modal="true"
		aria-labelledby="modal-title"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && onclose()}
	>
		<div class="absolute inset-0 bg-black/30 backdrop-blur-md"></div>
        <div
            class="relative bg-secondary/95 border border-tertiary rounded-xl sm:rounded-2xl shadow-2xl shadow-accent/10 w-full max-w-lg md:max-w-xl lg:max-w-2xl m-4 p-0 animate-fade-in-up"
            onclick={(e) => e.stopPropagation()}
            role="document"
        >
            <div class="sticky top-0 z-10 flex items-center justify-between px-6 pt-5 pb-3 border-b border-tertiary/60 bg-secondary/95 rounded-t-xl sm:rounded-t-2xl">
                <h2 id="modal-title" class="text-xl md:text-2xl font-bold text-text-primary">{title}</h2>
                <button onclick={onclose} class="p-2 rounded-full hover:bg-tertiary transition-colors" aria-label="Закрыть модальное окно">
                    <X class="w-5 h-5 text-text-secondary" />
                </button>
            </div>
            <div class="text-text-primary px-6 pb-6 pt-4 max-h-[75vh] overflow-y-auto">
                {@render children()}
            </div>
        </div>
	</div>
{/if}