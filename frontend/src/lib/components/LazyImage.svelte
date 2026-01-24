<script lang="ts">
	import { onMount } from 'svelte';

	let {
		src,
		alt = '',
		class: className = '',
		placeholder = '',
		blurhash = '',
		sizes = '',
		srcSet = '',
		width = '',
		height = '',
		quality = 'high',
		decoding = 'async',
		loading = 'lazy'
	}: {
		src: string;
		alt?: string;
		class?: string;
		placeholder?: string;
		blurhash?: string;
		sizes?: string;
		srcSet?: string;
		width?: string;
		height?: string;
		quality?: 'low' | 'medium' | 'high';
		decoding?: 'sync' | 'async' | 'auto';
		loading?: 'lazy' | 'eager';
	} = $props();

	let containerElement: HTMLDivElement | undefined = $state();
	let imageElement: HTMLImageElement | undefined = $state();
	let loaded = $state(false);
	let visible = $state(false);
	let error = $state(false);
	let retryCount = $state(0);
	const maxRetries = 2;

	// Генерируем srcset для responsive images
	// Note: We only append query params if the src is not a data/blob URL
	function generateSrcSet(baseSrc: string): string {
		if (srcSet) return srcSet;
		if (baseSrc.startsWith('data:') || baseSrc.startsWith('blob:')) return '';

		const widths = [320, 640, 1024, 1536, 2048];
		const qValue = quality === 'high' ? 90 : quality === 'medium' ? 75 : 60;
		return widths.map((w) => `${baseSrc}?w=${w}&q=${qValue} ${w}w`).join(', ');
	}

	onMount(() => {
		// If eager, show immediately
		if (loading === 'eager') {
			visible = true;
			return;
		}

		if (!containerElement) return;

		// Intersection Observer для lazy loading
		const observer = new IntersectionObserver(
			(entries) => {
				entries.forEach((entry) => {
					if (entry.isIntersecting) {
						visible = true;
						observer.disconnect();
					}
				});
			},
			{
				rootMargin: '100px',
				threshold: 0.01
			}
		);

		observer.observe(containerElement);

		return () => {
			observer.disconnect();
		};
	});

	function handleLoad() {
		loaded = true;
		retryCount = 0;
	}

	function handleError() {
		error = true;
		if (retryCount < maxRetries) {
			retryCount++;
			visible = false;
			setTimeout(() => {
				visible = true;
				loaded = false;
				error = false;
			}, 1000 * retryCount);
		}
		console.error(`[LazyImage] Failed to load image (attempt ${retryCount}/${maxRetries}):`, src);
	}
</script>

<div
	bind:this={containerElement}
	class="lazy-image-container {className}"
	class:loaded
	class:error
	style="width: {width ? width : 'auto'}; height: {height ? height : 'auto'};"
>
	{#if blurhash && !loaded}
		<div
			class="placeholder blur-up"
			style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);"
		>
			<div class="shimmer"></div>
		</div>
	{:else if placeholder && !loaded}
		<div
			class="placeholder blur-placeholder"
			style="background-image: url({placeholder}); background-size: cover; background-position: center;"
		></div>
	{:else if !loaded && !visible}
		<div class="placeholder skeleton">
			<div class="shimmer"></div>
		</div>
	{/if}

	{#if visible || loaded || loading === 'eager'}
		{#if src.startsWith('data:') || src.startsWith('blob:')}
			<img
				bind:this={imageElement}
				{src}
				{alt}
				{width}
				{height}
				{decoding}
				loading={loading === 'eager' ? undefined : 'lazy'}
				class="lazy-image"
				class:fade-in={loaded}
				onload={handleLoad}
				onerror={handleError}
			/>
		{:else}
			<picture>
				<!-- AVIF support (best compression) -->
				<source srcset={generateSrcSet(src.replace(/\.\w+$/, '.avif'))} type="image/avif" {sizes} />
				<!-- WebP fallback -->
				<source srcset={generateSrcSet(src.replace(/\.\w+$/, '.webp'))} type="image/webp" {sizes} />
				<!-- PNG/JPG fallback -->
				<img
					bind:this={imageElement}
					{src}
					srcset={generateSrcSet(src)}
					{sizes}
					{alt}
					{width}
					{height}
					{decoding}
					loading={loading === 'eager' ? undefined : 'lazy'}
					class="lazy-image"
					class:fade-in={loaded}
					onload={handleLoad}
					onerror={handleError}
				/>
			</picture>
		{/if}
	{/if}
</div>

<style>
	.lazy-image-container {
		position: relative;
		overflow: hidden;
		background: var(--color-tertiary, #f0f0f0);
		display: block;
		min-height: 50px; /* Ensure some height before load */
	}

	picture {
		display: block;
		width: 100%;
		height: 100%;
	}

	.placeholder {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		background-size: cover;
		background-position: center;
		filter: blur(20px);
		transform: scale(1.1);
		transition: opacity 0.3s ease;
		z-index: 1;
	}

	.blur-placeholder {
		animation: fadeOut 0.5s ease-in forwards;
	}

	.skeleton {
		background: linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.05) 25%,
			rgba(255, 255, 255, 0.15) 50%,
			rgba(255, 255, 255, 0.05) 75%
		);
		background-size: 200% 100%;
		animation: shimmer 1.5s infinite;
	}

	@keyframes shimmer {
		0% {
			background-position: 200% 0;
		}
		100% {
			background-position: -200% 0;
		}
	}

	@keyframes fadeOut {
		to {
			opacity: 0;
		}
	}

	.shimmer {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		background: linear-gradient(
			90deg,
			transparent 0%,
			rgba(255, 255, 255, 0.2) 50%,
			transparent 100%
		);
		animation: shimmer-move 2s infinite;
	}

	@keyframes shimmer-move {
		0% {
			transform: translateX(-100%);
		}
		100% {
			transform: translateX(100%);
		}
	}

	img.lazy-image {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
		opacity: 0;
		transition: opacity 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94);
		position: relative;
		z-index: 2;
	}

	img.lazy-image.fade-in {
		opacity: 1;
	}

	.loaded .placeholder {
		opacity: 0;
		pointer-events: none;
	}

	.error {
		background: var(--color-secondary, #e0e0e0);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.error::after {
		content: '⚠️';
		font-size: 2rem;
		opacity: 0.5;
	}
</style>
