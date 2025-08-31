<script lang="ts">
    import { Calendar } from '@lucide/svelte';
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
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
</script>
<div class="bg-secondary/30 rounded-xl p-4 border border-tertiary">
    <div class="flex items-center gap-3 mb-3">
        <Calendar class="w-5 h-5 text-accent" />
        <span class="font-semibold">{$t('stats.timeline')}</span>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
            <div class="text-sm text-text-secondary mb-1">{$t('stats.first_request')}</div>
            <div class="font-medium">{formatDate(stats.first_request_at)}</div>
        </div>
        <div>
            <div class="text-sm text-text-secondary mb-1">{$t('stats.last_request')}</div>
            <div class="font-medium">{formatDate(stats.last_request_at)}</div>
        </div>
    </div>
</div>