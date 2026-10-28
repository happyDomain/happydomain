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
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import {
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domains_idx, domainLink } from "$lib/stores/domains";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { retrieveZone as StoreRetrieveZone, sortedDomains } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import { controls as ctrlDomainWhois } from "./ModalDomainWhois.svelte";
    import { controls as ctrlUploadZone } from "./ModalUploadZone.svelte";
    import { controls as ctrlApplyTemplate } from "./ModalApplyTemplate.svelte";
    import { controls as ctrlSaveAsTemplate } from "./ModalSaveAsTemplate.svelte";

    interface Props {
        origin: Domain;
        selectedDomain: string;
        selectedHistory: string | undefined;
        onretrieveZoneDone: (zm: ZoneMeta) => void;
    }

    let { origin, selectedDomain, selectedHistory, onretrieveZoneDone }: Props = $props();

    let readOnly = $derived(isReadOnlyProvider($providers_idx, origin.id_provider));

    let retrievalInProgress = $state(false);
    async function retrieveZone() {
        retrievalInProgress = true;
        const zm = await StoreRetrieveZone(origin);
        retrievalInProgress = false;
        onretrieveZoneDone(zm);
    }

    function viewZone(): void {
        if (!selectedHistory) return;
        goto(resolve("/domains/[dn]/export", { dn: domainLink(selectedDomain) }));
    }
</script>

<Dropdown class="d-flex flex-shrink-0" size="sm">
    <DropdownToggle
        color="secondary"
        outline
        size="sm"
        class="d-flex flex-column align-items-center justify-content-center gap-1 lh-sm text-nowrap"
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
        <span class="small">{$t("domains.actions.label")}</span>
    </DropdownToggle>
    <DropdownMenu end style="z-index: 11;">
        <DropdownItem header>
            {$t("domains.actions.menu-title")}
            <span class="d-block font-monospace fw-normal text-muted text-truncate">
                {origin.domain}
            </span>
        </DropdownItem>
        <DropdownItem href={`/domains/${domainLink(selectedDomain)}/history`}>
            {$t("domains.actions.history")}
        </DropdownItem>
        <DropdownItem href={`/domains/${domainLink(selectedDomain)}/logs`}>
            {$t("domains.actions.audit")}
        </DropdownItem>
        <DropdownItem on:click={() => ctrlDomainWhois.Open()}>
            {$t("domains.actions.whois")}
        </DropdownItem>
        <DropdownItem divider />
        <DropdownItem on:click={retrieveZone}>
            {$t("domains.actions.reimport")}
        </DropdownItem>
        <DropdownItem on:click={viewZone} disabled={!$sortedDomains}>
            {$t("domains.actions.view")}
        </DropdownItem>
        <DropdownItem on:click={() => ctrlSaveAsTemplate.Open()} disabled={!$sortedDomains}>
            {$t("domains.actions.save-template")}
        </DropdownItem>
        <DropdownItem
            disabled={readOnly}
            title={readOnly ? $t("domains.read-only-edit") : undefined}
            on:click={() => ctrlUploadZone.Open()}
        >
            {$t("domains.actions.upload")}
        </DropdownItem>
        <DropdownItem
            disabled={readOnly}
            title={readOnly ? $t("domains.read-only-edit") : undefined}
            on:click={() => ctrlApplyTemplate.Open()}
        >
            {$t("domains.actions.apply-template")}
        </DropdownItem>
        <DropdownItem divider />
        <DropdownItem disabled title="Coming soon...">
            {$t("domains.actions.share")}
        </DropdownItem>
        <DropdownItem on:click={() => ctrlDomainDelete.Open()}>
            {$t("domains.stop")}
        </DropdownItem>
    </DropdownMenu>
</Dropdown>
