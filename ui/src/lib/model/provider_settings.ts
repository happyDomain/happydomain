import type { FormState, FormResponse } from '$lib/model/custom_form';
import type { Provider } from '$lib/model/provider';

export interface ProviderSettingsState extends FormState {
    Provider?: Provider;
};

export interface ProviderSettingsResponse extends FormResponse<ProviderSettingsState> {

};
