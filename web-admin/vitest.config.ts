import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vitest/config";

export default defineConfig({
    plugins: [sveltekit()],
    test: {
        environment: "node",
        include: ["src/**/*.{test,spec}.ts"],
        coverage: {
            provider: "v8",
            reporter: ["text", "html", "lcov"],
            reportsDirectory: "./coverage",
            include: ["src/lib/**/*.ts"],
            exclude: [
                "src/lib/**/*.test.ts",
                "src/lib/**/*.spec.ts",
                "src/lib/api-base/**",
                "src/lib/api-admin/**",
                "src/lib/test-utils/**",
            ],
        },
    },
});
