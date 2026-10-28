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

    import ServiceCard from "./ServiceCard.svelte";
    import { useZoneEditor } from "./editor";
    import { controls as ctrlNewService } from "$lib/components/services/NewServicePath.svelte";
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ServiceWithValue } from "$lib/model/service.svelte";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { t } from "$lib/translations";
    import { highlightTarget } from "$lib/utils/zone-highlight";

    interface Props {
        dn: string;
        origin: Domain;
        services: Array<ServiceWithValue>;
        zoneId: string;
    }

    let { dn, origin, services, zoneId }: Props = $props();

    const editor = useZoneEditor(() => origin);

    let readOnly = $derived(
        editor.readOnly || isReadOnlyProvider($providers_idx, origin.id_provider),
    );
</script>

<div class="d-flex flex-column gap-3 pb-3">
    {#each services as service, i (service._id ?? i)}
        <div
            id={service._id ? `svc-${service._id}` : undefined}
            use:highlightTarget={{ dn, serviceId: service._id }}
        >
            <ServiceCard {dn} {origin} {service} {zoneId} />
        </div>
    {/each}
    {#if !readOnly}
        <button
            type="button"
            class="new-service-card btn d-flex align-items-center justify-content-center gap-2 py-3 small"
            onclick={() => ctrlNewService.Open(dn)}
        >
            <Icon name="plus-square" />
            {$t("domains.add-a-service-to")}
            <span class="font-monospace">{fqdn(dn, origin.domain)}</span>
        </button>
    {/if}
</div>

<style>
    .new-service-card {
        --bs-btn-color: var(--bs-secondary-color);
        --bs-btn-border-color: var(--bs-border-color);
        --bs-btn-hover-color: var(--bs-body-color);
        --bs-btn-hover-bg: var(--bs-tertiary-bg);
        --bs-btn-hover-border-color: var(--bs-secondary-color);
        --bs-btn-active-color: var(--bs-body-color);
        --bs-btn-active-bg: var(--bs-secondary-bg);
        --bs-btn-active-border-color: var(--bs-secondary-color);
        border-style: dashed;
    }
</style>
