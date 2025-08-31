<script lang="ts">
	import { Clock } from '@lucide/svelte';
	import { _ as t, locale } from 'svelte-i18n';
	interface PerformanceStatsData {
		avg_response_time_ms: number;
		min_response_time_ms: number;
		max_response_time_ms: number;
	}
	let { stats }: { stats: PerformanceStatsData } = $props();
	function formatDuration(ms: number): string {
		const safe = Number.isFinite(ms) ? ms : 0;
		const isRu = ($locale || 'en-US').startsWith('ru');
		const msUnit = isRu ? 'мс' : 'ms';
		const sUnit = isRu ? 'с' : 's';
		const mUnit = isRu ? 'м' : 'm';
		if (safe < 1000) return `${safe}${msUnit}`;
		if (safe < 60000) return `${(safe/1000).toFixed(1)}${sUnit}`;
		return `${(safe/60000).toFixed(1)}${mUnit}`;
	}
</script>
<div class="bg-gradient-to-br from-orange-500/10 to-orange-600/20 rounded-xl p-4 border border-orange-500/20">
	<div class="flex items-center gap-3 mb-3">
		<Clock class="w-5 h-5 text-orange-500" />
		<span class="font-semibold text-orange-500">{$t('stats.performance')}</span>
	</div>
	<div class="space-y-2">
		<div class="flex justify-between"><span class="text-text-secondary">{$t('stats.avg')}</span><span class="font-medium">{formatDuration(stats.avg_response_time_ms)}</span></div>
		<div class="flex justify-between"><span class="text-text-secondary">{$t('stats.best')}</span><span class="font-medium text-green-600">{formatDuration(stats.min_response_time_ms)}</span></div>
		<div class="flex justify-between"><span class="text-text-secondary">{$t('stats.worst')}</span><span class="font-medium text-red-600">{formatDuration(stats.max_response_time_ms)}</span></div>
	</div>
</div>