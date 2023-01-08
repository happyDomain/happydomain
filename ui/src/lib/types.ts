import type { Field } from '$lib/model/custom_form';

export function fillUndefinedValues(value: any, spec: Field) {
    if (value[spec.id] === undefined) {
        if (spec.type == "[]uint8") value[spec.id] = "";
        else if (spec.type.startsWith("[]")) value[spec.id] = [];
        else if (spec.type != "string" && !spec.type.startsWith("uint") && !spec.type.startsWith("int") && !spec.type.startsWith("time.Duration")) value[spec.id] = { };
    }
}
