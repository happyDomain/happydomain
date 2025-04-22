<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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

<script lang="ts">
    import { goto } from "$app/navigation";

    import { Alert, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import { retrieveZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    export let data: { domain: Domain };

    let rz = retrieveZone(data.domain);
    rz.then(
        () => {
            goto(`/domains/${encodeURIComponent(data.domain.domain)}`);
        },
        (e) => {},
    );
</script>

{#await rz}
    <div class="flex-fill d-flex flex-column">
        <h2 class="d-flex align-items-center">
            <Spinner label="Spinning" type="grow" />
            <span class="ms-2 mt-1 font-monospace">
                {data.domain.domain}
            </span>
        </h2>

        <div class="mt-4 text-center flex-fill">
            <Spinner label={$t("common.spinning")} />
            <p>{$t("wait.importing")}</p>
        </div>
    </div>
{:then}
    <div class="flex-fill d-flex flex-column">
        <h2 class="d-flex align-items-center">
            <Spinner label="Spinning" type="grow" />
            <span class="ms-2 mt-1 font-monospace">
                {data.domain.domain}
            </span>
        </h2>

        <div class="mt-4 text-center flex-fill">
            <Spinner label={$t("common.spinning")} />
            <p>{$t("wait.wait")}</p>
        </div>
    </div>
{:catch main_error}
    <div class="mt-4 text-center flex-fill">
        <Alert color="danger" fade={false}>
            <strong>{$t("errors.domain-import")}</strong>
            {main_error}
        </Alert>
    </div>
{/await}
