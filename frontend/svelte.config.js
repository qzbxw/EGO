import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter(),
		prerender: {
			entries: ['/login', '/register', '/terms', '/privacy', '/'],
			crawl: true,
			handleHttpError: 'warn'
		},
		inlineStyleThreshold: 2048
	}
};

export default config;
