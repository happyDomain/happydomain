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
    import { goto } from "$app/navigation";

    import {
        Button,
        Card,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import ProviderList from "$lib/components/providers/List.svelte";
    import type { Provider } from "$lib/model/provider";
    import { appConfig } from "$lib/stores/config";
    import {
        providers,
        providersSpecs,
    } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    export let filteredProvider: Provider | null = null
</script>

<Card {...$$restProps}>
    <div class="card-header d-flex justify-content-between">
        {$t("provider.title")}
        {#if !$appConfig.disable_providers}
            <Button size="sm" color="light" href="/providers/new">
                <Icon name="plus" />
            </Button>
        {/if}
    </div>
    {#if !$providers || !$providersSpecs}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <ProviderList
            flush
            items={$providers}
            noLabel
            bind:selectedProvider={filteredProvider}
            on:new-provider={() => goto("/providers/new")}
        />
    {/if}
</Card>
