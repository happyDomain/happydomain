import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
    input: "../docs/swagger.yaml",
    output: "src/lib/api-base",
    plugins: [
        {
            name: "@hey-api/client-fetch",
            runtimeConfigPath: "$lib/hey-api.ts",
        },
    ],
});
