export interface ProviderInfos {
    name: string;
    description: string;
    capabilities: Array<string>;
};

export function getAvailableResourceTypes(pi: ProviderInfos): Array<number> {
    const availableResourceTypes = [];

    for (const cap of pi.capabilities) {
        if (cap.startsWith('rr-')) {
            availableResourceTypes.push(parseInt(cap.substring(3, cap.indexOf('-', 4))))
        }
    }

    return availableResourceTypes;
}

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

export interface Provider extends ProviderMeta {
    Provider: any;
};
