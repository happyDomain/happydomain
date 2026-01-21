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

import { redirect } from '@sveltejs/kit';
import { getProviders } from '$lib/api-admin';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
    const providerId = params.provider;

    // Fetch all providers to find the owner
    const providersResponse = await getProviders();

    if (providersResponse.data) {
        const provider = providersResponse.data.find(p => p._id === providerId);

        if (provider && provider._ownerid) {
            // Redirect to the user-specific provider route
            throw redirect(302, `/users/${provider._ownerid}/providers/${providerId}`);
        }
    }

    // If provider not found or no owner, throw 404
    throw redirect(302, '/providers');
};
