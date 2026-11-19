import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.js',
      injectManifest: {
        // Das fontes, só o alfabeto latino (com acentos) vai para o offline; os outros alfabetos
        // continuam carregando sob demanda, se aparecerem.
        globPatterns: ['**/*.{js,css,html,png,svg}', '**/*-latin-*.woff2'],
      },
      manifest: {
        name: 'Trilha de Estudo',
        short_name: 'Trilha',
        lang: 'pt-BR',
        theme_color: '#151515',
        background_color: '#151515',
        display: 'standalone',
        start_url: '/',
        // Compartilhar do Android (T36): o link do YouTube vai direto para a Entrada.
        share_target: { action: '/compartilhar', method: 'GET', params: { title: 'title', text: 'text', url: 'url' } },
        icons: [
          { src: 'icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
    }),
  ],
  server: {
    proxy: { '/api': process.env.API_URL || 'http://localhost:8080' },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.js'],
  },
})
