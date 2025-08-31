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
    let stats: UserStats | null = $state(null);
    let providerTokens: ProviderTokens[] = $state([]);
    let isLoading = $state(false);
    let error = $state('');
    $effect(() => {
        if (uiStore.showUserStatsModal && !stats) {
            loadStats();
        }
        if (!uiStore.showUserStatsModal) {
            stats = null; error = ''; providerTokens = [];
        }
    });
    async function loadStats() {
        if (isLoading) return;
        isLoading = true; error = '';
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
        } finally { isLoading = false; }
    }
</script>
<Modal title={$t('stats.title')} show={uiStore.showUserStatsModal} onclose={() => setShowUserStatsModal(false)}>
    <div class="p-6 max-w-6xl">
        {#if isLoading}
            <div class="flex items-center justify-center py-12">
                <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-accent"></div>
                <span class="ml-3 text-text-secondary">{$t('stats.loading')}</span>
            </div>
        {:else if error}
            <div class="text-center py-12">
                <div class="text-red-500 mb-4">{error}</div>
                <button class="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors" onclick={loadStats}>{$t('stats.retry')}</button>
            </div>
        {:else if stats}
            <div class="space-y-6">
                <div class="text-center mb-6">
                    <h2 class="text-2xl font-bold text-text-primary mb-2">{stats.username}</h2>
                    <p class="text-text-secondary">{$t('stats.subtitle')}</p>
                </div>
                <StatsOverview {stats} />
                <PerformanceStats {stats} />
                <UsageBreakdown {stats} {providerTokens} />
                <Timeline {stats} />
                <div class="flex justify-center pt-4">
                    <button class="px-6 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors disabled:opacity-50 flex items-center gap-2" onclick={loadStats} disabled={isLoading}>
                        {#if isLoading}
                            <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                            {$t('stats.loading')}
                        {:else}
                            {$t('stats.update')}
                        {/if}
                    </button>
                </div>
            </div>
        {:else}
            <div class="text-center py-12">
                <div class="text-text-secondary mb-4">{$t('stats.no_data_title')}</div>
                <p class="text-sm text-text-secondary mb-6">{$t('stats.no_data_hint')}</p>
                <button class="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors" onclick={loadStats}>{$t('stats.load')}</button>
            </div>
        {/if}
    </div>
</Modal>