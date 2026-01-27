<script lang="ts">
	import { BarChart3, FileText, Zap, TrendingUp } from '@lucide/svelte';
	import { _ as t } from 'svelte-i18n';
	import type { UserStats } from '$lib/types';

	let { stats }: { stats: UserStats } = $props();

	function formatNumber(num: number): string {
		const safe = Number.isFinite(num) ? num : 0;
		if (safe >= 1000000) return (safe / 1000000).toFixed(1) + 'M';
		if (safe >= 1000) return (safe / 1000).toFixed(1) + 'k';
		return new Intl.NumberFormat('en-US').format(safe);
	}

	function safeFixed(num: number, digits = 1): string {
		const safe = Number.isFinite(num) ? num : 0;
		return safe.toFixed(digits);
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}
</script>

<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
	<div
		class="relative overflow-hidden rounded-2xl border border-blue-500/20 bg-blue-500/5 p-5 transition-all hover:bg-blue-500/10"
	>
		<div class="mb-4 flex items-center justify-between">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/20 text-blue-500"
			>
				<BarChart3 class="h-5 w-5" />
			</div>
			<div
				class="flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-blue-500/70"
			>
				<TrendingUp class="h-3 w-3" />
				Live
			</div>
		</div>
		<div class="space-y-1">
			<div class="text-3xl font-black tracking-tight text-text-primary">
				{formatNumber(stats.total_requests)}
			</div>
			<div class="text-xs font-medium text-text-secondary">
				{$t('stats.total_requests')}
			</div>
		</div>
		<div class="mt-4 flex items-center gap-2 border-t border-blue-500/10 pt-3">
			<span class="text-[10px] font-bold uppercase text-blue-500/80">Daily Average</span>
			<span class="text-xs font-bold text-text-primary">{safeFixed(stats.requests_per_day, 1)}</span
			>
		</div>
	</div>

	<div
		class="relative overflow-hidden rounded-2xl border border-emerald-500/20 bg-emerald-500/5 p-5 transition-all hover:bg-emerald-500/10"
	>
		<div class="mb-4 flex items-center justify-between">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/20 text-emerald-500"
			>
				<FileText class="h-5 w-5" />
			</div>
		</div>
		<div class="space-y-1">
			<div class="text-3xl font-black tracking-tight text-text-primary">
				{formatNumber(stats.total_sessions)}
			</div>
			<div class="text-xs font-medium text-text-secondary">
				{$t('stats.sessions')}
			</div>
		</div>
		<div class="mt-4 flex items-center gap-2 border-t border-emerald-500/10 pt-3">
			<span class="text-xs font-bold text-text-primary">{stats.total_files_uploaded}</span>
			<span class="text-[10px] font-medium uppercase text-text-secondary"
				>{$t('stats.files')} ({formatBytes(stats.total_files_size_mb * 1024 * 1024)})</span
			>
		</div>
	</div>

	{#if stats.total_tokens > 0}
		<div
			class="relative overflow-hidden rounded-2xl border border-purple-500/20 bg-purple-500/5 p-5 transition-all hover:bg-purple-500/10"
		>
			<div class="mb-4 flex items-center justify-between">
				<div
					class="flex h-10 w-10 items-center justify-center rounded-xl bg-purple-500/20 text-purple-500"
				>
					<Zap class="h-5 w-5" />
				</div>
			</div>
			<div class="space-y-1">
				<div class="text-3xl font-black tracking-tight text-text-primary">
					{formatNumber(stats.total_tokens)}
				</div>
				<div class="text-xs font-medium text-text-secondary">
					{$t('stats.tokens')}
				</div>
			</div>
			<div class="mt-4 border-t border-purple-500/10 pt-3">
				<div class="flex justify-between text-[10px] font-medium text-text-secondary">
					<span>IN: {formatNumber(stats.total_prompt_tokens)}</span>
					<span>OUT: {formatNumber(stats.total_completion_tokens)}</span>
				</div>
				<div class="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-purple-500/10">
					<div
						class="h-full bg-purple-500 transition-all duration-1000"
						style="width: {(stats.total_prompt_tokens / stats.total_tokens) * 100}%"
					></div>
				</div>
			</div>
		</div>
	{/if}
</div>
