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
    import { Alert, Icon } from "@sveltestrap/sveltestrap";

    import AliasModal from "$lib/components/modals/Alias.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import UserResource from "$lib/components/zones/UserResource.svelte";
    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        // "" for the apex.
        subdomain: string;
        // A past version of the zone, shown read-only.
        historical?: boolean;
    }

    let { domain, subdomain, historical = false }: Props = $props();

    let services = $derived($thisZone?.services[subdomain] ?? []);
</script>

{#if $thisZone}
    <div style="max-width: 100%;" class="w-100 pt-1 mb-5">
        <PageTitle
            title={$t("zones.viewer")}
            subtitle={$t("zones.viewer-subtitle")}
            domain={fqdn(subdomain, domain.domain)}
        >
            {#if historical}
                <span class="badge text-bg-warning">
                    <Icon name="clock-history" />
                    {$t("history.title")}
                </span>
            {/if}
        </PageTitle>
        {#key subdomain}
            <UserResource
                dn={subdomain ? subdomain : "@"}
                origin={domain}
                {services}
                zoneId={$thisZone.id}
            />
        {/key}
        {#if $thisZone.partial}
            <Alert color="info" fade={false} class="mx-2">
                <Icon name="binoculars" />
                <strong>{$t("zones.partial.title")}</strong>
                {$t("zones.partial.description")}
            </Alert>
        {/if}
    </div>
{/if}

<AliasModal origin={domain} />
