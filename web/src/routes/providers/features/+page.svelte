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
    import { Container, Icon, Spinner, Table } from "@sveltestrap/sveltestrap";

    import { listProviders } from "$lib/api/provider_specs";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
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
    <title>{$t("menu.provider-features")} - happyDomain</title>
</svelte:head>

<Container class="d-flex flex-column flex-fill py-3">
    <PageTitle title={$t("menu.provider-features")} />

    {#await listProviders()}
        <div class="flex-fill d-flex justify-content-center align-items-center">
            <Spinner size="lg" />
        </div>
    {:then providers}
        <div class="features-wrapper">
            <Table hover>
                <thead>
                    <tr>
                        <th class="provider-col"></th>
                        {#each capabilities as cap}
                            <th class="cap-col">
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
                            <td class="provider-cell">
                                <ImgProvider {ptype} style="max-width: 100%; max-height: 1.2em" />
                                <strong title={provider.name}>
                                    {provider.name}
                                </strong>
                            </td>
                            {#each capabilities as cap}
                                <td
                                    class="align-middle text-center"
                                    class:table-danger={!provider.capabilities.includes(cap)}
                                    class:table-success={provider.capabilities.includes(cap)}
                                >
                                    {#if provider.capabilities.includes(cap)}
                                        <Icon name="check-lg" />
                                    {:else}
                                        <Icon name="x-lg" />
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
        background: #fff;
        box-shadow: 0 1px 8px rgba(0, 0, 0, 0.06);
        border: 1px solid rgba(0, 0, 0, 0.06);
    }

    .features-wrapper :global(table) {
        margin-bottom: 0;
    }

    .features-wrapper :global(thead th) {
        position: sticky;
        top: 0;
        background: #f8f9fa;
        border-bottom: 2px solid rgba(0, 0, 0, 0.08);
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
