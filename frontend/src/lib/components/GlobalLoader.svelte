<script lang="ts">
  import { Brain, Sparkles, Zap } from '@lucide/svelte';
  interface Props {
    show?: boolean;
    text?: string;
    variant?: 'brain' | 'sparkles' | 'zap' | 'pulse';
  }
  let { 
    show = false, 
    text = 'Загрузка...', 
    variant = 'brain' 
  }: Props = $props();
</script>
{#if show}
  <div class="fixed inset-0 z-[9999] flex items-center justify-center bg-background/80 backdrop-blur-sm">
    <div class="flex flex-col items-center gap-4 p-8 rounded-2xl bg-secondary/90 backdrop-blur-md border border-border shadow-2xl">
      {#if variant === 'brain'}
        <div class="relative">
          <Brain class="w-12 h-12 text-accent animate-pulse" />
          <div class="absolute inset-0 w-12 h-12 border-2 border-accent/30 rounded-full animate-spin"></div>
          <div class="absolute inset-2 w-8 h-8 border-2 border-accent/50 rounded-full animate-spin" style="animation-direction: reverse; animation-duration: 1.5s;"></div>
        </div>
      {:else if variant === 'sparkles'}
        <div class="relative">
          <Sparkles class="w-12 h-12 text-accent animate-spin" style="animation-duration: 2s;" />
          <div class="absolute -top-2 -right-2 w-4 h-4 bg-accent/60 rounded-full animate-ping"></div>
          <div class="absolute -bottom-2 -left-2 w-3 h-3 bg-accent/40 rounded-full animate-ping" style="animation-delay: 0.5s;"></div>
        </div>
      {:else if variant === 'zap'}
        <div class="relative">
          <Zap class="w-12 h-12 text-accent animate-bounce" />
          <div class="absolute inset-0 w-12 h-12 bg-accent/20 rounded-full animate-pulse"></div>
        </div>
      {:else if variant === 'pulse'}
        <div class="flex gap-2">
          <div class="w-4 h-4 bg-accent rounded-full animate-bounce" style="animation-delay: 0ms;"></div>
          <div class="w-4 h-4 bg-accent rounded-full animate-bounce" style="animation-delay: 150ms;"></div>
          <div class="w-4 h-4 bg-accent rounded-full animate-bounce" style="animation-delay: 300ms;"></div>
        </div>
      {/if}
      <p class="text-text-primary font-medium animate-pulse">{text}</p>
    </div>
  </div>
{/if}
<style>
  @keyframes bounce {
    0%, 80%, 100% {
      transform: translateY(0);
    }
    40% {
      transform: translateY(-10px);
    }
  }
  .animate-bounce {
    animation: bounce 1.4s infinite ease-in-out;
  }
</style>