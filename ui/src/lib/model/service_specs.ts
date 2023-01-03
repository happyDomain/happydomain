import type { Field } from '$lib/model/custom_form';

export class ServiceRestrictions {
    alone: boolean = false;
    exclusive: Array<string> = [];
    glue: boolean = false;
    leaf: boolean = false;
    nearAlone: boolean = false;
    needTypes: Array<number> = [];
    rootOnly: boolean = false;
    single: boolean = false;

    constructor(o: ServiceRestrictions|null = null) {
        if (o) {
            const {alone, exclusive, glue, leaf, nearAlone, needTypes, rootOnly, single} = o
            this.alone = alone;
            this.exclusive = exclusive;
            this.glue = glue;
            this.leaf = leaf;
            this.nearAlone = nearAlone;
            this.needTypes = needTypes;
            this.rootOnly = rootOnly;
            this.single = single;
        }
    }
}

export class ServiceInfos {
    name: string = "";
    _svctype: string = "";
    _svcicon: string = "";
    description: string = "";
    family: string = "";
    categories: Array<string> = [];
    tabs: boolean = false;
    restrictions: ServiceRestrictions = new ServiceRestrictions();

    constructor({name, _svctype, _svcicon, description, family, categories, tabs, restrictions}: ServiceInfos) {
        this.name = name;
        this._svctype = _svctype;
        this._svcicon = _svcicon;
        this.description = description;
        this.family = family;
        this.categories = categories;
        this.tabs = tabs;
        this.restrictions = restrictions;
    }
}

export class ServiceSpec {
    fields: Array<Field> = [];

    constructor({fields}: ServiceSpec) {
        this.fields = fields;
    }
}
