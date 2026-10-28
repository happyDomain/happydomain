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
    import { Button, Icon } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import { controls as ctrlNewSubdomain } from "$lib/components/zones/NewSubdomainPath.svelte";
    import SubdomainListTiny from "$lib/components/zones/SubdomainListTiny.svelte";
    import { useZoneEditor } from "$lib/components/zones/editor";
    import { isReverseZone } from "$lib/dns";
    import { checksBySubdomain } from "$lib/stores/domainchecks";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { sortedDomains, sortedDomainsWithIntermediate, thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import HealthDot from "./HealthDot.svelte";

    interface Props {
        origin: Domain;
        selectedHistory: string | undefined;
    }

    let { origin, selectedHistory }: Props = $props();

    const editor = useZoneEditor(() => origin);

    let readOnly = $derived(isReadOnlyProvider($providers_idx, origin.id_provider));

    let subdomains = $derived(
        isReverseZone(origin.domain) ? $sortedDomains : $sortedDomainsWithIntermediate,
    );

    let checks = $derived(
        $checksBySubdomain?.domainId === origin.id ? $checksBySubdomain.sections : undefined,
    );
</script>

<!-- Flush right, as the rows: its end lines up with their health dots. -->
<div
    class="d-flex align-items-center gap-1 pt-3 pb-1 small fw-semibold text-uppercase text-body-secondary"
>
    <span class="flex-fill text-truncate">
        {$t("domains.sidebar.subdomains")}
    </span>
    <!-- A disabled button gets no pointer events: its wrapper carries the tooltip.
         As wide as a health dot, it holds the button centered over them. -->
    <span
        class="position-relative flex-shrink-0 p-1"
        title={readOnly ? $t("domains.read-only-edit") : $t("domains.add-a-subdomain")}
    >
        <Button
            type="button"
            color="link"
            size="sm"
            class="position-absolute top-50 start-50 translate-middle text-body-secondary p-0 lh-1"
            aria-label={$t("domains.add-a-subdomain")}
            disabled={!$sortedDomains || readOnly}
            on:click={() => ctrlNewSubdomain.Open()}
        >
            <Icon name="plus-lg" />
        </Button>
    </span>
</div>
{#if subdomains && $thisZone && $thisZone.id == selectedHistory}
    {@const zoneId = $thisZone.id}
    <div class="min-h-0 overflow-y-auto">
        <!-- The health of each subdomain, leading to its page in the zone. -->
        <SubdomainListTiny
            domains={subdomains}
            {origin}
            href={(dn) => editor.subdomainHref(zoneId, dn)}
        >
            {#snippet end(dn)}
                <HealthDot section={checks?.get(dn)} />
            {/snippet}
        </SubdomainListTiny>
    </div>
{/if}
