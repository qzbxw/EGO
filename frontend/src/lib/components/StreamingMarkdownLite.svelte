<script lang="ts">
  import { marked } from 'marked';
  import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
  import { textStreamStore } from '$lib/stores/stream.svelte.ts';
  try {
    marked.setOptions({ gfm: true, breaks: true, headerIds: false, mangle: false });
  } catch {}
  let { text = '', invert = false } = $props<{ text?: string; invert?: boolean }>();
  let liveText = $state('');
  $effect(() => {
    const unsub = textStreamStore.subscribe((v) => {
      liveText = v;
      scheduleRender();
    });
    return () => unsub();
  });
  let container: HTMLDivElement | undefined = $state();
  async function renderContent() {
    try {
      if (!container) return;
      const src = (text || liveText || '');
      if (!src) {
        container.innerHTML = '';
        return;
      }
      const html = await marked(src, { async: true });
      if (!container) return;
      container.innerHTML = html;
    } catch (err) {
      console.error('[StreamingMarkdownLite] render error', err);
    }
  }
  let __renderTimer: number | null = null;
  let __lastRenderAt = 0;
  function scheduleRender() {
    const now = performance.now();
    const minInterval = 80; 
    const elapsed = now - __lastRenderAt;
    if (elapsed >= minInterval) {
      __lastRenderAt = now;
      if (__renderTimer) {
        clearTimeout(__renderTimer);
        __renderTimer = null;
      }
      void renderContent().catch((e) => console.error('[StreamingMarkdownLite] renderContent rejected', e));
    } else {
      const wait = Math.max(8, minInterval - elapsed);
      if (__renderTimer) clearTimeout(__renderTimer);
      __renderTimer = window.setTimeout(() => {
        __lastRenderAt = performance.now();
        void renderContent().catch((e) => console.error('[StreamingMarkdownLite] renderContent rejected', e));
        __renderTimer = null;
      }, wait);
    }
  }
  $effect(() => {
    if (!container) return;
    void text;
    scheduleRender();
  });
  $effect(() => () => {
    if (__renderTimer) {
      clearTimeout(__renderTimer);
      __renderTimer = null;
    }
  });
</script>
<style>
  :global(.prose pre) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    word-break: break-word;
  }
  :global(.prose code) {
    word-break: break-word;
  }
</style>
<div
  bind:this={container}
  class="prose w-full min-w-0 max-w-full break-words"
  class:prose-invert={invert || preferencesStore.theme === 'dark'}
  aria-live="polite"
></div>