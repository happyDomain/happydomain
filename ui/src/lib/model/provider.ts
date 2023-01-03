import { deleteProvider, listImportableDomains, updateProvider } from '$lib/api/provider';

export class ProviderInfos {
    name: string = "";
    description: string = "";
    capabilites: Array<string> = [];

    constructor({ name, description, capabilites }: ProviderInfos) {
        this.name = name;
        this.description = description;
        this.capabilites = capabilites;
    }
};

export type ProviderList = Record<string, ProviderInfos>;

export interface ProviderMeta {
    _srctype: string;
    _id: string;
    _ownerid: string;
    _comment: string;
};

export interface ProviderData extends ProviderMeta {
    Provider: any;
}

export class Provider implements ProviderMeta {
    _srctype: string = "";
    _id: string = "";
    _ownerid: string = "";
    _comment: string = "";
    Provider: any = { };

    constructor(prvdr: ProviderData|null = null) {
        if (prvdr) {
            this._update(prvdr);
        }
    }

    _update({_srctype, _id, _ownerid, _comment, Provider}: ProviderData) {
        this._srctype = _srctype;
        this._id = _id;
        this._ownerid = _ownerid;
        this._comment = _comment;
        this.Provider = Provider;
    }

    delete(): Promise<boolean> {
        return deleteProvider(this._id);
    }

    async save() {
        this._update(await updateProvider(this));
    }

    listImportableDomains(): Promise<Array<string>> {
        return listImportableDomains(this);
    }
};
