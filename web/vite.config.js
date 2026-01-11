import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 3000,
		proxy: {
			'/api': {
				target: process.env.API_URL || 'http://localhost:8080',
				changeOrigin: true
			}
		}
	}
});
