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
    import { navigate } from "$lib/stores/config";

    import { Alert, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import type { Domain } from "$lib/model/domain";
    import { retrieveZone } from "$lib/stores/thiszone";
    import { domains_idx, refreshDomains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    let rz = $derived(retrieveZone(data.domain));
    $effect(() => {
        rz.then(
            () => {
                refreshDomains().then(
                    () => {
                        navigate(
                            `/domains/${encodeURIComponent($domains_idx[data.domain.domain] ? data.domain.domain : data.domain.id)}`,
                        );
                    },
                );
            },
            (e) => {},
        );
    });
</script>

<div class="flex-fill d-flex flex-column">
    <PageTitle title={$t("zones.retrieve")} domain={data.domain.domain} subtitle={$t("zones.retrieve-subtitle")} />
    {#await rz}
        <div class="mt-4 text-center flex-fill">
            <Spinner />
            <p>{$t("wait.importing")}</p>
        </div>
    {:then}
        <div class="mt-4 text-center flex-fill">
            <Spinner />
            <p>{$t("wait.wait")}</p>
        </div>
    {:catch main_error}
        <div class="mt-4 text-center flex-fill">
            <Alert color="danger" fade={false}>
                <strong>{$t("errors.domain-import")}</strong>
                {main_error}
            </Alert>
        </div>
    {/await}
</div>
