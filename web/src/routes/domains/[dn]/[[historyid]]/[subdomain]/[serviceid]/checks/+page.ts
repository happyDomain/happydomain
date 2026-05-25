import type { Load } from "@sveltejs/kit";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    const subdomain = params.subdomain === "@" ? "" : params.subdomain;
    const serviceid = params.serviceid;

    return {
        ...data,
        subdomain,
        serviceid,
    };
};
