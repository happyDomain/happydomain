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
    import { Container, Icon, Table } from "@sveltestrap/sveltestrap";

    import { listProviders } from "$lib/api/provider_specs";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import TileLoader from "$lib/components/TileLoader.svelte";
    import { t } from "$lib/translations";

    const capabilities = [
        "ListDomains",
        "rr-1-A",
        "rr-257-CAA",
        "rr-61-OPENPGPKEY",
        "rr-12-PTR",
        "rr-33-SRV",
        "rr-44-SSHFP",
        "rr-52-TLSA",
    ];
</script>

<svelte:head>
    <title>{$t("menu.provider-features")} - HappyDomain</title>
</svelte:head>

<Container class="d-flex flex-column flex-fill my-5 pb-5">
    <PageTitle class="border-0" title={$t("menu.provider-features")} />

    {#await listProviders()}
        <div class="flex-fill d-flex justify-content-center align-items-center">
            <TileLoader />
        </div>
    {:then providers}
        <div class="features-wrapper">
            <Table hover>
                <caption class="visually-hidden">
                    {$t("provider.features-table-caption")}
                </caption>
                <thead>
                    <tr>
                        <th class="provider-col" scope="col">
                            <span class="visually-hidden">{$t("provider.title")}</span>
                        </th>
                        {#each capabilities as cap}
                            <th class="cap-col" scope="col">
                                {#if cap == "rr-1-A"}
                                    {$t("record.common-records")}
                                {:else if cap.startsWith("rr-")}
                                    {cap.slice(cap.lastIndexOf("-") + 1)}
                                {:else}
                                    {$t("provider.capability." + cap, { default: cap })}
                                {/if}
                            </th>
                        {/each}
                    </tr>
                </thead>
                <tbody>
                    {#each Object.keys(providers) as ptype (ptype)}
                        {@const provider = providers[ptype]}
                        <tr>
                            <th scope="row" class="provider-cell fw-normal">
                                <ImgProvider {ptype} style="max-width: 100%; max-height: 1.2em" />
                                <strong title={provider.name}>
                                    {provider.name}
                                </strong>
                            </th>
                            {#each capabilities as cap}
                                {@const supported = provider.capabilities.includes(cap)}
                                <td
                                    class="align-middle text-center"
                                    class:table-danger={!supported}
                                    class:table-success={supported}
                                >
                                    {#if supported}
                                        <Icon name="check-lg" aria-hidden="true" />
                                        <span class="visually-hidden">{$t("common.supported")}</span>
                                    {:else}
                                        <Icon name="x-lg" aria-hidden="true" />
                                        <span class="visually-hidden"
                                            >{$t("common.not-supported")}</span
                                        >
                                    {/if}
                                </td>
                            {/each}
                        </tr>
                    {/each}
                </tbody>
            </Table>
        </div>
    {/await}
</Container>

<style>
    .features-wrapper {
        overflow-x: auto;
        border-radius: 0.75rem;
        background: var(--bs-body-bg);
        box-shadow: 0 1px 8px rgba(var(--bs-black-rgb), 0.06);
        border: 1px solid var(--bs-border-color-translucent);
    }

    .features-wrapper :global(table) {
        margin-bottom: 0;
    }

    .features-wrapper :global(thead th) {
        position: sticky;
        top: 0;
        background: var(--bs-tertiary-bg);
        border-bottom: 2px solid var(--bs-border-color-translucent);
        font-size: 0.82rem;
        font-weight: 600;
        white-space: nowrap;
        text-align: center;
        vertical-align: bottom;
        padding: 0.6rem 0.5rem;
        z-index: 1;
    }

    .provider-col {
        min-width: 180px;
    }

    .cap-col {
        width: 1%;
    }

    .provider-cell {
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 220px;
    }
</style>
