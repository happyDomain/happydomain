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
    import { Alert, Badge, Icon } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import SubdomainItem from "$lib/components/zones/SubdomainItem.svelte";
    import SubdomainList from "$lib/components/zones/SubdomainList.svelte";
    import UserResource from "$lib/components/zones/UserResource.svelte";
    import {
        countServices,
        isGlobalTemplate,
        templateOrigin,
        type ZoneTemplate,
    } from "$lib/model/zone_template";
    import { sortedDomains, sortedDomainsWithIntermediate, thisZone } from "$lib/stores/thiszone";
    import { thisTemplate } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";

    interface Props {
        template: ZoneTemplate;
        // Nothing can be changed, as a global template shown to a user.
        readOnly: boolean;
    }

    let { template: loaded, readOnly }: Props = $props();

    let template = $derived($thisTemplate?.id === loaded.id ? $thisTemplate : loaded);
    const origin = templateOrigin();
</script>

<div style="max-width: 100%;" class="w-100 pt-1 mb-5">
    <PageTitle title={template.name} subtitle={template.description || undefined}>
        <div class="d-flex flex-wrap align-items-center justify-content-end gap-2">
            {#if isGlobalTemplate(template)}
                <Badge color="info" title={$t("zone-templates.global-hint")}>
                    <Icon name="globe" />
                    {$t("zone-templates.global")}
                </Badge>
            {/if}
            <Badge color="secondary">
                {$t("zone-templates.services", { count: countServices(template) })}
            </Badge>
        </div>
    </PageTitle>

    {#if readOnly && isGlobalTemplate(template)}
        <Alert color="info" fade={false} class="mx-2">
            <Icon name="globe" />
            {$t("zone-templates.global-hint")}
        </Alert>
    {/if}

    {#if $thisZone && $sortedDomains}
        <SubdomainList
            subdomains={!$sortedDomainsWithIntermediate || $sortedDomains.length == 0
                ? [""]
                : $sortedDomainsWithIntermediate}
        >
            {#snippet subdomain(dn, services)}
                <SubdomainItem {dn} {origin} {services}>
                    <UserResource dn={dn ? dn : "@"} {origin} {services} zoneId={template.id} />
                </SubdomainItem>
            {/snippet}
        </SubdomainList>
    {/if}
</div>
