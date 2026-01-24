<script lang="ts">
	import { marked } from 'marked';
	import hljs from 'highlight.js';
	import 'highlight.js/styles/github-dark.css';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
	import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	hljs.configure({
		languages: [
			'javascript',
			'typescript',
			'python',
			'java',
			'cpp',
			'c',
			'csharp',
			'go',
			'rust',
			'php',
			'ruby',
			'swift',
			'kotlin',
			'scala',
			'html',
			'css',
			'scss',
			'json',
			'xml',
			'yaml',
			'markdown',
			'bash',
			'shell',
			'sql',
			'dockerfile',
			'nginx',
			'apache'
		]
	});
	try {
		marked.setOptions({ gfm: true, breaks: true });
	} catch {}
	let { text, invert = false } = $props<{ text: string; invert?: boolean }>();
	let container: HTMLDivElement | undefined = $state();
	function getLanguage(className: string): string | null {
		const match = /language-(\w+)/.exec(className || '');
		return match ? match[1] : null;
	}
	function copyCode(event: MouseEvent) {
		const button = event.currentTarget as HTMLButtonElement;
		const pre = button.closest('.code-wrapper')?.querySelector('pre');
		const codeElement = pre?.querySelector('code');
		if (codeElement?.textContent) {
			navigator.clipboard
				.writeText(codeElement.textContent)
				.then(() => {
					const icon = button.querySelector('svg');
					const originalIconHTML = icon?.outerHTML;
					if (!icon || !originalIconHTML) return;
					button.innerHTML =
						'<svg width="16" height="16" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"></path></svg>';
					button.disabled = true;
					setTimeout(() => {
						button.innerHTML = originalIconHTML;
						button.disabled = false;
					}, 2000);
				})
				.catch((err) => {
					toast.error($_('toasts.copy_response_error'));
					console.error('Copy to clipboard failed: ', err);
				});
		}
	}
	async function renderContent() {
		if (!container) return;
		if (!text) {
			container.innerHTML = '';
			return;
		}
		const html = await marked(text, { async: true });
		if (!container) return;
		container.innerHTML = html;
		container.querySelectorAll('table').forEach((table) => {
			const wrap = document.createElement('div');
			wrap.className = 'overflow-x-auto my-2 rounded-lg border border-tertiary/40';
			table.classList.add('min-w-full', 'table-auto');
			(table as HTMLElement).style.wordBreak = 'break-word';
			table.querySelectorAll('th, td').forEach((cell) => {
				(cell as HTMLElement).style.wordBreak = 'break-word';
				(cell as HTMLElement).style.whiteSpace = 'normal';
			});
			table.parentNode?.insertBefore(wrap, table);
			wrap.appendChild(table);
		});
		container.querySelectorAll('pre').forEach((pre) => {
			if (pre.parentElement?.classList.contains('code-wrapper')) {
				return;
			}
			const code = pre.querySelector('code');
			if (!code) return;
			const langName = getLanguage(code.className);
			if (langName && hljs.getLanguage(langName)) {
				try {
					const result = hljs.highlight(code.textContent || '', { language: langName });
					code.innerHTML = result.value;
					code.classList.add('hljs');
				} catch (e) {
					hljs.highlightElement(code as HTMLElement);
				}
			} else {
				hljs.highlightElement(code as HTMLElement);
			}
			const wrapper = document.createElement('div');
			wrapper.className =
				'code-wrapper group my-4 overflow-hidden rounded-xl border border-tertiary/60 bg-secondary/95 shadow-lg shadow-black/30 backdrop-blur-sm';
			const footer = document.createElement('div');
			footer.className =
				'code-footer flex items-center justify-between border-t border-tertiary/60 bg-secondary/90 px-3 py-2 text-[11px] text-text-secondary/80';
			const leftPart = document.createElement('div');
			leftPart.className = 'footer-left flex items-center gap-2';
			const copyButton = document.createElement('button');
			copyButton.title = 'Copy code';
			copyButton.innerHTML =
				'<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
			copyButton.className =
				'inline-flex h-7 w-7 items-center justify-center rounded-md border border-transparent bg-tertiary/60 text-text-secondary transition-colors hover:border-accent/70 hover:text-text-primary';
			copyButton.onclick = copyCode;
			copyButton.setAttribute('aria-label', 'Copy code');
			const warningSpan = document.createElement('span');
			warningSpan.innerText = 'Use code with caution.';
			warningSpan.className = 'hidden text-[11px] text-text-secondary/80 sm:inline';
			leftPart.appendChild(copyButton);
			leftPart.appendChild(warningSpan);
			const rightPart = document.createElement('div');
			rightPart.className =
				'footer-right text-[11px] uppercase tracking-wide text-text-secondary/70';
			rightPart.innerText = langName || '';
			footer.appendChild(leftPart);
			footer.appendChild(rightPart);
			pre.parentNode?.insertBefore(wrapper, pre);
			pre.style.overflowX = 'auto';
			(
				pre.style as CSSStyleDeclaration & { webkitOverflowScrolling: string }
			).webkitOverflowScrolling = 'touch';
			pre.style.wordBreak = 'break-word';
			wrapper.appendChild(pre);
			wrapper.appendChild(footer);
		});
		if (window.renderMathInElement) {
			window.renderMathInElement(container, {
				delimiters: [
					{ left: '$$', right: '$$', display: true },
					{ left: '$', right: '$', display: false }
				],
				throwOnError: false
			});
		}
	}
	let __renderTimer: number | null = null;
	$effect(() => {
		void text; // track text dependency
		if (__renderTimer) clearTimeout(__renderTimer);
		__renderTimer = window.setTimeout(() => {
			renderContent();
		}, 60);
		return () => {
			if (__renderTimer) clearTimeout(__renderTimer);
		};
	});
</script>

<div
	bind:this={container}
	class="prose w-full min-w-0 max-w-full break-words selection:bg-accent/30 selection:text-text-primary"
	class:prose-invert={invert || preferencesStore.theme === 'dark'}
></div>

<style>
	:global(.prose) {
		user-select: text !important;
		-webkit-user-select: text !important;
		font-variant-numeric: tabular-nums;
		line-height: 1.6;
	}
	:global(.prose p) {
		margin-top: 0.75rem;
		margin-bottom: 0.75rem;
	}
	:global(.prose p:first-child) {
		margin-top: 0;
	}
	:global(.prose p:last-child) {
		margin-bottom: 0;
	}
	:global(.prose table) {
		border-collapse: collapse;
		margin: 1rem 0;
	}
	:global(.prose table th),
	:global(.prose table td) {
		border: 1px solid color-mix(in oklab, currentColor 20%, transparent);
		padding: 0.5rem 0.75rem;
		vertical-align: top;
	}
	:global(.prose ul, .prose ol) {
		margin-top: 0.5rem;
		margin-bottom: 0.5rem;
		padding-left: 1.5rem;
	}
	:global(.prose li) {
		margin-top: 0.25rem;
		margin-bottom: 0.25rem;
	}
</style>
