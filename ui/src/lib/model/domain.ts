import { applyZone, diffZone, getZone, importZone, viewZone } from '$lib/api/zone';
import { deleteDomain, updateDomain } from '$lib/api/domains';
import type { Zone, ZoneMeta } from '$lib/model/zone';

export class Domain {
    id: string = "";
    id_owner: string = "";
    id_provider: string = "";
    domain: string = "";
    group: string = "";
    zone_history: Array<string> = [];

    // interface property
    wait: boolean = false;

    constructor(dn: Domain) {
        this._update(dn)
    }

    _update({ id, id_owner, id_provider, domain, group, zone_history }: Domain) {
        this.id = id;
        this.id_owner = id_owner;
        this.id_provider = id_provider;
        this.domain = domain;
        this.group = group;
        this.zone_history = zone_history;
        this.wait = false;
    }

    delete(): Promise<boolean> {
        return deleteDomain(this.id);
    }

    async save() {
        this._update(await updateDomain(this));
    }

    importZone(): Promise<ZoneMeta> {
        return importZone(this);
    }

    getZone(zoneid: string): Promise<Zone> {
        return getZone(this, zoneid);
    }

    viewZone(zoneid: string): Promise<string> {
        return viewZone(this, zoneid);
    }

    diffZone(zoneid1: string, zoneid2: string): Promise<Array<string>> {
        return diffZone(this, zoneid1, zoneid2);
    }

    applyZoneDiff(id: string, diffs: Array<string>): Promise<ZoneMeta> {
        return applyZone(this, id, diffs);
    }
};
