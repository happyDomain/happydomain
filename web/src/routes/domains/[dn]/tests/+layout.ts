import { type Load } from "@sveltejs/kit";

import { plugins, refreshPlugins } from "$lib/stores/plugins";
import { get } from "svelte/store";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (get(plugins) === undefined) {
        refreshPlugins();
    }

    return {
        ...data,
        isTestsPage: true,
    };
};
