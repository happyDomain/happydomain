// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

import type { dnsTypeTXT } from "$lib/dns_rr";
import { verificationProviders, type VerificationProvider } from "./providers";

// findVerificationProvider tells the provider of a record, as the backend does
// in services/abstract/domain-verification.go: the provider is not stored.
export function findVerificationProvider(
    record: dnsTypeTXT | undefined,
): VerificationProvider | undefined {
    if (!record) return undefined;

    return verificationProviders.find(
        (p) => (p.label ?? "") === record.Hdr.Name && record.Txt.startsWith(p.prefix ?? ""),
    );
}

// setVerificationValue writes the value pasted by the user. When it belongs to
// a provider expecting its record under a label, the record moves there: the
// value given by a provider is enough to tell where it goes.
export function setVerificationValue(record: dnsTypeTXT, txt: string) {
    record.Txt = txt.trim();

    if (!findVerificationProvider(record)) {
        const p = verificationProviders.find((p) => p.prefix && record.Txt.startsWith(p.prefix));
        if (p) record.Hdr.Name = p.label ?? "";
    }
}

// selectVerificationProvider makes the record match the chosen provider,
// keeping the code already typed.
export function selectVerificationProvider(record: dnsTypeTXT, name: string) {
    const p = verificationProviders.find((p) => p.name === name);
    if (!p) return;

    const code = record.Txt.slice(findVerificationProvider(record)?.prefix?.length ?? 0);

    record.Hdr.Name = p.label ?? "";
    if (!record.Txt.startsWith(p.prefix ?? "")) {
        record.Txt = (p.prefix ?? "") + code;
    }
}
