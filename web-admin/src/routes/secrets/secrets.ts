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

import type { SecretCounts } from "$lib/api-admin";

// sealedCount sums the secrets that open, whatever the kind of their safe.
export function sealedCount(c: SecretCounts): number {
    return Object.values(c.sealed ?? {}).reduce((acc, n) => acc + n, 0);
}

// sealedSplit separates the secrets sealed under the instance keyset from
// those sealed in a keystore the users chose.
export function sealedSplit(c: SecretCounts): { instance: number; user: number } {
    const instance = c.sealed?.instance ?? 0;
    return { instance, user: sealedCount(c) - instance };
}
