import type { PageLoad } from './$types';
import { browser } from '$app/environment';
import { auth } from '$lib/stores/auth.svelte';
import { redirect } from '@sveltejs/kit';
export const load: PageLoad = async () => {
	if (browser && auth.accessToken) {
		throw redirect(307, '/chat');
	}
	return {};
};
