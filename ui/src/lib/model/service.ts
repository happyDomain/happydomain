export interface ServiceMeta {
    _svctype: string;
    _id: string;
    _ownerid: string;
    _domain: string;
    _ttl: number;
    _comment: string;
    _mycomment: string;
    _aliases: Array<string>;
    _tmp_hint_nb: number;
};
