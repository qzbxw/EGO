<script lang="ts">
	import { TrendingUp, Activity, Cpu, Globe } from '@lucide/svelte';
	import { _ as t } from 'svelte-i18n';
	import type { ProviderTokens } from '$lib/types';

	interface UsageStatsData {
		total_requests: number;
		provider_usage: Record<string, number>;
		model_usage: Record<string, number>;
	}

	let {
		stats,
		providerTokens
	}: {
		stats: UsageStatsData;
		providerTokens: ProviderTokens[];
	} = $props();

	function formatNumber(num: number): string {
		const safe = Number.isFinite(num) ? num : 0;
		if (safe >= 1000) return (safe / 1000).toFixed(1) + 'k';
		return new Intl.NumberFormat('en-US').format(safe);
	}

	function getPercentage(count: number, total: number): number {
		if (total === 0) return 0;
		return Math.min(100, Math.max(0, (count / total) * 100));
	}
</script>

<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
	<div
		class="rounded-2xl border border-tertiary bg-secondary/30 p-6 transition-all hover:bg-secondary/50"
	>
		<div class="mb-6 flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10 text-accent">
					<Globe class="h-5 w-5" />
				</div>
				<span class="font-bold text-text-primary">{$t('stats.providers')}</span>
			</div>
		</div>

		<div class="space-y-4">
			{#each Object.entries(stats.provider_usage).sort((a, b) => b[1] - a[1]) as [provider, count]}
				<div class="space-y-2">
					<div class="flex items-center justify-between text-xs font-medium">
						<span class="capitalize text-text-primary">{provider}</span>
						<span class="text-text-secondary"
							>{formatNumber(Number(count))} ({getPercentage(
								Number(count),
								stats.total_requests
							).toFixed(1)}%)</span
						>
					</div>
					<div class="h-1.5 w-full overflow-hidden rounded-full bg-tertiary/50">
						<div
							class="h-full bg-accent transition-all duration-1000"
							style="width: {getPercentage(Number(count), stats.total_requests)}%"
						></div>
					</div>
				</div>
			{/each}
		</div>

		{#if providerTokens.length > 0}
			<div class="mt-8 border-t border-tertiary/50 pt-6 space-y-3">
				<h4 class="text-[10px] font-black uppercase tracking-widest text-text-secondary">Token Distribution</h4>
				{#each providerTokens.filter((row) => (row.total_tokens ?? 0) > 0) as row}
					<div class="flex items-center justify-between text-[11px]">
						<div class="flex items-center gap-2">
							<div class="h-1.5 w-1.5 rounded-full bg-accent"></div>
							<span class="capitalize font-medium text-text-primary">{row.llm_provider}</span>
						</div>
						<div class="flex items-center gap-3 text-text-secondary">
							<span>{formatNumber(row.total_tokens ?? 0)} tokens</span>
							<span class="text-[10px] opacity-50">{formatNumber(row.prompt_tokens ?? 0)} / {formatNumber(row.completion_tokens ?? 0)}</span>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div
		class="rounded-2xl border border-tertiary bg-secondary/30 p-6 transition-all hover:bg-secondary/50"
	>
		<div class="mb-6 flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10 text-accent">
					<Cpu class="h-5 w-5" />
				</div>
				<span class="font-bold text-text-primary">{$t('stats.models')}</span>
			</div>
		</div>

		<div class="space-y-4">
			{#each Object.entries(stats.model_usage)
				.sort((a, b) => (Number(b[1]) || 0) - (Number(a[1]) || 0))
				.slice(0, 8) as [model, count]}
				<div class="space-y-2">
					<div class="flex items-center justify-between text-xs font-medium">
						<span class="truncate pr-4 text-text-primary">{model}</span>
						<span class="shrink-0 text-text-secondary">{formatNumber(Number(count))}</span>
					</div>
					<div class="h-1.5 w-full overflow-hidden rounded-full bg-tertiary/50">
						<div
							class="h-full bg-accent/60 transition-all duration-1000"
							style="width: {getPercentage(Number(count), stats.total_requests)}%"
						></div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>
