import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // REST и WebSocket идут через origin frontend, поэтому cookie работает без отдельной настройки CORS.
    proxy: { '/api': { target: 'http://localhost:8080', ws: true } },
  },
});
