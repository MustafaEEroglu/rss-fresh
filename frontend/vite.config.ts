import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
  },
  plugins: [
    svelte(),
    tailwindcss(),
    VitePWA({      registerType: 'autoUpdate',
      injectRegister: 'auto',
      includeAssets: ['favicon.svg', 'apple-touch-icon.svg'],
        manifest: {
        name: 'RSS-Fresh',
        short_name: 'RSS-Fresh',
        description: 'Personal lightweight RSS / news reader',
        theme_color: '#0f172a',
        background_color: '#0f172a',
        display: 'standalone',
        start_url: '/',
        scope: '/',
        icons: [
          { src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' },
          { src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'maskable' },
          // TODO: replace apple-touch-icon.svg with a real 180×180 PNG for iOS home screen.
          { src: '/apple-touch-icon.svg', sizes: '180x180', type: 'image/svg+xml', purpose: 'any' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2}'],
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//],
        runtimeCaching: [
          {
            // GET reads — NetworkFirst with 3s timeout. Background write-through to
            // Dexie happens in the app code, not here.
            urlPattern: ({ url, request }) =>
              url.pathname.startsWith('/api/v1/') && request.method === 'GET',
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-get',
              networkTimeoutSeconds: 3,
              expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Mutations — NetworkOnly with Background Sync, queued offline.
            urlPattern: ({ url, request }) =>
              url.pathname.startsWith('/api/v1/') &&
              ['POST', 'PATCH', 'DELETE'].includes(request.method),
            handler: 'NetworkOnly',
            method: 'POST',
            options: {
              backgroundSync: {
                name: 'rss-fresh-mutations',
                options: { maxRetentionTime: 24 * 60 },
              },
            },
          },
          {
            // Article-body images: SWR with bounded LRU.
            urlPattern: ({ request }) => request.destination === 'image',
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'images',
              expiration: { maxEntries: 100, maxAgeSeconds: 60 * 60 * 24 * 30 },
            },
          },
        ],
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:3000',
    },
  },
  build: {
    target: 'es2020',
    sourcemap: false,
    cssCodeSplit: true,
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        manualChunks: {
          // Dexie (~150 KB min) in its own content-hashed chunk — app-code
          // changes won't bust the Dexie cache entry in the SW or the browser.
          dexie: ['dexie'],
        },
      },
    },
  },
});
