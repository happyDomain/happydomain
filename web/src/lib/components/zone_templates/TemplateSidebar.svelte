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
    import { templateLinks } from "$links";

    import { Button, Icon } from "@sveltestrap/sveltestrap";

    import { controls as ctrlNewSubdomain } from "$lib/components/zones/NewSubdomainPath.svelte";
    import ServiceSidebar from "$lib/components/zones/ServiceSidebar.svelte";
    import SubdomainListTiny from "$lib/components/zones/SubdomainListTiny.svelte";
    import { templateOrigin, type ZoneTemplate } from "$lib/model/zone_template";
    import { sortedDomains, sortedDomainsWithIntermediate } from "$lib/stores/thiszone";
    import { zoneTemplates } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";
    import SelectTemplate from "./SelectTemplate.svelte";

    interface Props {
        template: ZoneTemplate;
        // Nothing can be changed, as a global template shown to a user.
        readOnly: boolean;
        // The service being edited, when on its page.
        service?: { subdomain: string; serviceid: string };
        onselect: (id: string) => void;
    }

    let { template, readOnly, service, onselect }: Props = $props();

    const origin = templateOrigin();
</script>

<!-- Header: back to the template + template selector -->
<div class="d-flex">
    {#if service}
        <Button
            href={templateLinks().template(encodeURIComponent(template.id))}
            class="fw-bolder"
            color="link"
            title={$t("zones.return-to")}
        >
            <Icon name="chevron-left" />
        </Button>
    {/if}
    <SelectTemplate templates={$zoneTemplates} current={template} {onselect} />
</div>

{#if service}
    <ServiceSidebar
        {origin}
        subdomain={service.subdomain}
        serviceid={service.serviceid}
        historyId={template.id}
    />
{:else}
    <div class="d-flex gap-2 pb-2 sticky-top" style="padding-top: 10px">
        <!-- A disabled button gets no pointer events: its wrapper carries the tooltip. -->
        <span
            class="d-flex flex-fill"
            style="min-width: 0"
            title={readOnly ? $t("zone-templates.global-hint") : undefined}
        >
            <Button
                type="button"
                color="secondary"
                outline
                size="sm"
                class="flex-fill text-truncate"
                disabled={!$sortedDomains || readOnly}
                on:click={() => ctrlNewSubdomain.Open()}
            >
                <Icon name="server" />
                {$t("domains.add-a-subdomain")}
            </Button>
        </span>
    </div>
    <div style="min-height:0; overflow-y: auto;">
        {#if $sortedDomains && $sortedDomainsWithIntermediate}
            <SubdomainListTiny
                domains={$sortedDomains.length ? $sortedDomainsWithIntermediate : [""]}
                {origin}
            />
        {/if}
    </div>
{/if}
