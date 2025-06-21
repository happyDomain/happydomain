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
    import { Button, Col, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import AliasModal from "./AliasModal.svelte";
    import SubdomainList from "./SubdomainList.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { Zone } from "$lib/model/zone";
    import { domains_idx } from "$lib/stores/domains";
    import { servicesSpecs, refreshServicesSpecs } from "$lib/stores/services";
    import { sortedDomains, sortedDomainsWithIntermediate, thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    if (!$servicesSpecs) refreshServicesSpecs();

    interface Props {
        data: { domain: Domain; history: string; zoneId: string };
    }

    let { data }: Props = $props();
</script>

{#if !data.domain}
    <div class="mt-5 text-center flex-fill">
        <Spinner />
        <p>{$t("wait.loading")}</p>
    </div>
{:else if !data.domain.zone_history || data.domain.zone_history.length == 0}
    <div class="mt-4 text-center flex-fill">
        <Spinner />
        <p>{$t("wait.importing")}</p>
    </div>
{:else if !$thisZone || !$sortedDomains}
    <div class="mt-4 text-center flex-fill">
        <Spinner />
        <p>{$t("wait.loading")}</p>
    </div>
{:else}
    <div style="max-width: 100%;" class="w-100 pt-1">
        {#if !$sortedDomainsWithIntermediate || $sortedDomains.length == 0}
            <SubdomainList
                origin={data.domain}
                sortedDomainsWithIntermediate={['']}
            />
        {:else}
            <SubdomainList
                origin={data.domain}
                sortedDomainsWithIntermediate={$sortedDomainsWithIntermediate}
            />
        {/if}
    </div>
{/if}

<AliasModal
    origin={data.domain}
/>
