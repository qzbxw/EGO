<script lang="ts">
	import { BarChart3, FileText, Zap } from '@lucide/svelte';
	import { _ as t } from 'svelte-i18n';
	import type { UserStats } from '$lib/types';
	let { stats }: { stats: UserStats } = $props();
	function formatNumber(num: number): string {
		const safe = Number.isFinite(num) ? num : 0;
		return new Intl.NumberFormat('en-US').format(safe);
	}
	function safeFixed(num: number, digits = 1): string {
		const safe = Number.isFinite(num) ? num : 0;
		return safe.toFixed(digits);
	}
	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024; 
		const sizes = ['B','KB','MB','GB'];
		const i = Math.floor(Math.log(bytes)/Math.log(k));
		return parseFloat((bytes/Math.pow(k,i)).toFixed(2)) + ' ' + sizes[i];
	}
</script>
<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
	<div class="bg-gradient-to-br from-blue-500/10 to-blue-600/20 rounded-xl p-4 border border-blue-500/20">
		<div class="flex items-center gap-3 mb-2">
			<BarChart3 class="w-5 h-5 text-blue-500" />
			<span class="font-semibold text-blue-500">{$t('stats.total_requests')}</span>
		</div>
		<div class="text-2xl font-bold">{formatNumber(stats.total_requests)}</div>
		<div class="text-sm text-text-secondary mt-1">{safeFixed(stats.requests_per_day, 1)} {$t('stats.per_day')}</div>
	</div>
	<div class="bg-gradient-to-br from-green-500/10 to-green-600/20 rounded-xl p-4 border border-green-500/20">
		<div class="flex items-center gap-3 mb-2">
			<FileText class="w-5 h-5 text-green-500" />
			<span class="font-semibold text-green-500">{$t('stats.sessions')}</span>
		</div>
		<div class="text-2xl font-bold">{formatNumber(stats.total_sessions)}</div>
		<div class="text-sm text-text-secondary mt-1">{stats.total_files_uploaded} {$t('stats.files')} ({formatBytes(stats.total_files_size_mb * 1024 * 1024)})</div>
	</div>
	{#if stats.total_tokens > 0}
	<div class="bg-gradient-to-br from-purple-500/10 to-purple-600/20 rounded-xl p-4 border border-purple-500/20">
		<div class="flex items-center gap-3 mb-2">
			<Zap class="w-5 h-5 text-purple-500" />
			<span class="font-semibold text-purple-500">{$t('stats.tokens')}</span>
		</div>
		<div class="text-2xl font-bold">{formatNumber(stats.total_tokens)}</div>
		<div class="text-sm text-text-secondary mt-1">{formatNumber(stats.total_prompt_tokens)} / {formatNumber(stats.total_completion_tokens)}</div>
	</div>
	{/if}
</div>