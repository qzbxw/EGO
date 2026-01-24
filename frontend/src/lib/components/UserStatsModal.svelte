<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { uiStore, setShowUserStatsModal } from '$lib/stores/ui.svelte.ts';
	import { api } from '$lib/api';
	import { toast } from 'svelte-sonner';
	import { _ as t } from 'svelte-i18n';
	import type { UserStats, ProviderTokens } from '$lib/types';
	import StatsOverview from '$lib/components/stats/StatsOverview.svelte';
	import PerformanceStats from '$lib/components/stats/PerformanceStats.svelte';
	import UsageBreakdown from '$lib/components/stats/UsageBreakdown.svelte';
	import Timeline from '$lib/components/stats/Timeline.svelte';
	import { RefreshCw, User, BarChart2 } from '@lucide/svelte';

	let stats: UserStats | null = $state(null);
	let providerTokens: ProviderTokens[] = $state([]);
	let isLoading = $state(false);
	let error = $state('');

	$effect(() => {
		if (uiStore.showUserStatsModal && !stats) {
			loadStats();
		}
		if (!uiStore.showUserStatsModal) {
			stats = null;
			error = '';
			providerTokens = [];
		}
	});

	async function loadStats() {
		if (isLoading) return;
		isLoading = true;
		error = '';
		try {
			const response = await api.get<UserStats>('/statistics');
			stats = response;
			providerTokens = await api.get<ProviderTokens[]>('/statistics/provider_tokens');
		} catch (e: unknown) {
			if (e instanceof Error) {
				error = e.message;
			} else {
				error = 'Ошибка загрузки статистики';
			}
			toast.error(error);
		} finally {
			isLoading = false;
		}
	}
</script>

<Modal
	title={$t('stats.title')}
	show={uiStore.showUserStatsModal}
	onclose={() => setShowUserStatsModal(false)}
	size="4xl"
>
	<div class="space-y-8">
		{#if isLoading && !stats}
			<div class="flex flex-col items-center justify-center py-20">
				<div class="relative h-12 w-12">
					<div
						class="absolute inset-0 animate-ping rounded-full bg-accent/20"
						style="animation-duration: 2s"
					></div>
					<div
						class="relative flex h-full w-full items-center justify-center rounded-full bg-accent/10 text-accent"
					>
						<RefreshCw class="h-6 w-6 animate-spin" />
					</div>
				</div>
				<span class="mt-4 font-medium text-text-secondary">{$t('stats.loading')}</span>
			</div>
		{:else if error}
			<div class="py-20 text-center">
				<div
					class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10 text-red-500"
				>
					<BarChart2 class="h-6 w-6" />
				</div>
				<div class="mb-6 text-red-500">{error}</div>
				<button
					class="flex items-center gap-2 rounded-xl bg-accent px-6 py-2.5 font-bold text-white shadow-lg shadow-accent/20 transition-all hover:bg-accent-hover active:scale-95"
					onclick={loadStats}
				>
					<RefreshCw class="h-4 w-4" />
					{$t('stats.retry')}
				</button>
			</div>
		{:else if stats}
			<div class="space-y-8">
				<div class="flex items-center justify-between rounded-2xl bg-secondary/50 p-4 ring-1 ring-white/5">
					<div class="flex items-center gap-4">
						<div
							class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-accent to-purple-600 shadow-lg shadow-accent/20"
						>
							<User class="h-6 w-6 text-white" />
						</div>
						<div>
							<h2 class="text-xl font-bold tracking-tight text-text-primary">{stats.username}</h2>
							<p class="text-sm text-text-secondary">{$t('stats.subtitle')}</p>
						</div>
					</div>
					<button
						class="group flex h-10 w-10 items-center justify-center rounded-xl bg-tertiary/50 text-text-secondary transition-all hover:bg-tertiary hover:text-text-primary disabled:opacity-50"
						onclick={loadStats}
						disabled={isLoading}
						title={$t('stats.update')}
					>
						<RefreshCw class="h-5 w-5 transition-transform group-hover:rotate-180 {isLoading ? 'animate-spin' : ''}" />
					</button>
				</div>

				<StatsOverview {stats} />

				<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
					<PerformanceStats {stats} />
					<Timeline {stats} />
				</div>

				<UsageBreakdown {stats} {providerTokens} />
			</div>
		{:else}
			<div class="py-20 text-center">
				<div
					class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-accent/5 text-accent"
				>
					<BarChart2 class="h-8 w-8 opacity-40" />
				</div>
				<div class="mb-2 text-xl font-bold text-text-primary">{$t('stats.no_data_title')}</div>
				<p class="mb-8 text-sm text-text-secondary">{$t('stats.no_data_hint')}</p>
				<button
					class="rounded-xl bg-accent px-8 py-3 font-bold text-white shadow-lg shadow-accent/20 transition-all hover:bg-accent-hover hover:shadow-accent/40 active:scale-95"
					onclick={loadStats}
				>
					{$t('stats.load')}
				</button>
			</div>
		{/if}
	</div>
</Modal>
