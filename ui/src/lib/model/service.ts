export class ServiceMeta {
    _svctype: string;
    _id: string;
    _ownerid: string;
    _domain: string;
    _ttl: number;
    _comment: string;
    _mycomment: string;
    _aliases: Array<string>;
    _tmp_hint_nb: number;

    constructor({_svctype, _id, _ownerid, _domain, _ttl, _comment, _mycomment, _aliases, _tmp_hint_nb}: ServiceMeta) {
        this._svctype = _svctype;
        this._id = _id;
        this._ownerid = _ownerid;
        this._domain = _domain;
        this._ttl = _ttl;
        this._comment = _comment;
        this._mycomment = _mycomment;
        this._aliases = _aliases;
        this._tmp_hint_nb = _tmp_hint_nb;
    }
};

export class ServiceCombined extends ServiceMeta {
    Service: any;

    constructor(svc: ServiceCombined) {
        super(svc);
        this.Service = svc.Service;
    }
}
