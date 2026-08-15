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
    import { createEventDispatcher, onMount, tick } from "svelte";

    import { Input, ListGroup, ListGroupItem, Spinner } from "@sveltestrap/sveltestrap";

    import { listProviders } from "$lib/api/provider_specs";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import type { ProviderInfos, ProviderList } from "$lib/model/provider";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        value?: string | null;
        [key: string]: unknown;
    }

    let { value = $bindable(null), ...rest }: Props = $props();
    let isLoading = $state(true);
    let providers: ProviderList = $state({});
    let filter = $state("");
    let filterInput: HTMLInputElement | undefined = $state();

    onMount(async () => {
        // When rendered inside a Modal, its content is portalled into
        // document.body after mount (so the native `autofocus` attribute
        // fires too early), and the Modal itself steals focus back to its
        // wrapper once its fade transition ends. Retry focusing once the
        // DOM has settled, and again after the transition would have ended.
        await tick();
        filterInput?.focus();
        setTimeout(() => filterInput?.focus(), 350);
    });

    listProviders().then((res) => {
        providers = res;
        isLoading = false;
    });

    function selectProvider(provider: ProviderInfos, ptype: string) {
        value = ptype;
        dispatch("provider-selected", { provider, ptype });
    }

    let filteredPtypes = $derived(
        Object.keys(providers).filter((ptype) => {
            if (!filter) return true;
            const needle = filter.toLowerCase();
            const provider = providers[ptype];
            return (
                provider.name.toLowerCase().includes(needle) ||
                provider.description.toLowerCase().includes(needle) ||
                (provider.website ?? "").toLowerCase().includes(needle)
            );
        }),
    );
</script>

<Input
    type="search"
    autofocus
    class="mb-2"
    placeholder={$t("common.filter")}
    bind:value={filter}
    bind:inner={filterInput}
/>
<ListGroup {...rest}>
    {#if isLoading}
        <ListGroupItem class="d-flex justify-content-center align-items-center gap-2">
            <Spinner color="primary" />
            {$t("wait.retrieving-provider")}
        </ListGroupItem>
    {/if}
    {#each filteredPtypes as ptype (ptype)}
        {@const provider = providers[ptype]}
        <ListGroupItem
            active={value === ptype}
            tag="button"
            class="d-flex ps-1 gap-1 py-1"
            on:click={() => selectProvider(provider, ptype)}
        >
            <div class="align-self-center text-center" style="min-width:50px;width:50px;">
                <ImgProvider {ptype} />
            </div>
            <div class="align-self-center" style="line-height: 1.1">
                <strong>{provider.name}</strong> &ndash;
                <small class="text-muted" title={provider.description}>{provider.description}</small
                >
            </div>
        </ListGroupItem>
    {/each}
</ListGroup>
<p class="mt-3 fw-bold">
    {$t("onboarding.connect.noprovider")}
    <a
        href="https://github.com/happyDomain/happydomain/issues/new"
        target="_blank"
        data-umami-event="need-another-provider"
    >
        {$t("onboarding.connect.noproviderTellUs")}
    </a>.
</p>
