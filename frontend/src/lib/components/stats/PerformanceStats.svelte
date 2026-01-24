<script lang="ts">
	import { Clock, Zap, Gauge } from '@lucide/svelte';
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
		if (safe < 60000) return `${(safe / 1000).toFixed(1)}${sUnit}`;
		return `${(safe / 60000).toFixed(1)}${mUnit}`;
	}

	// Calculate relative widths for a "bar" visualization
	const max = stats.max_response_time_ms || 1;
	const avgWidth = (stats.avg_response_time_ms / max) * 100;
	const minWidth = (stats.min_response_time_ms / max) * 100;
</script>

<div class="rounded-2xl border border-tertiary bg-secondary/30 p-6 transition-all hover:bg-secondary/50">
	<div class="mb-6 flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-orange-500/10 text-orange-500">
				<Gauge class="h-5 w-5" />
			</div>
			<span class="font-bold text-text-primary">{$t('stats.performance')}</span>
		</div>
		<div class="flex items-center gap-1.5 rounded-lg bg-orange-500/10 px-2.5 py-1 text-[10px] font-black uppercase text-orange-500">
			<Zap class="h-3 w-3" />
			Speed
		</div>
	</div>

	<div class="space-y-6">
		<div class="relative space-y-2">
			<div class="flex justify-between text-xs font-medium">
				<span class="text-text-secondary">{$t('stats.avg')}</span>
				<span class="text-text-primary">{formatDuration(stats.avg_response_time_ms)}</span>
			</div>
			<div class="h-2 w-full rounded-full bg-tertiary/50">
				<div
					class="h-full rounded-full bg-orange-500 transition-all duration-1000"
					style="width: {avgWidth}%"
				></div>
			</div>
		</div>

		<div class="grid grid-cols-2 gap-4">
			<div class="rounded-xl border border-emerald-500/10 bg-emerald-500/5 p-3">
				<div class="mb-1 text-[10px] font-bold uppercase text-emerald-500/70">{$t('stats.best')}</div>
				<div class="text-lg font-black text-emerald-500">
					{formatDuration(stats.min_response_time_ms)}
				</div>
			</div>
			<div class="rounded-xl border border-red-500/10 bg-red-500/5 p-3">
				<div class="mb-1 text-[10px] font-bold uppercase text-red-500/70">{$t('stats.worst')}</div>
				<div class="text-lg font-black text-red-500">
					{formatDuration(stats.max_response_time_ms)}
				</div>
			</div>
		</div>
	</div>
</div>
