export class ResolverForm {
    domain: string = "";
    type: string = "ANY";
    resolver: string = "local";
    custom?: string = undefined;

    constructor(o: ResolverForm|null = null) {
        if (o != null) {
            const {domain, type, resolver, custom} = o;
            this.domain = domain;
            this.type = type;
            this.resolver = resolver;
            this.custom = custom;
        }
    }
};
