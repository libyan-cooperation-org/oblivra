import { defineConfig } from 'vite';
import solidPlugin from 'vite-plugin-solid';
import { resolve } from 'path';

export default defineConfig({
  plugins: [solidPlugin()],

  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      '@components': resolve(__dirname, 'src/components'),
      '@core': resolve(__dirname, 'src/core'),
      '@pages': resolve(__dirname, 'src/pages'),
      '@styles': resolve(__dirname, 'src/styles'),
      '@assets': resolve(__dirname, 'src/assets'),
    },
  },

  build: {
    target: 'esnext',
    outDir: 'dist',
    minify: 'esbuild',
    sourcemap: false,
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('solid-js')) return 'vendor-solid';
            if (id.includes('@xterm')) return 'vendor-xterm';
            if (id.includes('echarts')) return 'vendor-charts';
            return 'vendor';
          }
        },
      },
    },
  },

  server: {
    port: 3000,
    strictPort: true,
  },
});
