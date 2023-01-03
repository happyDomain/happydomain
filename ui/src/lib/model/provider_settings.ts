import { FormState, FormResponse } from '$lib/model/custom_form';
import { Provider } from '$lib/model/provider';

export class ProviderSettingsState extends FormState {
    Provider?: Provider;

    constructor(form: ProviderSettingsState|null = null) {
        super(form);

        if (form && form.Provider)
            this.Provider = form.Provider;
        else
            this.Provider = new Provider();
    }
};

export class ProviderSettingsResponse extends FormResponse<ProviderSettingsState> {
    Provider: Provider;

    constructor(form: ProviderSettingsResponse|null = null) {
        super(form);

        if (form && form.Provider)
            this.Provider = form.Provider;
        else
            this.Provider = new Provider();
    }
};
