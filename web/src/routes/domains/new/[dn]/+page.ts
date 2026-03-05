import { redirect } from "@sveltejs/kit";
import type { Load } from "@sveltejs/kit";

export const load: Load = async ({ params }) => {
    redirect(302, "/domains/?new=" + encodeURIComponent(params.dn ?? ""));
};
