import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // host: true so a phone on the same Wi-Fi can open the dev server.
  server: { port: 5174, host: true },
})
