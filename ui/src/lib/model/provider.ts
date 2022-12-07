import type Map from '$lib/model/golang';

export class ProviderInfos {
    name: string;
    description: string;
    capabilites: Array<string>;

    constructor({ name, description, capabilites }: ProviderInfos) {
        this.name = name;
        this.description = description;
        this.capabilites = capabilites;
    }
};

export type ProviderList = Map<string, ProviderInfos>;

export interface ProviderMeta {
    _srctype: string;
    _id: string;
    _ownerid: string;
    _comment: string;
};

export interface Provider extends ProviderMeta {

};
