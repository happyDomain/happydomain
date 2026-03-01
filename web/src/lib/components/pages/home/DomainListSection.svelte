<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import { Icon } from "@sveltestrap/sveltestrap";

    import FilterDomainInput from "$lib/components/pages/home/FilterDomainInput.svelte";
    import CardImportableDomains from "$lib/components/providers/CardImportableDomains.svelte";
    import ZoneList from "$lib/components/zones/ZoneList.svelte";
    import { fqdnCompare } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { domains } from "$lib/stores/domains";
    import { filteredGroup, filteredName, filteredProvider } from "$lib/stores/home";
    import { t } from "$lib/translations";
    import { getStatusColor, getStatusIcon } from "$lib/utils/check";

    let noDomainsList = $state(false);

    let filteredDomains: Array<Domain> = $derived(refreshFilteredDomains());

    function refreshFilteredDomains(): Array<Domain> {
        let myDomains: Array<Domain> = [];

        if ($domains) {
            myDomains = $domains.filter(
                (d) =>
                    (!$filteredName || d.domain.indexOf($filteredName) >= 0) &&
                    (!$filteredProvider || d.id_provider === $filteredProvider._id) &&
                    ($filteredGroup === null ||
                        d.group === $filteredGroup ||
                        (($filteredGroup === "" || $filteredGroup === "undefined") &&
                            (d.group === "" || d.group === undefined))),
            );
            myDomains.sort(fqdnCompare);
        }

        return myDomains;
    }
</script>

<FilterDomainInput class="mb-3" />

{#if filteredDomains.length}
    <ZoneList button display_by_groups domains={filteredDomains} links>
        {#snippet badges(domain: Domain)}
            {#if domain.last_check_status !== undefined}
                <a
                    href="/domains/{encodeURIComponent(domain.domain)}/checks"
                    class={"text-" + getStatusColor(domain.last_check_status)}
                >
                    <Icon name={getStatusIcon(domain.last_check_status)} />
                </a>
            {/if}
        {/snippet}
    </ZoneList>
{:else}
    <div class="my-4 text-center text-muted">
        {$t("domains.filtered-no-result")}
    </div>
{/if}

{#if $filteredProvider}
    <CardImportableDomains
        class={filteredDomains.length > 0 ? "mt-4" : ""}
        provider={$filteredProvider}
        bind:noDomainsList
    />
{/if}
