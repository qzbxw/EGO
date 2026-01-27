<script lang="ts">
	import { fade } from 'svelte/transition';
</script>

<div class="w-full animate-pulse space-y-4" transition:fade={{ duration: 200 }}>
	{#each Array(8) as _, i (i)}
		<div
			class="flex items-center justify-between rounded-lg border border-transparent p-3"
			title={String(_)}
		>
			<div class="flex w-full items-center space-x-3 overflow-hidden">
				<div class="h-4 w-4 flex-shrink-0 rounded-full bg-tertiary/50"></div>
				<div class="h-4 w-3/4 rounded bg-tertiary/50"></div>
			</div>
		</div>
	{/each}
</div>
