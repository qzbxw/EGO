import type { RequestHandler } from '@sveltejs/kit';
export const GET: RequestHandler = async () => {
	return new Response(null, {
		status: 302,
		headers: {
			Location: '/logodark.png',
			'Cache-Control': 'public, max-age=86400'
		}
	});
};
