export interface ZoneHistory {
    id: string;
    id_author: string;
    default_ttl: number;
    last_modified: Date;
    published?: Date;
};

export interface DomainInList {
    id: string;
    id_owner: string;
    id_provider: string;
    domain: string;
    group: string;
    zone_history: Array<string>;

    // interface property
    wait: boolean;
};

export interface Domain {
    id: string;
    id_owner: string;
    id_provider: string;
    domain: string;
    group: string;
    zone_history: Array<ZoneHistory>;

    // interface property
    wait: boolean;
};
