<script lang="ts" generics="T extends { id?: string | number }">
	import { onMount, tick, type Snippet } from 'svelte';
	import { SvelteMap } from 'svelte/reactivity';

	let {
		items = [],
		estimatedItemHeight = 100,
		overscan = 3,
		children
	}: {
		items: T[];
		estimatedItemHeight?: number;
		overscan?: number;
		children: Snippet<[T, number]>;
	} = $props();

	let scrollContainer: HTMLDivElement | undefined = $state();
	let scrollTop = $state(0);
	let containerHeight = $state(0);
	let itemHeights = new SvelteMap<number, number>();
	let totalHeight = $state(0);
	let isScrolling = $state(false);
	let scrollTimeout: ReturnType<typeof setTimeout> | null = null;

	// Оптимизированный расчет диапазона видимых элементов
	let visibleRange = $derived.by(() => {
		if (!containerHeight) return { start: 0, end: items.length };

		let cumulativeHeight = 0;
		let start = items.length;
		let end = 0;

		for (let i = 0; i < items.length; i++) {
			const itemHeight = itemHeights.get(i) || estimatedItemHeight;

			if (cumulativeHeight + itemHeight >= scrollTop && start === items.length) {
				start = Math.max(0, i - overscan);
			}

			if (cumulativeHeight >= scrollTop + containerHeight && end === 0) {
				end = Math.min(items.length, i + overscan + 1);
				break;
			}

			cumulativeHeight += itemHeight;
		}

		if (end === 0) end = items.length;

		return { start, end };
	});

	let visibleItems = $derived(items.slice(visibleRange.start, visibleRange.end));

	// Вычисляем смещение для видимых элементов
	let offsetY = $derived.by(() => {
		let offset = 0;
		for (let i = 0; i < visibleRange.start; i++) {
			offset += itemHeights.get(i) || estimatedItemHeight;
		}
		return offset;
	});

	// Обновляем общую высоту
	$effect(() => {
		let height = 0;
		for (let i = 0; i < items.length; i++) {
			height += itemHeights.get(i) || estimatedItemHeight;
		}
		totalHeight = height;
	});

	// Измеряем высоту элементов
	function measureItems() {
		if (!scrollContainer) return;

		const itemElements = scrollContainer.querySelectorAll('[data-virtual-index]');
		itemElements.forEach((el) => {
			const index = parseInt(el.getAttribute('data-virtual-index') || '0', 10);
			const height = el.getBoundingClientRect().height;
			if (height > 0 && itemHeights.get(index) !== height) {
				itemHeights.set(index, height);
			}
		});
	}

	// Оптимизированный обработчик скролла с debounce
	function handleScroll(e: Event) {
		if (e.target instanceof HTMLElement) {
			const newScrollTop = e.target.scrollTop;
			scrollTop = newScrollTop;

			// Флаг для состояния скролла
			isScrolling = true;
			if (scrollTimeout) clearTimeout(scrollTimeout);
			scrollTimeout = setTimeout(() => {
				isScrolling = false;
			}, 150);
		}

		// Измеряем только если не скроллим (или с задержкой)
		if (!isScrolling) {
			measureItems();
		}
	}

	// Измеряем элементы после рендера
	$effect(() => {
		if (visibleItems.length > 0) {
			tick().then(() => {
				measureItems();
			});
		}
	});

	onMount(() => {
		if (!scrollContainer) return;

		// Обновляем высоту контейнера
		const resizeObserver = new ResizeObserver((entries) => {
			for (const entry of entries) {
				containerHeight = entry.contentRect.height;
			}
		});

		resizeObserver.observe(scrollContainer);

		// Passive event listener для лучшей производительности скролла
		const passiveOptions = { passive: true };
		scrollContainer.addEventListener('scroll', handleScroll, passiveOptions);

		return () => {
			resizeObserver.disconnect();
			if (scrollTimeout) clearTimeout(scrollTimeout);
			scrollContainer?.removeEventListener(
				'scroll',
				handleScroll,
				passiveOptions as EventListenerOptions
			);
		};
	});

	// Экспортируемые методы
	export function scrollToBottom(behavior: 'smooth' | 'auto' = 'smooth') {
		if (!scrollContainer) return;
		scrollContainer.scrollTo({
			top: totalHeight,
			behavior
		});
	}

	export function isNearBottom(threshold = 200) {
		if (!scrollContainer) return true;
		return totalHeight - scrollTop <= containerHeight + threshold;
	}
</script>

<div bind:this={scrollContainer} class="virtual-list-container" class:is-scrolling={isScrolling}>
	<div class="virtual-list-spacer" style="height: {totalHeight}px">
		<div class="virtual-list-content" style="transform: translateY({offsetY}px)">
			{#each visibleItems as item, idx (item.id || visibleRange.start + idx)}
				<div data-virtual-index={visibleRange.start + idx}>
					{@render children(item, visibleRange.start + idx)}
				</div>
			{/each}
		</div>
	</div>
</div>

<style>
	.virtual-list-container {
		overflow-y: auto;
		overflow-x: hidden;
		height: 100%;
		width: 100%;
		position: relative;
		/* Оптимизация для скролла */
		-webkit-overflow-scrolling: touch;
	}

	.virtual-list-container.is-scrolling {
		/* Отключаем дорогие операции во время скролла */
		scroll-behavior: auto;
	}

	.virtual-list-spacer {
		position: relative;
		width: 100%;
		contain: layout style paint;
	}

	.virtual-list-content {
		will-change: transform;
		width: 100%;
		/* GPU acceleration */
		transform: translateZ(0);
		backface-visibility: hidden;
		perspective: 1000px;
	}

	/* Smoother scrolling on mobile */
	@supports (scroll-behavior: smooth) {
		.virtual-list-container {
			scroll-behavior: smooth;
		}
	}
</style>
