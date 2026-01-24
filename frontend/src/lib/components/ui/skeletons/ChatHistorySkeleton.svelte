<script lang="ts">
	import { fade } from 'svelte/transition';
</script>

<div class="flex-1 animate-pulse space-y-8 p-4 md:p-8" transition:fade={{ duration: 300 }}>
	{#each Array(3) as _}
		<div class="flex flex-col gap-4">
			<div class="flex max-w-[80%] flex-col items-end gap-2 self-end">
				<div class="h-10 w-64 rounded-2xl rounded-tr-none bg-tertiary/50"></div>
			</div>

			<div class="flex max-w-[80%] flex-col items-start gap-2 self-start">
				<div class="mb-1 flex items-center gap-2">
					<div class="h-8 w-8 rounded-full bg-tertiary/50"></div>
					<div class="h-4 w-12 rounded bg-tertiary/50"></div>
				</div>
				<div
					class="h-24 w-full rounded-2xl rounded-tl-none border border-tertiary/20 bg-tertiary/30 md:w-[600px]"
				></div>
			</div>
		</div>
	{/each}
</div>
