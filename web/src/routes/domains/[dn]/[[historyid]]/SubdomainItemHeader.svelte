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
    import { Badge, Button, Icon, Popover, Spinner } from "@sveltestrap/sveltestrap";

    import { deleteZoneService } from "$lib/api/zone";
    import { controls as ctrlNewAlias } from "./AliasModal.svelte";
    import { controls as ctrlRecord } from '$lib/components/domains/RecordModal.svelte';
    import { controls as ctrlNewService } from "$lib/components/services/NewServicePath.svelte";
    import { controls as ctrlService } from "$lib/components/services/ServiceModal.svelte";
    import { fqdn, unreverseDomain } from "$lib/dns";
    import type { dnsRR } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceCombined } from "$lib/model/service";
    import { ZoneViewGrid } from "$lib/model/usersettings";
    import { servicesSpecs } from "$lib/stores/services";
    import { thisAliases, thisZone } from "$lib/stores/thiszone";
    import { userSession } from "$lib/stores/usersession";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        services: Array<ServiceCombined>;
        zoneId: string;
        reverseZone?: boolean;
        showResources?: boolean;
    }

    let {
        dn,
        origin,
        services,
        zoneId,
        reverseZone = false,
        showResources = $bindable(true)
    }: Props = $props();

    function isCNAME(services: Array<ServiceCombined>) {
        return services.length === 1 && services[0]._svctype === "svcs.CNAME";
    }

    function isPTR(services: Array<ServiceCombined>) {
        return services.length === 1 && services[0]._svctype === "svcs.PTR";
    }

    let deleteServiceInProgress = $state(false);
    function deleteCNAME() {
        deleteServiceInProgress = true;
        deleteZoneService(origin, zoneId, services[0]).then(
            (z) => {
                thisZone.set(z);
                deleteServiceInProgress = false;
            },
            (err) => {
                deleteServiceInProgress = false;
                throw err;
            },
        );
    }

    function showRecordModal({record, service}: {record: dnsRR; service: ServiceCombined;}) {
        ctrlRecord.Open({record, service});
    }

    function showServiceModal(service: ServiceCombined) {
        ctrlService.Open(service);
    }
</script>

<div
    class="sticky-top bg-light d-flex align-items-center mb-2 gap-2"
    style="z-index: 1"
>
    <h2
        role="button"
        tabindex="0"
        class="text-truncate"
        class:text-muted={services.length === 0 && dn != ""}
        style:cursor={(services.length || dn == "") && !isPTR(services) && !isCNAME(services) ? "pointer": "default"}
        onclick={() => (showResources = !showResources)}
        onkeypress={() => (showResources = !showResources)}
    >
        {#if services.length === 0 && dn != ""}
            <Icon name="plus-square-dotted" title="Intermediate domain with no service" />
        {:else if isPTR(services)}
            <Icon name="signpost" title="PTR" />
        {:else if isCNAME(services)}
            <Icon name="sign-turn-right" title="CNAME" />
        {:else if showResources}
            <Icon name="chevron-down" />
        {:else}
            <Icon name="chevron-right" />
        {/if}
        <span class="font-monospace" title={fqdn(dn, origin.domain)}>
            {#if reverseZone}
                {unreverseDomain(fqdn(dn, origin.domain))}
            {:else}
                {fqdn(dn, origin.domain)}
            {/if}
        </span>
    </h2>
    {#if isCNAME(services) || isPTR(services)}
        <span class="text-truncate text-muted lead">
            <Icon name="arrow-right" />
            <span class="font-monospace">
                {#if isPTR(services)}
                    {services[0].Service.ptr.Target}
                {:else}
                    {services[0].Service.cname.Target}
                {/if}
            </span>
        </span>
    {:else if !showResources && services.length}
        <Badge id={"popoversvc-" + dn.replace(".", "__")} style="cursor: pointer;">
            {$t("domains.n-services", { count: services.length })}
        </Badge>
        <Popover
            dismissible
            placement="bottom"
            target={"popoversvc-" + dn.replace(".", "__")}
        >
            {#each services as service}
                {#if $servicesSpecs && $servicesSpecs[service._svctype]}
                    <strong>{$servicesSpecs[service._svctype].name}:</strong>
                {/if}
                <span class="text-muted">{service._comment}</span>
                <br />
            {/each}
        </Popover>
    {/if}
    {#if $thisAliases[dn] && $thisAliases[dn].length != 0}
        <Badge id={"popoverbadge-" + dn.replace(".", "__")} style="cursor: pointer;">
            + {$t("domains.n-aliases", { count: $thisAliases[dn].length })}
        </Badge>
        <Popover
            dismissible
            placement="bottom"
            target={"popoverbadge-" + dn.replace(".", "__")}
            class="font-monospace"
        >
            {#each $thisAliases[dn] as alias}
                <a href={"#" + alias}>
                    {alias}
                </a>
                <br />
            {/each}
        </Popover>
    {/if}
    <div class="flex-fill"></div>
    {#if isCNAME(services) || isPTR(services)}
        <Button
            type="button"
            color="info"
            outline
            size="sm"
            title={$t("domains.edit-target")}
            on:click={() => showServiceModal(services[0])}
        >
            <Icon name="pencil" />
        </Button>
        <Button
            type="button"
            color="danger"
            disabled={deleteServiceInProgress}
            outline
            size="sm"
            title={isPTR(services) ? $t("domains.drop-pointer") : $t("domains.drop-alias")}
            on:click={deleteCNAME}
        >
            {#if deleteServiceInProgress}
                <Spinner size="sm" />
            {:else}
                <Icon name="x-circle" />
            {/if}
        </Button>
    {:else if showResources && services.length}
        <Button
            type="button"
            color="primary"
            outline
            size="sm"
            title={$t("domains.add-an-alias")}
            on:click={() => ctrlNewAlias.Open(dn)}
        >
            <Icon name="link" />
        </Button>
    {/if}
    {#if !showResources || ($userSession.settings && $userSession.settings.zoneview !== ZoneViewGrid)}
        <Button
            type="button"
            color="primary"
            size="sm"
            title={$t("domains.add-a-service")}
            on:click={() => ctrlNewService.Open(dn)}
        >
            <Icon name="plus" />
        </Button>
    {/if}
</div>
