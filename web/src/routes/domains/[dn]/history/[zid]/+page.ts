import { error, type Load } from "@sveltejs/kit";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    if (!params.zid || !data.historyDomain?.zone_meta?.[params.zid]) {
        error(404, { message: "Zone version not found" });
    }

    // The layout's ModalDiffZone rolls back to page.data.history.
    return { zid: params.zid, history: params.zid };
};
