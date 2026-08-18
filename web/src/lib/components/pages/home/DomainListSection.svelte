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
    import { domainLinks } from "$links";
    import { Icon } from "@sveltestrap/sveltestrap";

    import FilterDomainInput from "$lib/components/pages/home/FilterDomainInput.svelte";
    import CardImportableDomains from "$lib/components/providers/CardImportableDomains.svelte";
    import ZoneList from "$lib/components/zones/ZoneList.svelte";
    import { domains } from "$lib/stores/domains";
    import { filterDomains, filteredGroup, filteredName, filteredProvider } from "$lib/stores/home";
    import { t } from "$lib/translations";
    import { getStatusColor, getStatusIcon } from "$lib/utils/checkers";

    let noDomainsList = $state(false);

    let filteredDomains = $derived(
        filterDomains($domains, $filteredName, $filteredProvider, $filteredGroup),
    );
</script>

<FilterDomainInput class="mb-3" />

{#if filteredDomains.length}
    <ZoneList button display_by_groups domains={filteredDomains} links show_empty_groups>
        {#snippet badges({ domain })}
            <a
                href={domainLinks().checks(encodeURIComponent(domain.domain))}
                class={"text-" + getStatusColor(domain.last_check_status)}
            >
                <Icon name={getStatusIcon(domain.last_check_status)} />
            </a>
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
