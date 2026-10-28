// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { error, type Load } from "@sveltejs/kit";

import { getZoneTemplate } from "$lib/api/zone_template";
import type { ZoneTemplate } from "$lib/model/zone_template";

export const load: Load = async ({ parent, params, fetch }) => {
    const data = await parent();

    let template: ZoneTemplate;
    try {
        template = await getZoneTemplate(params.tid ?? "", fetch);
    } catch {
        error(404, { message: "Zone template not found" });
    }

    return { ...data, template };
};
