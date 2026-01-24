export function autosize(node: HTMLTextAreaElement) {
	const minHeight = node.style.height || 'auto';
	function resize() {
		node.style.height = 'auto';
		const newHeight = Math.max(node.scrollHeight, 44);
		node.style.height = `${newHeight}px`;
		if (!node.value.trim()) {
			node.style.height = minHeight;
		}
	}
	node.addEventListener('input', resize);
	const observer = new MutationObserver(() => {
		resize();
	});
	observer.observe(node, {
		attributes: true,
		attributeFilter: ['value']
	});
	let lastValue = node.value;
	const checkValue = () => {
		if (node.value !== lastValue) {
			lastValue = node.value;
			resize();
		}
	};
	const intervalId = setInterval(checkValue, 100);
	resize();
	return {
		destroy() {
			node.removeEventListener('input', resize);
			observer.disconnect();
			clearInterval(intervalId);
		}
	};
}
