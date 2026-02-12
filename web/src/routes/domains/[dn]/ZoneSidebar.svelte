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

<script lang="ts">
    import {
        Button,
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { isReverseZone } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domains_idx } from "$lib/stores/domains";
    import {
        retrieveZone as StoreRetrieveZone,
        sortedDomains,
        sortedDomainsWithIntermediate,
        thisZone,
    } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import { navigate } from "$lib/stores/config";
    import { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import { controls as ctrlUploadZone } from "./ModalUploadZone.svelte";
    import { controls as ctrlNewSubdomain } from "./NewSubdomainPath.svelte";
    import SubdomainListTiny from "./SubdomainListTiny.svelte";

    interface Props {
        origin: Domain;
        selectedDomain: string;
        selectedHistory: string | undefined;
        onretrieveZoneDone: (zm: ZoneMeta) => void;
    }

    let { origin, selectedDomain, selectedHistory, onretrieveZoneDone }: Props = $props();

    function domainLink(dn: string): string {
        return $domains_idx[$domains_idx[dn].domain] ? $domains_idx[dn].domain : dn;
    }

    let retrievalInProgress = $state(false);
    async function retrieveZone() {
        retrievalInProgress = true;
        const zm = await StoreRetrieveZone(origin);
        retrievalInProgress = false;
        onretrieveZoneDone(zm);
    }

    function viewZone(): void {
        if (!selectedHistory) return;
        navigate(`/domains/${domainLink(selectedDomain)}/export`);
    }
</script>

<div class="d-flex gap-2 pb-2 sticky-top" style="padding-top: 10px">
    <Button
        type="button"
        color="secondary"
        outline
        size="sm"
        class="flex-fill text-truncate"
        disabled={!$sortedDomains}
        on:click={() => ctrlNewSubdomain.Open()}
    >
        <Icon name="server" />
        {$t("domains.add-a-subdomain")}
    </Button>
    <Dropdown>
        <DropdownToggle
            color="secondary"
            outline
            size="sm"
            aria-label={$t("domains.actions.others", {
                domain: $domains_idx[selectedDomain].domain,
            })}
            title={$t("domains.actions.others", {
                domain: $domains_idx[selectedDomain].domain,
            })}
        >
            {#if retrievalInProgress}
                <Spinner size="sm" />
            {:else}
                <Icon name="gear-fill" aria-hidden="true" />
            {/if}
        </DropdownToggle>
        <DropdownMenu>
            <DropdownItem header class="font-monospace">
                {origin.domain}
            </DropdownItem>
            <DropdownItem href={`/domains/${domainLink(selectedDomain)}/history`}>
                {$t("domains.actions.history")}
            </DropdownItem>
            <DropdownItem href={`/domains/${domainLink(selectedDomain)}/logs`}>
                {$t("domains.actions.audit")}
            </DropdownItem>
            <DropdownItem href={`/domains/${domainLink(selectedDomain)}/checks`}>
                {$t("domains.actions.view-checks")}
            </DropdownItem>
            <DropdownItem divider />
            <DropdownItem on:click={viewZone} disabled={!$sortedDomains}>
                {$t("domains.actions.view")}
            </DropdownItem>
            <DropdownItem on:click={retrieveZone}>
                {$t("domains.actions.reimport")}
            </DropdownItem>
            <DropdownItem on:click={() => ctrlUploadZone.Open()}>
                {$t("domains.actions.upload")}
            </DropdownItem>
            <DropdownItem divider />
            <DropdownItem disabled title="Coming soon...">
                {$t("domains.actions.share")}
            </DropdownItem>
            <DropdownItem on:click={() => ctrlDomainDelete.Open()}>
                {$t("domains.stop")}
            </DropdownItem>
            <DropdownItem divider />
            <DropdownItem
                href={"/providers/" +
                    encodeURIComponent($domains_idx[selectedDomain].id_provider)}
            >
                {$t("provider.update")}
            </DropdownItem>
        </DropdownMenu>
    </Dropdown>
</div>
<div style="min-height:0; overflow-y: auto;" class="placeholder-glow">
    {#if $sortedDomains && $sortedDomainsWithIntermediate && $thisZone && $thisZone.id == selectedHistory}
        {#if isReverseZone(origin.domain)}
            <SubdomainListTiny domains={$sortedDomains} {origin} />
        {:else}
            <SubdomainListTiny domains={$sortedDomainsWithIntermediate} {origin} />
        {/if}
    {:else}
        <span class="d-block text-truncate font-monospace text-muted">
            {origin.domain}
        </span>
        <span class="d-block placeholder ms-3 mb-1">
            {origin.domain}
        </span>
        <span class="d-block placeholder ms-3 mb-1">
            {origin.domain}
        </span>
        <span class="d-block placeholder ms-4 mb-1">
            {origin.domain}
        </span>
        <span class="d-block placeholder ms-4 mb-1">
            {origin.domain}
        </span>
        <span class="d-block placeholder ms-3">
            {origin.domain}
        </span>
    {/if}
</div>
