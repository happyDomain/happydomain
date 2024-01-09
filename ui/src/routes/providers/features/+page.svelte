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
     Container,
     Icon,
     Table,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { listProviders } from '$lib/api/provider_specs';
 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import { t } from '$lib/translations';

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

<Container class="d-flex flex-column flex-fill" fluid>
    {#await listProviders()}
        <Spinner size="lg" />
    {:then providers}
        <div
            style="overflow-x: scroll"
        >
        <Table
            hover
        >
            <thead>
                <tr>
                    <th>Fournisseurs</th>
                    {#each capabilities as cap}
                        <th class="text-center" style="white-space: nowrap;">
                            {#if cap == 'rr-1-A'}
                                {$t('record.common-records')}
                            {:else if cap.startsWith('rr-')}
                                {$t('common.records', { n: 2, type: cap.slice(cap.lastIndexOf('-')+1) })}
                            {:else}
                                {$t('provider.capability.' + cap, { default: cap })}
                            {/if}
                        </th>
                    {/each}
                </tr>
            </thead>
            <tbody>
                {#each Object.keys(providers) as ptype (ptype)}
                    {@const provider = providers[ptype]}
                    <tr>
                        <td class="text-center">
                            <ImgProvider
                                {ptype}
                                style="max-width: 100%; max-height: 2.5em"
                            /><br>
                            <strong>
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
