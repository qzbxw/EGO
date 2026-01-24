import { browser } from '$app/environment';
import { toast } from 'svelte-sonner';
export function initializeUpdateManager() {
	if (!browser || !('serviceWorker' in navigator)) {
		return;
	}
	navigator.serviceWorker.ready.then((registration) => {
		if (registration.waiting) {
			showUpdateToast(registration.waiting);
		}
		registration.addEventListener('updatefound', () => {
			const newWorker = registration.installing;
			if (newWorker) {
				newWorker.addEventListener('statechange', () => {
					if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
						showUpdateToast(newWorker);
					}
				});
			}
		});
	});
}
function showUpdateToast(worker: ServiceWorker) {
	toast.info('Доступно обновление!', {
		duration: Infinity,
		action: {
			label: 'Обновить',
			onClick: () => {
				worker.postMessage({ type: 'SKIP_WAITING' });
				window.location.reload();
			}
		},
		cancel: {
			label: 'Позже',
			onClick: () => {}
		}
	});
}
