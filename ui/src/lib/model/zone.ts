import type { ServiceCombined } from '$lib/model/service';

export interface ServiceRecord {
    str: string;
    fields: any; // dns.RR

    // ui
    edit?: boolean;
}

export interface ZoneMeta {
    id: string;
    id_author: string;
    default_ttl: Number;
    last_modified: Date;
    commit_message?: string;
    commit_date?: Date;
    published?: Date;
};

export interface Zone extends ZoneMeta {
    services: Record<string, Array<ServiceCombined>>;
}
