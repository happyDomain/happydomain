import { sveltekit } from '@sveltejs/kit/vite';
import { searchForWorkspaceRoot } from 'vite';
import { defineConfig } from 'vitest/config';

const config = defineConfig({
        server: {
            fs: {
                allow: [
                    searchForWorkspaceRoot(process.cwd()),
                    searchForWorkspaceRoot(process.cwd() + "/../web"),
                ],
            },
            port: 5174,
            hmr: {
                port: 10001
            }
        },

	plugins: [sveltekit()],

	// Two kinds of tests live side by side: the ones exercising plain functions run
	// in Node, and the ones mounting a component need a DOM, hence jsdom. They are
	// told apart by their name: a component test is named after the component, so
	// it ends with `.svelte.test.ts`.
	test: {
	    projects: [
	        {
	            extends: true,
	            test: {
	                name: "unit",
	                environment: "node",
	                include: ["src/**/*.test.ts"],
	                exclude: ["src/**/*.svelte.test.ts"],
	            },
	        },
	        {
	            extends: true,
	            // Svelte ships a server build and a browser build; mounting a
	            // component for real calls for the latter.
	            resolve: { conditions: ["browser"] },
	            test: {
	                name: "component",
	                environment: "jsdom",
	                include: ["src/**/*.svelte.test.ts"],
	                // Inside src, so its matcher types reach the test files.
	                setupFiles: ["./src/vitest-setup.ts"],
	            },
	        },
	    ],
	},
});

export default config;
