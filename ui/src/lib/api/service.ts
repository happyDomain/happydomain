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

import { handleEmptyApiResponse, handleApiResponse } from '$lib/errors';
import type { ServiceCombined } from '$lib/model/service';

export async function updateService(zoneid: string, svc: ServiceCombined): Promise<ServiceCombined> {
    const res = await fetch('/api/zone/' + zoneid + '/services/' + (svc._id ? `/${svc._id}` : ''), {
        method: svc._id?'PUT':'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(svc),
    });
    return await handleApiResponse<ServiceCombined>(res);
}

export async function deleteService(zoneid: string, id: string): Promise<boolean> {
    const res = await fetch(`/api/zone/${zoneid}/services/${id}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}
