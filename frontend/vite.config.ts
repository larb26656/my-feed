import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";

import { tanstackRouter } from "@tanstack/router-plugin/vite";

import viteReact from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

const config = defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), "");
	return {
		server: {
			proxy: {
				"/api": env.VITE_API_PROXY_TARGET ?? "http://localhost:8080",
			},
		},
		resolve: { tsconfigPaths: true },
		plugins: [
			devtools(),
			tailwindcss(),
			tanstackRouter({ target: "react", autoCodeSplitting: true }),
			viteReact(),
		],
	};
});

export default config;
