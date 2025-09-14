// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import type { Field } from "$lib/model/custom_form.svelte";

export function fillUndefinedValues(value: any, spec: Field) {
    if (value[spec.id] === undefined && spec.type.length) {
        let vartype = spec.type;
        if (vartype[0] == "*") vartype = vartype.substring(1);

        if (spec.default !== undefined) value[spec.id] = spec.default;
        else if (vartype == "bool") value[spec.id] = false;
        else if (vartype == "[]uint8") value[spec.id] = "";
        else if (vartype.startsWith("[]")) value[spec.id] = [];
        else if (
            vartype != "string" &&
            !vartype.startsWith("uint") &&
            !vartype.startsWith("int") &&
            vartype != "net.IP" &&
            vartype != "common.URL" &&
            vartype != "time.Duration" &&
            vartype != "common.Duration"
        )
            value[spec.id] = {};
    }
}
