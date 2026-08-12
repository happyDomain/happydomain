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

import { get, writable } from "svelte/store";

import { FORGES } from "$lib/utils/report";

// The report dialog lives in the root layout, but it is opened from anywhere:
// the user menu, an error toast, ... so it registers its opener here.
type Opener = (reportedError?: string) => void;

const opener = writable<Opener | undefined>(undefined);

export function registerReportModal(open: Opener) {
    opener.set(open);
}

export function openReportModal(reportedError?: string) {
    const open = get(opener);
    if (open) {
        open(reportedError);
    } else if (typeof window !== "undefined") {
        // The dialog should always be mounted; should it ever not be, opening
        // the tracker is still better than swallowing the click.
        window.open(`${FORGES[0].issues}/new`, "_blank", "noopener");
    }
}
