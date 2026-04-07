<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script module lang="ts">
    import { getDomainInfo } from "$lib/api/domaininfo";
    import type { DomainInfo } from "$lib/model/domaininfo";

    export interface DomainInfoState {
        info: DomainInfo | null;
        error: string | null;
        notFound: boolean;
        pending: boolean;
    }

    export function createDomainInfoState(): DomainInfoState {
        return {
            info: null,
            error: null,
            notFound: false,
            pending: false,
        };
    }

    export function fetchDomainInfo(domain: string, state: DomainInfoState): void {
        if (!domain) return;

        state.pending = true;
        state.error = null;
        state.notFound = false;
        state.info = null;

        getDomainInfo(domain).then(
            (result) => {
                state.info = result;
                state.pending = false;
            },
            (err: unknown) => {
                const msg = err instanceof Error ? err.message : String(err);
                if (msg.toLowerCase().includes("not found") || msg.toLowerCase().includes("doesn't exist")) {
                    state.notFound = true;
                } else {
                    state.error = msg;
                }
                state.pending = false;
            },
        );
    }
</script>
