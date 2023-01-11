import type { Field } from '$lib/model/custom_form';

export function fillUndefinedValues(value: any, spec: Field) {
    if (value[spec.id] === undefined && spec.type.length) {
        let vartype = spec.type;
        if (vartype[0] == "*") vartype = vartype.substring(1);

        if (vartype == "[]uint8") value[spec.id] = "";
        else if (vartype.startsWith("[]")) value[spec.id] = [];
        else if (vartype != "string" && !vartype.startsWith("uint") && !vartype.startsWith("int") && vartype != "time.Duration" && vartype != "net.IP") value[spec.id] = { };
    }
}
