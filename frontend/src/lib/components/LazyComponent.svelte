<script lang="ts">
	import { onMount } from 'svelte';
	import { createObserver, lazyLoadConfig } from '$lib/utils/intersectionObserver';

	let {
		component = undefined,
		importFn = undefined,
		fallback = undefined,
		error = undefined,
		delay = 0,
		eagerLoad = false,
		preload = false,
		...restProps
	}: {
		component?: any;
		importFn?: () => Promise<{ default: any }>;
		fallback?: any;
		error?: any;
		delay?: number;
		eagerLoad?: boolean;
		preload?: boolean;
		[key: string]: any;
	} = $props();

	let loaded = $state(false);
	let visible = $state(eagerLoad);
	let loadError = $state<Error | null>(null);
	let container: HTMLDivElement | undefined = $state();
	let loadingComponent = $state(true);
	let inProgress = $state(false);

	async function loadComponent() {
		if (loaded || inProgress) return;
		inProgress = true;

		try {
			if (delay > 0) {
				await new Promise((resolve) => setTimeout(resolve, delay));
			}

			if (importFn) {
				const mod = await importFn();
				component = mod.default;
			}

			loaded = true;
			loadingComponent = false;
		} catch (err) {
			loadError = err instanceof Error ? err : new Error(String(err));
			loadingComponent = false;
			console.error('[LazyComponent] Load error:', loadError);
		} finally {
			inProgress = false;
		}
	}

	onMount(() => {
		if (eagerLoad) {
			loadComponent();
			return;
		}

		if (!container || !importFn) return;

		// Setup intersection observer для lazy load
		const disconnect = createObserver(
			container,
			(isVisible) => {
				if (isVisible && !loaded && !inProgress) {
					loadComponent();
				}
			},
			preload ? lazyLoadConfig.normal : lazyLoadConfig.lazy
		);

		return disconnect;
	});

	// Preload при монтировании если нужно
	$effect(() => {
		if (preload && importFn && !loaded && !inProgress && !eagerLoad) {
			// Небольшая задержка перед preload
			const timer = setTimeout(() => {
				importFn?.().catch(() => {});
			}, 100);

			return () => clearTimeout(timer);
		}
	});
</script>

<div bind:this={container} class="lazy-component-wrapper">
	{#if loadError && error}
		{@const ErrorComponent = error}
		<ErrorComponent {loadError} />
	{:else if loadingComponent && fallback}
		{@const FallbackComponent = fallback}
		<FallbackComponent />
	{:else if component}
		{@const Component = component}
		<Component {...restProps} />
	{/if}
</div>

<style>
	.lazy-component-wrapper {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
	}
</style>
