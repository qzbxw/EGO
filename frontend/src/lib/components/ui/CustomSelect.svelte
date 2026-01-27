<script lang="ts">
	import { slide } from 'svelte/transition';
	import { ChevronDown, Check } from '@lucide/svelte';
	import { clickOutside } from '$lib/actions/clickOutside';

	let {
		value = $bindable(),
		options = [],
		placeholder = 'Select...',
		disabled = false
	}: {
		value: string;
		options: { value: string; label: string }[];
		placeholder?: string;
		disabled?: boolean;
	} = $props();

	let isOpen = $state(false);

	function toggle() {
		if (!disabled) isOpen = !isOpen;
	}

	function select(optionValue: string) {
		if (disabled) return;
		value = optionValue;
		// Dispatch/Effect is handled by binding
		isOpen = false;
	}

	function close() {
		isOpen = false;
	}

	let selectedLabel = $derived(options.find((o) => o.value === value)?.label || placeholder);
</script>

<div class="relative w-full" use:clickOutside={close}>
	<button
		type="button"
		class="flex w-full items-center justify-between rounded-lg border border-tertiary bg-secondary/50 p-3 text-left outline-none transition-all hover:border-accent/40 focus:border-accent/60 {disabled
			? 'cursor-not-allowed opacity-60'
			: 'cursor-pointer'}"
		onclick={toggle}
		{disabled}
	>
		<span class="text-sm font-medium {value ? 'text-text-primary' : 'text-text-secondary'}">
			{selectedLabel}
		</span>
		<ChevronDown
			class="h-4 w-4 text-text-secondary transition-transform duration-200 {isOpen
				? 'rotate-180'
				: ''}"
		/>
	</button>

	{#if isOpen}
		<div
			class="absolute left-0 right-0 top-full z-50 mt-1 max-h-60 overflow-y-auto rounded-lg border border-tertiary bg-secondary shadow-xl"
			transition:slide={{ duration: 150 }}
		>
			{#each options as option (option.value)}
				<button
					type="button"
					class="group flex w-full items-center justify-between px-4 py-2.5 text-left text-sm transition-colors hover:bg-tertiary/50 {option.value ===
					value
						? 'bg-accent/5 font-medium text-accent'
						: 'text-text-primary'}"
					onclick={() => select(option.value)}
				>
					<span>{option.label}</span>
					{#if option.value === value}
						<Check class="h-4 w-4 text-accent" />
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>
