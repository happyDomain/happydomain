import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
    input: "../docs-admin/swagger.yaml",
    output: "src/lib/api-admin",
    plugins: [
        {
            name: "@hey-api/client-fetch",
            runtimeConfigPath: "$lib/hey-api-admin.ts",
        },
    ],
});
