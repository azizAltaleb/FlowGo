import path from "path"
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return
          }
          if (id.includes("@xyflow/react")) {
            return "vendor-flow"
          }
          if (id.includes("recharts")) {
            return "vendor-charts"
          }
          if (id.includes("react-oidc-context") || id.includes("oidc-client-ts")) {
            return "vendor-auth"
          }
          if (id.includes("react") || id.includes("react-dom") || id.includes("react-router-dom")) {
            return "vendor-react"
          }
        },
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api/query": {
        target: process.env.VITE_QUERY_TARGET || "http://localhost:8081",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/query/, ""),
      },
      "/api": {
        target: process.env.VITE_API_TARGET || "http://localhost:8080",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ""),
      },
    },
  },
})
