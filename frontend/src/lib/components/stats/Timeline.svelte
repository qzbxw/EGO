<script lang="ts">
	import { Calendar, Clock, History } from '@lucide/svelte';
	import { _ as t, locale } from 'svelte-i18n';

	interface TimelineData {
		first_request_at: string | null;
		last_request_at: string | null;
		days_active: number;
	}

	let { stats }: { stats: TimelineData } = $props();

	function formatDate(dateStr: string | null): string {
		if (!dateStr) return 'Never';
		const loc = $locale || 'en-US';
		return new Date(dateStr).toLocaleDateString(loc, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function formatTime(dateStr: string | null): string {
		if (!dateStr) return '';
		const loc = $locale || 'en-US';
		return new Date(dateStr).toLocaleTimeString(loc, {
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="rounded-2xl border border-tertiary bg-secondary/30 p-6 transition-all hover:bg-secondary/50">
	<div class="mb-6 flex items-center gap-3">
		<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10 text-accent">
			<History class="h-5 w-5" />
		</div>
		<span class="font-bold text-text-primary">{$t('stats.timeline')}</span>
	</div>

	<div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
		<div class="relative pl-6">
			<div class="absolute left-0 top-1 h-2 w-2 rounded-full bg-accent"></div>
			<div class="absolute bottom-1 left-[3.5px] top-4 w-[1px] bg-tertiary"></div>
			<div class="mb-1 text-[10px] font-bold uppercase tracking-wider text-text-secondary">
				{$t('stats.first_request')}
			</div>
			<div class="text-sm font-bold text-text-primary">{formatDate(stats.first_request_at)}</div>
			<div class="text-[10px] text-text-secondary">{formatTime(stats.first_request_at)}</div>
		</div>

		<div class="relative pl-6">
			<div class="absolute left-0 top-1 h-2 w-2 rounded-full bg-accent"></div>
			<div class="mb-1 text-[10px] font-bold uppercase tracking-wider text-text-secondary">
				{$t('stats.last_request')}
			</div>
			<div class="text-sm font-bold text-text-primary">{formatDate(stats.last_request_at)}</div>
			<div class="text-[10px] text-text-secondary">{formatTime(stats.last_request_at)}</div>
		</div>
	</div>
</div>
