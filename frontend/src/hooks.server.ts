import type { Handle } from '@sveltejs/kit';
export const handle: Handle = async ({ event, resolve }) => {
	const response = await resolve(event, {
		filterSerializedResponseHeaders: () => true
	});
	const headers = new Headers(response.headers);
	headers.set('Cross-Origin-Opener-Policy', 'same-origin-allow-popups');
	headers.set('Cross-Origin-Embedder-Policy', 'unsafe-none');
	return new Response(response.body, {
		status: response.status,
		headers
	});
};
