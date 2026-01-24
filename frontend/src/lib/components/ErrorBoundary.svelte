<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import { _ } from 'svelte-i18n';

	let {
		children,
		fallback = null,
		onError = null,
		showDetails = false
	}: {
		children?: import('svelte').Snippet;
		fallback?: import('svelte').Snippet<[Error]>;
		onError?: ((error: Error, errorInfo: { componentStack?: string }) => void) | null;
		showDetails?: boolean;
	} = $props();

	let hasError = $state(false);
	let error = $state<Error | null>(null);
	let errorInfo = $state<{ componentStack?: string }>({});

	function handleError(event: ErrorEvent) {
		const err = event.error || new Error(event.message);

		hasError = true;
		error = err;
		errorInfo = {
			componentStack: err.stack
		};

		// Call custom error handler
		if (onError) {
			onError(err, errorInfo);
		}

		// Prevent default error handling
		event.preventDefault();
	}

	function handleUnhandledRejection(event: PromiseRejectionEvent) {
		const err = event.reason instanceof Error ? event.reason : new Error(String(event.reason));

		hasError = true;
		error = err;
		errorInfo = {
			componentStack: err.stack
		};

		// Call custom error handler
		if (onError) {
			onError(err, errorInfo);
		}
	}

	function resetError() {
		hasError = false;
		error = null;
		errorInfo = {};
	}

	onMount(() => {
		if (browser) {
			window.addEventListener('error', handleError);
			window.addEventListener('unhandledrejection', handleUnhandledRejection);
		}
	});

	onDestroy(() => {
		if (browser) {
			window.removeEventListener('error', handleError);
			window.removeEventListener('unhandledrejection', handleUnhandledRejection);
		}
	});
</script>

{#if hasError && error}
	{#if fallback}
		{@render fallback?.(error)}
	{:else}
		<div class="flex min-h-[400px] flex-col items-center justify-center p-8 text-center">
			<div
				class="w-full max-w-md rounded-lg border-2 border-red-200 bg-red-50 p-6 dark:border-red-800 dark:bg-red-900/20"
			>
				<div class="mb-4 text-red-600 dark:text-red-400">
					<svg class="mx-auto h-16 w-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				</div>

				<h2 class="mb-2 text-xl font-bold text-red-900 dark:text-red-100">
					{$_('errors.something_went_wrong') || 'Something went wrong'}
				</h2>

				<p class="mb-4 text-sm text-red-800 dark:text-red-200">
					{$_('errors.component_error') ||
						'This component encountered an error. Please try refreshing the page.'}
				</p>

				{#if showDetails && error.message}
					<details class="mt-4 text-left">
						<summary class="mb-2 cursor-pointer text-sm font-medium text-red-700 dark:text-red-300">
							{$_('errors.technical_details') || 'Technical Details'}
						</summary>
						<pre
							class="overflow-x-auto rounded bg-red-100 p-3 text-xs text-red-900 dark:bg-red-950 dark:text-red-100">{error.message}{#if errorInfo.componentStack}
								{errorInfo.componentStack}{/if}</pre>
					</details>
				{/if}

				<div class="mt-6 flex gap-2">
					<button
						onclick={resetError}
						class="flex-1 rounded-lg bg-red-600 px-4 py-2 text-white transition-colors duration-200 hover:bg-red-700"
					>
						{$_('errors.try_again') || 'Try Again'}
					</button>
					{#if browser}
						<button
							onclick={() => window.location.reload()}
							class="flex-1 rounded-lg bg-gray-600 px-4 py-2 text-white transition-colors duration-200 hover:bg-gray-700"
						>
							{$_('errors.reload_page') || 'Reload Page'}
						</button>
					{/if}
				</div>
			</div>
		</div>
	{/if}
{:else}
	{@render children?.()}
{/if}
