import { handleApiResponse } from '$lib/errors';
import type { Provider } from '$lib/model/provider';
import type { ProviderSettingsResponse } from '$lib/model/provider_settings';

export async function getProviderSettings(psid: string, state: number, settings: any, recallid: number|undefined = undefined): Promise<ProviderSettingsResponse> {
    if (!state) state = 0;
    if (!settings) settings = {};
    settings.state = state;
    if (recallid) settings.recall = recallid;

    const res = await fetch('/api/providers/_specs/' + encodeURIComponent(psid) + '/settings', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(settings),
    });
    const data = await handleApiResponse<any>(res);
    if (data._id) {
        throw data as Provider;
    } else if (data.form) {
        return data as ProviderSettingsResponse;
    } else {
        throw new Error("Not implemented");
    }
}
