<script lang="ts">
	import { Copy, Check, Maximize2, Play } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import { canvasStore } from '$lib/stores/canvas.svelte.ts';
	
	interface Props {
		code: string;
		language?: string;
		inCanvas?: boolean;
	}
	let { code, language = '', inCanvas = false }: Props = $props();
	let copied = $state(false);
	let copyTimeout: number;
	const dispatch = createEventDispatcher();

	let isPreviewable = $derived(['html', 'svg'].includes(language.toLowerCase()));

	function openInCanvas() {
		canvasStore.open(code, language, isPreviewable ? 'Preview' : 'Code');
	}

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
			js: 'JavaScript',
			javascript: 'JavaScript',
			ts: 'TypeScript',
			typescript: 'TypeScript',
			py: 'Python',
			python: 'Python',
			go: 'Go',
			rust: 'Rust',
			java: 'Java',
			cpp: 'C++',
			c: 'C',
			html: 'HTML',
			css: 'CSS',
			scss: 'SCSS',
			json: 'JSON',
			yaml: 'YAML',
			yml: 'YAML',
			xml: 'XML',
			sql: 'SQL',
			bash: 'Bash',
			sh: 'Shell',
			dockerfile: 'Dockerfile',
			md: 'Markdown',
			markdown: 'Markdown'
		};
		return languageMap[lang.toLowerCase()] || lang.toUpperCase();
	}
</script>

<div class="code-wrapper group" data-language={getLanguageDisplay(language)}>
	<pre><code class="hljs language-{language}">{code}</code></pre>
	
	<div class="absolute top-3 right-3 flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-10">
		{#if !inCanvas}
			<button
				class="action-button flex items-center gap-2 pr-3"
				onclick={openInCanvas}
				title={isPreviewable ? 'Preview in Canvas' : 'Open in Canvas'}
			>
				{#if isPreviewable}
					<Play class="h-4 w-4 text-emerald-400" />
					<span class="text-xs font-medium text-emerald-100">Preview</span>
				{:else}
					<Maximize2 class="h-4 w-4 text-blue-400" />
					<span class="text-xs font-medium text-blue-100">Canvas</span>
				{/if}
			</button>
		{/if}

		<button
			class="action-button"
			onclick={copyCode}
			title={copied ? 'Copied!' : 'Copy code'}
			aria-label={copied ? 'Code copied to clipboard' : 'Copy code to clipboard'}
		>
			{#if copied}
				<Check class="h-4 w-4" />
			{:else}
				<Copy class="h-4 w-4" />
			{/if}
		</button>
	</div>
</div>

<style>
	.code-wrapper {
		position: relative;
		margin: 1rem 0;
	}
	/* Removed explicit hover logic for child since we handle it on the container div */
	
	.action-button {
		padding: 0.5rem;
		border-radius: 0.5rem;
		background-color: rgba(var(--color-tertiary-rgb), 0.6);
		color: rgb(var(--color-text-secondary-rgb));
		border: 1px solid rgba(255, 255, 255, 0.1);
		cursor: pointer;
		transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
		backdrop-filter: blur(8px);
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.action-button:hover {
		background-color: rgba(var(--color-tertiary-rgb), 0.9);
		color: rgb(var(--color-text-primary-rgb));
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
		border-color: rgba(255, 255, 255, 0.2);
	}
	.action-button:active {
		transform: translateY(0);
	}
</style>
