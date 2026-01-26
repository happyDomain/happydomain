// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { postProvidersSpecsByProviderTypeSettings } from "$lib/api-base/sdk.gen";
import type { Provider } from "$lib/model/provider";
import type { ProviderSettingsResponse } from "$lib/model/provider_settings";
import { unwrapSdkResponse } from "./errors";

export async function getProviderSettings(
    psid: string,
    state: number,
    settings: any,
    recallid: number | undefined = undefined,
): Promise<ProviderSettingsResponse> {
    if (!state) state = 0;
    if (!settings) settings = {};
    settings.state = state;
    if (recallid) settings.recall = recallid;

    const data = unwrapSdkResponse(
        await postProvidersSpecsByProviderTypeSettings({
            path: { providerType: psid },
            body: settings as any,
        }),
    );

    // If the response has _id, it means the provider setup is complete
    // Throw the Provider object to match old API behavior
    if ((data as any)._id) {
        throw data as Provider;
    } else if ((data as any).form) {
        return data as ProviderSettingsResponse;
    } else {
        throw new Error("Not implemented");
    }
}
