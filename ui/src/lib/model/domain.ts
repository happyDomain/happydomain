export interface Domain {
    id: string;
    id_owner: string;
    id_provider: string;
    domain: string;
    group: string;
    zone_history: Array<string>;
};
