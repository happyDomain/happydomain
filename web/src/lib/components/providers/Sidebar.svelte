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
    import { providerLinks } from "$links";
    import { resolve } from "$app/paths";
    import { Icon, Spinner } from "@sveltestrap/sveltestrap";

    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import { providers, providersSpecs, refreshProvidersSpecs } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        currentProviderId: string;
    }

    let { currentProviderId }: Props = $props();

    if (!$providersSpecs) refreshProvidersSpecs();
</script>

<nav class="provider-sidebar d-flex flex-column h-100">
    <a
        href={resolve("/providers")}
        class="sidebar-back d-flex align-items-center gap-1 mb-3 text-muted text-decoration-none fw-semibold"
    >
        <Icon name="chevron-left" />
        {$t("provider.title")}
    </a>

    {#if !$providers || !$providersSpecs}
        <div class="d-flex gap-2 align-items-center justify-content-center my-3 text-muted">
            <Spinner size="sm" color="primary" />
        </div>
    {:else}
        <ul class="list-unstyled mb-0 flex-fill overflow-auto">
            {#each $providers as provider}
                {@const isActive = provider._id === currentProviderId}
                <li class="d-flex align-items-center">
                    <a
                        href={providerLinks().provider(encodeURIComponent(provider._id))}
                        class="provider-item d-flex align-items-center gap-2 py-2 px-2 rounded text-decoration-none flex-fill {isActive
                            ? 'fw-bold text-primary active'
                            : 'text-muted'}"
                    >
                        <span class="provider-icon flex-shrink-0">
                            <ImgProvider
                                ptype={provider._srctype}
                                style="width: 1.5em; height: 1.5em; object-fit: contain;"
                            />
                        </span>
                        <span class="text-truncate flex-fill">
                            {provider._comment ||
                                ($providersSpecs[provider._srctype]?.name ?? provider._srctype)}
                        </span>
                    </a>
                    {#if isActive && $providersSpecs[provider._srctype]?.website}
                        <a
                            href={$providersSpecs[provider._srctype].website}
                            target="_blank"
                            rel="noopener noreferrer"
                            class="text-muted flex-shrink-0 px-2"
                            title={$t("provider.website")}
                            aria-label={$t("provider.website")}
                        >
                            <Icon name="box-arrow-up-right" />
                        </a>
                    {/if}
                </li>
            {/each}
        </ul>
    {/if}
</nav>

<style>
    .provider-item {
        transition: background-color 0.15s;
    }

    .provider-item:hover {
        background-color: rgba(0, 0, 0, 0.06);
    }

    .provider-item.active {
        background-color: rgba(var(--bs-primary-rgb), 0.1);
    }
</style>
