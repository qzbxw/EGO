import adapter from '@sveltejs/adapter-vercel';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		runes: true
	},
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			runtime: 'nodejs22.x',
			split: true,
			isr: {
				expiration: false 
			}
		}),
		prerender: {
			entries: [
				'/login',
				'/register',
				'/terms',
				'/privacy',
				'/'
			],
			crawl: true,
			handleHttpError: 'warn'
		},
		inlineStyleThreshold: 2048
	}
};

export default config;