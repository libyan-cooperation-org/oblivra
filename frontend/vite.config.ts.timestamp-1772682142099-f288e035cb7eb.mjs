// vite.config.ts
import { defineConfig } from "file:///C:/Users/KingKnull/Desktop/sovereign-terminal/frontend/node_modules/vite/dist/node/index.js";
import solidPlugin from "file:///C:/Users/KingKnull/Desktop/sovereign-terminal/frontend/node_modules/vite-plugin-solid/dist/esm/index.mjs";
import { resolve } from "path";
var __vite_injected_original_dirname = "C:\\Users\\KingKnull\\Desktop\\sovereign-terminal\\frontend";
var vite_config_default = defineConfig({
  plugins: [solidPlugin()],
  resolve: {
    alias: {
      "@": resolve(__vite_injected_original_dirname, "src"),
      "@components": resolve(__vite_injected_original_dirname, "src/components"),
      "@core": resolve(__vite_injected_original_dirname, "src/core"),
      "@pages": resolve(__vite_injected_original_dirname, "src/pages"),
      "@styles": resolve(__vite_injected_original_dirname, "src/styles"),
      "@assets": resolve(__vite_injected_original_dirname, "src/assets")
    }
  },
  build: {
    target: "esnext",
    outDir: "dist",
    minify: "esbuild",
    sourcemap: false,
    chunkSizeWarningLimit: 1e3,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules")) {
            if (id.includes("solid-js")) return "vendor-solid";
            if (id.includes("@xterm")) return "vendor-xterm";
            if (id.includes("echarts")) return "vendor-charts";
            return "vendor";
          }
        }
      }
    }
  },
  server: {
    port: 3e3,
    strictPort: true
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCJDOlxcXFxVc2Vyc1xcXFxLaW5nS251bGxcXFxcRGVza3RvcFxcXFxzb3ZlcmVpZ24tdGVybWluYWxcXFxcZnJvbnRlbmRcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIkM6XFxcXFVzZXJzXFxcXEtpbmdLbnVsbFxcXFxEZXNrdG9wXFxcXHNvdmVyZWlnbi10ZXJtaW5hbFxcXFxmcm9udGVuZFxcXFx2aXRlLmNvbmZpZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vQzovVXNlcnMvS2luZ0tudWxsL0Rlc2t0b3Avc292ZXJlaWduLXRlcm1pbmFsL2Zyb250ZW5kL3ZpdGUuY29uZmlnLnRzXCI7aW1wb3J0IHsgZGVmaW5lQ29uZmlnIH0gZnJvbSAndml0ZSc7XHJcbmltcG9ydCBzb2xpZFBsdWdpbiBmcm9tICd2aXRlLXBsdWdpbi1zb2xpZCc7XHJcbmltcG9ydCB7IHJlc29sdmUgfSBmcm9tICdwYXRoJztcclxuXHJcbmV4cG9ydCBkZWZhdWx0IGRlZmluZUNvbmZpZyh7XHJcbiAgcGx1Z2luczogW3NvbGlkUGx1Z2luKCldLFxyXG5cclxuICByZXNvbHZlOiB7XHJcbiAgICBhbGlhczoge1xyXG4gICAgICAnQCc6IHJlc29sdmUoX19kaXJuYW1lLCAnc3JjJyksXHJcbiAgICAgICdAY29tcG9uZW50cyc6IHJlc29sdmUoX19kaXJuYW1lLCAnc3JjL2NvbXBvbmVudHMnKSxcclxuICAgICAgJ0Bjb3JlJzogcmVzb2x2ZShfX2Rpcm5hbWUsICdzcmMvY29yZScpLFxyXG4gICAgICAnQHBhZ2VzJzogcmVzb2x2ZShfX2Rpcm5hbWUsICdzcmMvcGFnZXMnKSxcclxuICAgICAgJ0BzdHlsZXMnOiByZXNvbHZlKF9fZGlybmFtZSwgJ3NyYy9zdHlsZXMnKSxcclxuICAgICAgJ0Bhc3NldHMnOiByZXNvbHZlKF9fZGlybmFtZSwgJ3NyYy9hc3NldHMnKSxcclxuICAgIH0sXHJcbiAgfSxcclxuXHJcbiAgYnVpbGQ6IHtcclxuICAgIHRhcmdldDogJ2VzbmV4dCcsXHJcbiAgICBvdXREaXI6ICdkaXN0JyxcclxuICAgIG1pbmlmeTogJ2VzYnVpbGQnLFxyXG4gICAgc291cmNlbWFwOiBmYWxzZSxcclxuICAgIGNodW5rU2l6ZVdhcm5pbmdMaW1pdDogMTAwMCxcclxuICAgIHJvbGx1cE9wdGlvbnM6IHtcclxuICAgICAgb3V0cHV0OiB7XHJcbiAgICAgICAgbWFudWFsQ2h1bmtzKGlkKSB7XHJcbiAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ25vZGVfbW9kdWxlcycpKSB7XHJcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnc29saWQtanMnKSkgcmV0dXJuICd2ZW5kb3Itc29saWQnO1xyXG4gICAgICAgICAgICBpZiAoaWQuaW5jbHVkZXMoJ0B4dGVybScpKSByZXR1cm4gJ3ZlbmRvci14dGVybSc7XHJcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcygnZWNoYXJ0cycpKSByZXR1cm4gJ3ZlbmRvci1jaGFydHMnO1xyXG4gICAgICAgICAgICByZXR1cm4gJ3ZlbmRvcic7XHJcbiAgICAgICAgICB9XHJcbiAgICAgICAgfSxcclxuICAgICAgfSxcclxuICAgIH0sXHJcbiAgfSxcclxuXHJcbiAgc2VydmVyOiB7XHJcbiAgICBwb3J0OiAzMDAwLFxyXG4gICAgc3RyaWN0UG9ydDogdHJ1ZSxcclxuICB9LFxyXG59KTtcclxuIl0sCiAgIm1hcHBpbmdzIjogIjtBQUFnVyxTQUFTLG9CQUFvQjtBQUM3WCxPQUFPLGlCQUFpQjtBQUN4QixTQUFTLGVBQWU7QUFGeEIsSUFBTSxtQ0FBbUM7QUFJekMsSUFBTyxzQkFBUSxhQUFhO0FBQUEsRUFDMUIsU0FBUyxDQUFDLFlBQVksQ0FBQztBQUFBLEVBRXZCLFNBQVM7QUFBQSxJQUNQLE9BQU87QUFBQSxNQUNMLEtBQUssUUFBUSxrQ0FBVyxLQUFLO0FBQUEsTUFDN0IsZUFBZSxRQUFRLGtDQUFXLGdCQUFnQjtBQUFBLE1BQ2xELFNBQVMsUUFBUSxrQ0FBVyxVQUFVO0FBQUEsTUFDdEMsVUFBVSxRQUFRLGtDQUFXLFdBQVc7QUFBQSxNQUN4QyxXQUFXLFFBQVEsa0NBQVcsWUFBWTtBQUFBLE1BQzFDLFdBQVcsUUFBUSxrQ0FBVyxZQUFZO0FBQUEsSUFDNUM7QUFBQSxFQUNGO0FBQUEsRUFFQSxPQUFPO0FBQUEsSUFDTCxRQUFRO0FBQUEsSUFDUixRQUFRO0FBQUEsSUFDUixRQUFRO0FBQUEsSUFDUixXQUFXO0FBQUEsSUFDWCx1QkFBdUI7QUFBQSxJQUN2QixlQUFlO0FBQUEsTUFDYixRQUFRO0FBQUEsUUFDTixhQUFhLElBQUk7QUFDZixjQUFJLEdBQUcsU0FBUyxjQUFjLEdBQUc7QUFDL0IsZ0JBQUksR0FBRyxTQUFTLFVBQVUsRUFBRyxRQUFPO0FBQ3BDLGdCQUFJLEdBQUcsU0FBUyxRQUFRLEVBQUcsUUFBTztBQUNsQyxnQkFBSSxHQUFHLFNBQVMsU0FBUyxFQUFHLFFBQU87QUFDbkMsbUJBQU87QUFBQSxVQUNUO0FBQUEsUUFDRjtBQUFBLE1BQ0Y7QUFBQSxJQUNGO0FBQUEsRUFDRjtBQUFBLEVBRUEsUUFBUTtBQUFBLElBQ04sTUFBTTtBQUFBLElBQ04sWUFBWTtBQUFBLEVBQ2Q7QUFDRixDQUFDOyIsCiAgIm5hbWVzIjogW10KfQo=
