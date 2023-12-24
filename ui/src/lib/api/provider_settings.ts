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

import { handleApiResponse } from '$lib/errors';
import type { Provider } from '$lib/model/provider';
import type { ProviderSettingsResponse } from '$lib/model/provider_settings';

export async function getProviderSettings(psid: string, state: number, settings: any, recallid: number|undefined = undefined): Promise<ProviderSettingsResponse> {
    if (!state) state = 0;
    if (!settings) settings = {};
    settings.state = state;
    if (recallid) settings.recall = recallid;

    const res = await fetch('/api/providers/_specs/' + encodeURIComponent(psid) + '/settings', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(settings),
    });
    const data = await handleApiResponse<any>(res);
    if (data._id) {
        throw data as Provider;
    } else if (data.form) {
        return data as ProviderSettingsResponse;
    } else {
        throw new Error("Not implemented");
    }
}
