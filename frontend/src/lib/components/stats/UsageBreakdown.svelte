<script lang="ts">
	import { TrendingUp, Activity } from '@lucide/svelte';
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
		return new Intl.NumberFormat('en-US').format(safe);
	}
</script>
<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
	<div class="bg-secondary/50 rounded-xl p-4 border border-tertiary">
		<div class="flex items-center gap-3 mb-3">
			<TrendingUp class="w-5 h-5 text-accent" />
			<span class="font-semibold">{$t('stats.providers')}</span>
		</div>
		<div class="space-y-2">
			{#each Object.entries(stats.provider_usage).sort((a, b) => b[1] - a[1]) as [provider, count]}
				<div class="flex justify-between items-center">
					<span class="capitalize">{provider}</span>
					<div class="flex items-center gap-2">
						<div class="w-20 bg-tertiary rounded-full h-2"><div class="bg-accent h-2 rounded-full transition-all duration-300" style="width: {stats.total_requests > 0 ? (Math.min(100, Math.max(0, ((Number(count)||0) / stats.total_requests) * 100))) : 0}%"></div></div>
						<span class="text-sm font-medium w-12 text-right">{formatNumber(Number(count) || 0)}</span>
					</div>
				</div>
			{/each}
		</div>
		{#if providerTokens.length > 0}
		<div class="mt-4 space-y-2">
			{#each providerTokens.filter(row => (row.total_tokens ?? 0) > 0) as row}
				<div class="flex justify-between items-center text-sm">
					<span class="capitalize">{row.llm_provider}</span>
					<span class="text-text-secondary">{formatNumber(row.total_tokens ?? 0)} ({formatNumber(row.prompt_tokens ?? 0)} / {formatNumber(row.completion_tokens ?? 0)})</span>
				</div>
			{/each}
		</div>
		{/if}
	</div>
	<div class="bg-secondary/50 rounded-xl p-4 border border-tertiary">
		<div class="flex items-center gap-3 mb-3">
			<Activity class="w-5 h-5 text-accent" />
			<span class="font-semibold">{$t('stats.models')}</span>
		</div>
		<div class="space-y-2">
			{#each Object.entries(stats.model_usage).sort((a, b) => (Number(b[1])||0) - (Number(a[1])||0)).slice(0, 5) as [model, count]}
				<div class="flex justify-between items-center">
					<span class="text-sm truncate">{model}</span>
					<div class="flex items-center gap-2">
						<div class="w-16 bg-tertiary rounded-full h-2"><div class="bg-accent h-2 rounded-full transition-all duration-300" style="width: {stats.total_requests > 0 ? (Math.min(100, Math.max(0, ((Number(count)||0) / stats.total_requests) * 100))) : 0}%"></div></div>
						<span class="text-sm font-medium w-8 text-right">{formatNumber(Number(count) || 0)}</span>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>