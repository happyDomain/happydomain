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
    import { run } from 'svelte/legacy';

    // @ts-ignore
    import { escape } from "html-escaper";
    import {
        Button,
        Card,
        CardHeader,
        Col,
        Container,
        Icon,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { createDomain } from "$lib/api/provider";
    import FilterDomainInput from "$lib/components/pages/home/FilterDomainInput.svelte";
    import CardImportableDomains from "$lib/components/providers/CardImportableDomains.svelte";
    import DomainGroupList from "$lib/components/forms/DomainGroupList.svelte";
    import DomainGroupModal from "$lib/components/modals/DomainGroup.svelte";
    import Logo from "$lib/components/Logo.svelte";
    import ZoneList from "$lib/components/zones/ZoneList.svelte";
    import { fqdnCompare } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { domains } from "$lib/stores/domains";
    import { filteredGroup, filteredName, filteredProvider } from '$lib/stores/home';
    import { providersSpecs } from "$lib/stores/providers";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    let noDomainsList = $state(false);

    let filteredDomains: Array<Domain> = $derived(refreshFilteredDomains($domains, $filteredName, $filteredProvider, $filteredGroup));

    function refreshFilteredDomains() {
        let myDomains = [];

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

    function newDomainAdded(event: CustomEvent<Domain>) {
        toasts.addToast({
            title: $t("domains.attached-new"),
            message: $t("domains.added-success", { domain: event.detail.domain }),
            href: "/domains/" + event.detail.domain,
            color: "success",
            timeout: 5000,
        });
    }

    async function createDomainOnProvider(fqdn: string) {
        if (!$filteredProvider) return;

        return await createDomain($filteredProvider, fqdn)
    }
</script>

<FilterDomainInput class="mb-3" />

{#if filteredDomains.length}
    <ZoneList button display_by_groups domains={filteredDomains} links />
{:else}
    <div class="my-4 text-center text-muted">
        {$t('domains.filtered-no-result')}
    </div>
{/if}

{#if $filteredProvider}
    <CardImportableDomains
        class={filteredDomains.length > 0 ? "mt-4" : ""}
        provider={$filteredProvider}
        bind:noDomainsList
    />
{/if}
