import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
    preprocess: vitePreprocess(),

    kit: {
        adapter: adapter({
            fallback: "index.html",
        }),
        // Route knowledge that web-admin, which shares src/lib by symlink,
        // has no routes for.
        alias: {
            $links: "src/links.ts",
        },
        // The counterpart of web/src/links.ts, for the shared components.
        alias: {
            $links: "src/links.ts",
        },
        paths: {
            relative: process.env.NODE_ENV === "production",
        },
    },
};

export default config;
