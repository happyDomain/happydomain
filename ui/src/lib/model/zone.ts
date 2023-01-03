import type { ServiceCombined } from '$lib/model/service';

export class ServiceRecord {
    str: string = "";
    fields: any = { }; // dns.RR

    // Interface
    edit: boolean = false;

    constructor({str, fields}: ServiceRecord) {
        this.str = str;
        this.fields = fields;
    }
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

export class Zone implements ZoneMeta {
    id: string;
    id_author: string;
    default_ttl: Number;
    last_modified: Date;
    commit_message: string;
    commit_date: Date;
    published: Date;
    services: Record<string, Array<ServiceCombined>>;

    constructor({id, id_author, default_ttl, last_modified, commit_message, commit_date, published, services}: Zone) {
        this.id = id;
        this.id_author = id_author;
        this.default_ttl = default_ttl;
        this.last_modified = last_modified;
        this.commit_message = commit_message;
        this.commit_date = commit_date;
        this.published = published;
        this.services = services;
    }
}
