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

import type { Domain } from "$lib/model/domain";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { Zone } from "$lib/model/zone";

/**
 * Fixtures for the models the API sends.
 *
 * A test usually cares about two or three fields, which used to be written as
 * a partial object cast into shape. Filling the remaining fields here instead
 * keeps the overrides type checked, so a renamed or mistyped field shows up as
 * an error in the test rather than as an undefined at runtime.
 */

export function makeDomain(overrides: Partial<Domain> = {}): Domain {
    return {
        id: "domain-test",
        id_owner: "owner-test",
        id_provider: "provider-test",
        domain: "example.com.",
        zone_history: [],
        ...overrides,
    };
}

export function makeZone(overrides: Partial<Zone> = {}): Zone {
    return {
        id: "zone-test",
        id_author: "author-test",
        default_ttl: 3600,
        last_modified: new Date(0),
        services: {},
        ...overrides,
    };
}

/**
 * A service as it comes out of a zone: its type, and the body it carries.
 */
export function makeService(
    svctype: string,
    service: Record<string, any> = {},
    overrides: Partial<ServiceWithValue> = {},
): ServiceWithValue {
    return {
        _svctype: svctype,
        _domain: "",
        _ttl: 3600,
        _tmp_hint_nb: 1,
        Service: service,
        ...overrides,
    };
}
