import { goto } from '$app/navigation';
import cuid from 'cuid';

import { getProviderSettings } from '$lib/api/provider_settings';
import type { CustomForm } from '$lib/model/custom_form';
import type { ProviderSettingsState } from '$lib/model/provider_settings';

export class ProviderForm {
    ptype: string = "";
    state: number = 0;
    providerId: string = "";
    form: CustomForm|undefined = undefined;
    value: ProviderSettingsState = {state: 0};
    nextInProgress: boolean = false;
    previousInProgress: boolean = false;
    on_previous: null | (() => void);
    on_done: () => void;

    constructor(ptype: string, on_done: () => void, providerId: string | null = null, value: ProviderSettingsState | null = null, on_previous: null | (() => void) = null) {
        this.ptype = ptype;
        this.state = -1;
        this.providerId = providerId?providerId:cuid();
        this.form = undefined;
        this.value = value?value:{recall: this.providerId, state: this.state};
        this.on_done = on_done;
        this.on_previous = on_previous;

        if ((!this.value.Provider || !Object.keys(this.value.Provider).length)) {
            const sstore = sessionStorage.getItem("newprovider-" + this.providerId);
            if (sstore) {
                const data = JSON.parse(sstore);
                if (data) {
                    if (data._id) this.value._id = data._id;
                    this.value._comment = data._comment;
                    this.value.Provider = data.Provider;
                }
            }
        }
        if (!this.value.recall) {
            this.value.recall = this.providerId;
        }

        this.nextInProgress = false;
        this.previousInProgress = false;
    }

    async changeState(toState: number): Promise<CustomForm | undefined> {
        if (toState == -1) {
            this.state = toState;
            if (this.on_previous) this.on_previous();
            return undefined;
        } else {
            try {
                const res = await getProviderSettings(this.ptype, toState, this.value);
                this.state = toState;
                if (res.values) {
                    // @ts-ignore
                    this.value.Provider = { ...this.value.Provider, ...res.values };
                }
                return res.form;
            } catch (e) {
                if ("Provider" in (e as any) && "_id" in (e as any) && "_srctype" in (e as any)) {
                    sessionStorage.removeItem("newprovider-" + this.providerId);
                    this.on_done();
                    return undefined;
                } else {
                    this.nextInProgress = false;
                    this.previousInProgress = false;
                    throw e;
                }
            }
        }
    }

    saveState() {
        sessionStorage.setItem("newprovider-" + this.providerId, JSON.stringify(this.value))
    }

    async nextState() {
        this.nextInProgress = true;
        this.saveState();
        if (this.form && this.form.nextButtonLink) {
            goto(this.form.nextButtonLink);
        } else {
            this.form = await this.changeState(this.form && this.form.nextButtonState ? this.form.nextButtonState : 0);
        }
        this.nextInProgress = false;
    }

    async previousState() {
        this.previousInProgress = true;
        this.saveState();
        if (this.form && this.form.previousButtonLink) {
            goto(this.form.previousButtonLink);
        } else {
            this.form = await this.changeState(this.form && this.form.previousButtonState ? this.form.previousButtonState : 0);
        }
        this.previousInProgress = false;
    }

}
