<script lang="ts">
  import { Copy, Check } from '@lucide/svelte';
  import { createEventDispatcher } from 'svelte';
  interface Props {
    code: string;
    language?: string;
  }
  let { code, language = '' }: Props = $props();
  let copied = $state(false);
  let copyTimeout: number;
  const dispatch = createEventDispatcher();
  async function copyCode() {
    try {
      await navigator.clipboard.writeText(code);
      copied = true;
      dispatch('copy', { success: true });
      if (copyTimeout) clearTimeout(copyTimeout);
      copyTimeout = setTimeout(() => {
        copied = false;
      }, 2000);
    } catch (err) {
      console.error('Failed to copy code:', err);
      dispatch('copy', { success: false });
    }
  }
  function getLanguageDisplay(lang: string): string {
    const languageMap: Record<string, string> = {
      'js': 'JavaScript',
      'javascript': 'JavaScript',
      'ts': 'TypeScript',
      'typescript': 'TypeScript',
      'py': 'Python',
      'python': 'Python',
      'go': 'Go',
      'rust': 'Rust',
      'java': 'Java',
      'cpp': 'C++',
      'c': 'C',
      'html': 'HTML',
      'css': 'CSS',
      'scss': 'SCSS',
      'json': 'JSON',
      'yaml': 'YAML',
      'yml': 'YAML',
      'xml': 'XML',
      'sql': 'SQL',
      'bash': 'Bash',
      'sh': 'Shell',
      'dockerfile': 'Dockerfile',
      'md': 'Markdown',
      'markdown': 'Markdown'
    };
    return languageMap[lang.toLowerCase()] || lang.toUpperCase();
  }
</script>
<div class="code-wrapper group" data-language={getLanguageDisplay(language)}>
  <pre><code class="hljs language-{language}">{code}</code></pre>
  <button 
    class="copy-button"
    onclick={copyCode}
    title={copied ? 'Copied!' : 'Copy code'}
    aria-label={copied ? 'Code copied to clipboard' : 'Copy code to clipboard'}
  >
    {#if copied}
      <Check class="w-4 h-4" />
    {:else}
      <Copy class="w-4 h-4" />
    {/if}
  </button>
</div>
<style>
  .code-wrapper {
    position: relative;
    margin: 1rem 0;
  }
  .code-wrapper:hover .copy-button {
    opacity: 1;
  }
  .copy-button {
    position: absolute;
    top: 0.75rem;
    right: 0.75rem;
    padding: 0.5rem;
    border-radius: 0.5rem;
    background-color: rgba(var(--color-tertiary-rgb), 0.5);
    color: rgb(var(--color-text-secondary-rgb));
    border: none;
    cursor: pointer;
    opacity: 0;
    transition: all 0.3s ease;
    backdrop-filter: blur(4px);
    z-index: 10;
  }
  .copy-button:hover {
    background-color: rgb(var(--color-tertiary-rgb));
    color: rgb(var(--color-accent-rgb));
    transform: scale(1.1);
    box-shadow: 0 2px 8px rgba(var(--color-accent-rgb), 0.3);
  }
</style>