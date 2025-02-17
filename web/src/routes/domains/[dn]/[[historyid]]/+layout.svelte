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
    import { page } from "$app/stores";

    import { Spinner } from "@sveltestrap/sveltestrap";

    import NewServicePath from "$lib/components/services/NewServicePath.svelte";
    import RecordModal from "$lib/components/domains/RecordModal.svelte";
    import ServiceModal from "$lib/components/services/ServiceModal.svelte";
    import type { Domain } from "$lib/model/domain";
    import { domains_idx, refreshDomains } from "$lib/stores/domains";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    export let data: { domain: Domain; history: string; definedhistory: boolean };

    let selectedDomain = data.domain.id;
    let selectedHistory: string = data.history;
    $: historyChange(selectedHistory);
    $: historyChange(data.history);
    function historyChange(history: string) {
        if (data.history != history) {
            goto(
                "/domains/" +
                    encodeURIComponent(
                        $domains_idx[data.domain.domain]
                            ? $domains_idx[data.domain.domain].domain
                            : selectedDomain,
                    ) +
                    "/" +
                    encodeURIComponent(selectedHistory),
            );
        }
        if (history != selectedHistory) {
            selectedHistory = history;
        }
    }
</script>

{#if $thisZone && $thisZone.id == selectedHistory}
    <slot />

    <NewServicePath origin={data.domain} zone={$thisZone} />
    <RecordModal
        origin={data.domain}
        zone={$thisZone}
        on:update-zone-services={(event) => thisZone.set(event.detail)}
    />
    <ServiceModal
        origin={data.domain}
        zone={$thisZone}
        on:update-zone-services={(event) => thisZone.set(event.detail)}
    />
{:else}
    <div class="flex-fill d-flex flex-column">
        <h2 class="d-flex align-items-center">
            <Spinner type="grow" />
            <span class="ms-2 mt-1 font-monospace">
                {data.domain.domain}
            </span>
        </h2>

        <div class="mt-4 text-center flex-fill">
            <Spinner />
            <p>{$t("wait.loading")}</p>
        </div>
    </div>
{/if}
