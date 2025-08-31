import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import process from 'node:process';

// Allow overriding backend targets in containerized dev via env
const ENV = process.env ?? {};
// Default to Docker service names to work inside docker-compose; override locally via env
const BACKEND_TARGET: string = ENV.VITE_DEV_PROXY_TARGET || 'http://go-api:8080';
const BACKEND_WS_TARGET: string = BACKEND_TARGET.startsWith('https')
    ? BACKEND_TARGET.replace(/^https/, 'wss')
    : BACKEND_TARGET.replace(/^http/, 'ws');
const PY_TARGET: string = ENV.VITE_DEV_PY_TARGET || 'http://python-api:8000';

export default defineConfig({
    plugins: [sveltekit()],
    
        server: {
            host: true,
            port: 5173,
            strictPort: true,
            hmr: {
                protocol: 'ws',
                host: 'localhost',
                clientPort: 5173
            },
            proxy: {
                // Go backend (generic API prefix used by app code)
                '/api': {
                    target: BACKEND_TARGET,
                    changeOrigin: true,
                    secure: false,
                    ws: true
                },
                // Direct auth endpoints; use trailing slash to avoid matching '/authenticating'
                '/auth/': {
                    target: BACKEND_TARGET,
                    changeOrigin: true,
                    secure: false
                },
                '/me': {
                    target: BACKEND_TARGET,
                    changeOrigin: true,
                    secure: false
                },
                '/sessions': {
                    target: BACKEND_TARGET,
                    changeOrigin: true,
                    secure: false
                },
                // WebSocket passthrough to Go backend if needed
                '/api/ws': {
                    target: BACKEND_WS_TARGET,
                    ws: true,
                    changeOrigin: true,
                    secure: false
                },
                // Python backend helpers
                '/py': {
                    target: PY_TARGET,
                    changeOrigin: true,
                    secure: false,
                    ws: false,
                    rewrite: (path: string) => path.replace(/^\/py/, '')
                }
            }
        }
});