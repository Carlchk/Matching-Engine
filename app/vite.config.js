import { defineConfig } from 'vite';
import solidPlugin from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    port: 3000,
    strictPort: true,
    host: true, // needed for the Docker Container port mapping to work

  },
  build: {
    target: 'esnext',
  },
});
