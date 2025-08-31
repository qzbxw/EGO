<script lang="ts">
	import { marked } from 'marked';
	import hljs from 'highlight.js';
	import 'highlight.js/styles/github-dark.css';
	import { toast } from 'svelte-sonner';
	import { _ } from 'svelte-i18n';
    import { preferencesStore } from '$lib/stores/preferences.svelte.ts';
	hljs.configure({
		languages: ['javascript', 'typescript', 'python', 'java', 'cpp', 'c', 'csharp', 'go', 'rust', 'php', 'ruby', 'swift', 'kotlin', 'scala', 'html', 'css', 'scss', 'json', 'xml', 'yaml', 'markdown', 'bash', 'shell', 'sql', 'dockerfile', 'nginx', 'apache']
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
			wrapper.className = 'code-wrapper group';
            const footer = document.createElement('div');
			footer.className = 'code-footer';
			const leftPart = document.createElement('div');
			leftPart.className = 'footer-left';
            const copyButton = document.createElement('button');
            copyButton.title = 'Copy code';
			copyButton.innerHTML =
				'<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
			copyButton.onclick = copyCode;
            copyButton.setAttribute('aria-label', 'Copy code');
            const warningSpan = document.createElement('span');
            warningSpan.innerText = 'Use code with caution.';
            warningSpan.className = 'hidden sm:inline';
			leftPart.appendChild(copyButton);
			leftPart.appendChild(warningSpan);
			const rightPart = document.createElement('div');
			rightPart.className = 'footer-right';
			rightPart.innerText = langName || '';
			footer.appendChild(leftPart);
			footer.appendChild(rightPart);
			pre.parentNode?.insertBefore(wrapper, pre);
            pre.style.overflowX = 'auto';
            (pre.style as CSSStyleDeclaration & { webkitOverflowScrolling: string }).webkitOverflowScrolling = 'touch';
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
		if (__renderTimer) clearTimeout(__renderTimer);
		__renderTimer = window.setTimeout(() => {
			renderContent();
		}, 60);
		return () => {
			if (__renderTimer) clearTimeout(__renderTimer);
		};
	});
</script>
<style>
  :global(.prose table) {
    border-collapse: collapse;
  }
  :global(.prose table th),
  :global(.prose table td) {
    border: 1px solid color-mix(in oklab, currentColor 20%, transparent);
    padding: 0.4rem 0.6rem;
    vertical-align: top;
  }
</style>
<div
  bind:this={container}
  class="prose w-full min-w-0 max-w-full break-words"
  class:prose-invert={invert || preferencesStore.theme === 'dark'}
></div>