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
    import {
        Card,
        CardBody,
        CardText,
        CardTitle,
        CardSubtitle,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import ServiceBadges from "./ServiceBadges.svelte";
    import { controls as ctrlService } from "$lib/components/modals/Service.svelte";
    import type { Domain } from '$lib/model/domain';
    import type { ServiceCombined } from '$lib/model/service';
    import { servicesSpecs } from '$lib/stores/services';
    import { t } from '$lib/translations';

    interface Props {
        origin: Domain;
        service?: ServiceCombined | null;
        zoneId: string;
    }

    let { origin, service = $bindable(null), zoneId }: Props = $props();
</script>

<Card
    class="card-hover mb-3"
    style={"cursor: pointer; width: 32%; min-width: 225px;" +
          (!service ? "border-style: dashed; " : "")}
    on:click={() => ctrlService.Open(service)}
>
    {#if !$servicesSpecs}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <CardBody title={service ? $servicesSpecs[service._svctype].name : undefined}>
            <div class="d-flex justify-content-between gap-1 mb-2">
                <CardTitle class="text-truncate mb-0">
                    {#if service}
                        {$servicesSpecs[service._svctype].name}
                    {:else}
                        <Icon name="plus" /> {$t("service.new")}
                    {/if}
                </CardTitle>
                <ServiceBadges {service} />
            </div>
            <CardSubtitle class="mb-2 text-muted fst-italic">
                {#if service}
                    {$servicesSpecs[service._svctype].description}
                {:else}
                    {$t("service.new-description")}
                {/if}
            </CardSubtitle>
            {#if service && service._comment}
                <CardText style="font-size: 90%" class="text-truncate">
                    {service._comment}
                </CardText>
            {/if}
        </CardBody>
    {/if}
</Card>
